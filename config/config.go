package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds all configuration for the pg-backup application
type Config struct {
	Schedule          string `koanf:"schedule" yaml:"schedule"`
	PostgresDatabase  string `koanf:"postgres_database" yaml:"postgres_database"`
	PostgresUser      string `koanf:"postgres_user" yaml:"postgres_user"`
	PostgresPassword  string `koanf:"postgres_password" yaml:"postgres_password"`
	PostgresExtraOpts string `koanf:"postgres_extra_opts" yaml:"postgres_extra_opts"`
}

// LoadConfig loads configuration from YAML file and environment variables
// Priority: defaults < YAML file < environment variables
func LoadConfig(configPath string) (*Config, error) {
	k := koanf.New(".")

	// Set defaults
	defaults := map[string]interface{}{
		"schedule":            "@daily",
		"postgres_database":   "dbname",
		"postgres_user":       "user",
		"postgres_password":   "password",
		"postgres_extra_opts": "--schema=public --blobs",
	}

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("error loading defaults: %w", err)
	}

	// Load from YAML file if it exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
				return nil, fmt.Errorf("error loading config file: %w", err)
			}
		}
	}

	// Load from environment variables (highest priority)
	// Environment variables should be in uppercase (e.g., SCHEDULE, POSTGRES_DATABASE)
	if err := k.Load(env.Provider("", ".", func(s string) string {
		// Convert environment variable names to lowercase with underscores
		// POSTGRES_DATABASE -> postgres_database
		return strings.ToLower(s)
	}), nil); err != nil {
		return nil, fmt.Errorf("error loading environment variables: %w", err)
	}

	// Unmarshal into Config struct
	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	return &cfg, nil
}
