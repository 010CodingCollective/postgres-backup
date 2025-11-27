package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"pg-backup/config"

	"github.com/robfig/cron/v3"
)

type PostgresBackup struct {
	cfg     *config.Config
	tempDir string
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

	slog.Info("Backup job completed", "file", outfile)
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
	outfile := fmt.Sprintf("%s/%s-backup-%s.sql", b.tempDir, b.cfg.PostgresDatabase, ts)
	f, err := os.Create(outfile)
	if err != nil {
		slog.Error("Failed to create dump file", "file", outfile, "error", err)
		return "", err
	}
	defer f.Close()

	cmd.Stdout = f
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Redact sensitive data in logs
	logArgs := strings.Join(args, " ")
	slog.Info("Executing pg_dump", "args", logArgs, "output", outfile)

	if err = cmd.Run(); err != nil {
		// Remove partial file on failure
		_ = os.Remove(outfile)
		err = fmt.Errorf("pg_dump failed %w", err)
		return "", err
	}
	return outfile, nil
}

func newPostgresBackup(cfg *config.Config) *PostgresBackup {
	return &PostgresBackup{
		cfg:     cfg,
		tempDir: "/tmp",
	}
}
