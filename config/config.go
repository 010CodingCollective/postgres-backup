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

// S3Config holds configuration for S3 and S3-compatible storage services
type S3Config struct {
	Endpoint        string `koanf:"endpoint" yaml:"endpoint"`                   // Custom endpoint for S3-compatible services (e.g., "https://minio.example.com")
	Region          string `koanf:"region" yaml:"region"`                       // AWS region or any string for S3-compatible services
	Bucket          string `koanf:"bucket" yaml:"bucket"`                       // S3 bucket name
	AccessKeyID     string `koanf:"access_key_id" yaml:"access_key_id"`         // Access key ID
	SecretAccessKey string `koanf:"secret_access_key" yaml:"secret_access_key"` // Secret access key
	UsePathStyle    bool   `koanf:"use_path_style" yaml:"use_path_style"`       // Set to true for most S3-compatible services (MinIO, etc.)
	StorageClass    string `koanf:"storage_class" yaml:"storage_class"`         // Storage class for uploaded objects (e.g. "GLACIER"); empty uses the provider default
	Prefix          string `koanf:"prefix" yaml:"prefix"`                       // Key prefix for uploaded backups; empty places them at the root of the bucket
}

// Config holds all configuration for the pg-backup application
type Config struct {
	Schedule          string `koanf:"schedule" yaml:"schedule"`
	PostgresDatabase  string `koanf:"postgres_database" yaml:"postgres_database"`
	PostgresUser      string `koanf:"postgres_user" yaml:"postgres_user"`
	PostgresPassword  string `koanf:"postgres_password" yaml:"postgres_password"`
	PostgresHost      string `koanf:"postgres_host" yaml:"postgres_host"`
	PostgresPort      string `koanf:"postgres_port" yaml:"postgres_port"`
	PostgresExtraOpts string `koanf:"postgres_extra_opts" yaml:"postgres_extra_opts"`
	// RunAtStartup triggers an immediate backup run when the application starts
	RunAtStartup bool     `koanf:"run_at_startup" yaml:"run_at_startup"`
	S3           S3Config `koanf:"s3" yaml:"s3"`
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
		"postgres_host":       "localhost",
		"postgres_port":       "5432",
		"postgres_extra_opts": "--schema=public --blobs",
		"run_at_startup":      false,
		"s3.prefix":           "backups",
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
	// For nested structs, use underscore as delimiter (e.g., S3_SECRET_ACCESS_KEY -> s3.secret_access_key)
	if err := k.Load(env.Provider("", ".", func(s string) string {
		// Convert to lowercase
		s = strings.ToLower(s)
		// Replace first underscore after "s3" with dot for nested struct mapping
		// s3_secret_access_key -> s3.secret_access_key
		if strings.HasPrefix(s, "s3_") {
			return strings.Replace(s, "s3_", "s3.", 1)
		}
		return s
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
