package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"pg-backup/config"

	"github.com/robfig/cron/v3"
)

type PostgresBackup struct {
	cfg *config.Config
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

	slog.Info("Backup job completed")
}

func newPostgresBackup(cfg *config.Config) *PostgresBackup {
	return &PostgresBackup{cfg: cfg}
}
