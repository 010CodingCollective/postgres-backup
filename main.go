package main

import (
	"log/slog"
	"os"
	"pg-backup/config"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting postgres_back application")
	slog.Info("Reading configuration")
	// Or without YAML file (uses defaults + env vars)
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		slog.Error("error loading config", "error", err)
		os.Exit(1)
	}

	slog.Debug("Configuration loaded", "data", cfg)
	backupper, err := newPostgresBackup(cfg)
	if err != nil {
		slog.Error("error initializing backup service", "error", err)
		os.Exit(1)
	}
	err = backupper.Start()
	if err != nil {
		slog.Error("error while backupping", "error", err)
		os.Exit(1)
	}
	slog.Info("Closing application")
}
