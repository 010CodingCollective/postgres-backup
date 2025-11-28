package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pg-backup/config"
	"pg-backup/storage"

	"github.com/robfig/cron/v3"
)

type PostgresBackup struct {
	cfg      *config.Config
	tempDir  string
	s3Client *storage.S3Client
}

func (b *PostgresBackup) Start() error {
	slog.Info("Initializing backup scheduler", "schedule", b.cfg.Schedule)

	// Check for required dependencies
	if _, err := exec.LookPath("pg_dump"); err != nil {
		return fmt.Errorf("required command 'pg_dump' not found: %w", err)
	}

	// Create a new cron scheduler
	c := cron.New()

	// Add the backup job using the configured schedule
	_, err := c.AddFunc(b.cfg.Schedule, b.runBackup)
	if err != nil {
		return err
	}

	// Optionally run a backup immediately on startup
	if b.cfg.RunAtStartup {
		slog.Info("run_at_startup is enabled: triggering immediate backup run")
		go b.runBackup()
	}

	// Start the scheduler
	c.Start()
	slog.Info("Backup scheduler started")

	// Set up graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received
	sig := <-quit
	slog.Info("Received shutdown signal", "signal", sig)

	// Stop the scheduler gracefully
	ctx := c.Stop()
	<-ctx.Done()
	slog.Info("Backup scheduler stopped gracefully")

	return nil
}

func (b *PostgresBackup) runBackup() {
	slog.Info("Starting backup job")
	// Build pg_dump arguments from config
	outfile, err := b.doPostgresBackup()
	if err != nil {
		slog.Error("Error running backup job", "error", err)
		return
	}

	slog.Info("Backup file created", "file", outfile)

	// Upload to S3 if configured
	if b.s3Client != nil {
		if err := b.uploadToS3(outfile); err != nil {
			slog.Error("Failed to upload backup to S3", "error", err, "file", outfile)
			slog.Info("Local backup file retained due to S3 upload failure", "file", outfile)
			return
		}

		// Delete local file after successful S3 upload
		if err := os.Remove(outfile); err != nil {
			slog.Error("Failed to delete local backup file after S3 upload", "error", err, "file", outfile)
		} else {
			slog.Info("Local backup file deleted after successful S3 upload", "file", outfile)
		}
	}

	slog.Info("Backup job completed successfully")
}

// uploadToS3 uploads a backup file to S3 with one retry on failure
func (b *PostgresBackup) uploadToS3(localPath string) error {
	// Generate S3 key: backups/{database}-backup-{timestamp}.sql.gz
	filename := filepath.Base(localPath)
	s3Key := fmt.Sprintf("backups/%s", filename)

	ctx := context.Background()

	// Open the file for upload
	f, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file for S3 upload: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			slog.Error("Failed to close backup file after S3 upload", "error", err)
		}
	}(f)

	slog.Info("Uploading backup to S3", "key", s3Key, "bucket", b.s3Client.GetBucket())

	// First attempt
	err = b.s3Client.UploadFile(ctx, s3Key, f)
	if err != nil {
		slog.Warn("S3 upload failed, retrying once", "error", err)

		// Reset file pointer for retry
		_, seekErr := f.Seek(0, 0)
		if seekErr != nil {
			return fmt.Errorf("failed to reset file pointer for retry: %w", seekErr)
		}

		// Retry attempt
		err = b.s3Client.UploadFile(ctx, s3Key, f)
		if err != nil {
			return fmt.Errorf("S3 upload failed after retry: %w", err)
		}
	}

	slog.Info("Successfully uploaded backup to S3", "key", s3Key, "bucket", b.s3Client.GetBucket())
	return nil
}

// doPostgresBackup performs a backup of the PostgreSQL database using the pg_dump command and returns the output file path.
func (b *PostgresBackup) doPostgresBackup() (string, error) {
	var args []string
	if b.cfg.PostgresHost != "" {
		args = append(args, "-h", b.cfg.PostgresHost)
	}
	if b.cfg.PostgresPort != "" {
		args = append(args, "-p", b.cfg.PostgresPort)
	}
	if b.cfg.PostgresUser != "" {
		args = append(args, "-U", b.cfg.PostgresUser)
	}
	if strings.TrimSpace(b.cfg.PostgresExtraOpts) != "" {
		// Split extra options by whitespace (expects already shell-safe flags)
		args = append(args, strings.Fields(b.cfg.PostgresExtraOpts)...)
	}
	if b.cfg.PostgresDatabase != "" {
		args = append(args, b.cfg.PostgresDatabase)
	}

	// Prepare command
	cmd := exec.Command("pg_dump", args...)
	// Pass password via environment variable
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", b.cfg.PostgresPassword))

	// Dump to a timestamped file in the temp directory
	ts := time.Now().Format("20060102-150405")
	outfile := fmt.Sprintf("%s/%s-backup-%s.sql.gz", b.tempDir, b.cfg.PostgresDatabase, ts)
	f, err := os.Create(outfile)
	if err != nil {
		slog.Error("Failed to create dump file", "file", outfile, "error", err)
		return "", err
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			slog.Error("Failed to close dump file", "file", outfile, "error", err)
		}
	}(f)

	// Create gzip writer for compression
	gzipWriter := gzip.NewWriter(f)
	defer func(gzipWriter *gzip.Writer) {
		err = gzipWriter.Close()
		if err != nil {
			slog.Error("Failed to close gzip writer", "file", outfile, "error", err)
		}
	}(gzipWriter)

	cmd.Stdout = gzipWriter
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Redact sensitive data in logs
	logArgs := strings.Join(args, " ")
	slog.Info("Executing pg_dump with compression", "args", logArgs, "output", outfile)

	if err = cmd.Run(); err != nil {
		// Remove partial file on failure
		_ = os.Remove(outfile)
		slog.Error("Failed to execute pg_dump with compression", "error", err, "output", stderr)
		err = fmt.Errorf("pg_dump failed %w", err)
		return "", err
	}
	return outfile, nil
}

func newPostgresBackup(cfg *config.Config) (*PostgresBackup, error) {
	pb := &PostgresBackup{
		cfg:     cfg,
		tempDir: "/tmp",
	}

	// Initialize S3 client if S3 is configured
	if cfg.S3.Bucket != "" {
		s3Cfg := storage.S3Config{
			Endpoint:        cfg.S3.Endpoint,
			Region:          cfg.S3.Region,
			Bucket:          cfg.S3.Bucket,
			AccessKeyID:     cfg.S3.AccessKeyID,
			SecretAccessKey: cfg.S3.SecretAccessKey,
			UsePathStyle:    cfg.S3.UsePathStyle,
		}

		s3Client, err := storage.NewS3Client(context.Background(), s3Cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
		}
		pb.s3Client = s3Client
		slog.Info("S3 client initialized", "bucket", cfg.S3.Bucket)
	}

	return pb, nil
}
