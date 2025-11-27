package main

import (
	"log/slog"
	"os"
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
	// TODO: Implement actual backup logic using pg_dump
	slog.Info("Backup job completed")
}

func newPostgresBackup(cfg *config.Config) *PostgresBackup {
	return &PostgresBackup{cfg: cfg}
}
