package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"pg-backup/config"
	"pg-backup/storage"
)

func TestBuildObjectKey(t *testing.T) {
	const filename = "mydb-backup-20250128-020000.sql.gz"

	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{name: "default prefix", prefix: "backups", want: "backups/" + filename},
		{name: "empty prefix uploads to bucket root", prefix: "", want: filename},
		{name: "whitespace only prefix uploads to bucket root", prefix: "   ", want: filename},
		{name: "trailing slash is ignored", prefix: "backups/", want: "backups/" + filename},
		{name: "leading slash is ignored", prefix: "/backups", want: "backups/" + filename},
		{name: "surrounding slashes are ignored", prefix: "/backups/", want: "backups/" + filename},
		{name: "only slashes uploads to bucket root", prefix: "///", want: filename},
		{name: "nested prefix is preserved", prefix: "prod/mydb", want: "prod/mydb/" + filename},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildObjectKey(tt.prefix, filename); got != tt.want {
				t.Errorf("buildObjectKey(%q, %q) = %q, want %q", tt.prefix, filename, got, tt.want)
			}
		})
	}
}

// TestUploadToS3UsesConfiguredPrefix covers the wiring from config through to the
// request path, so a prefix that never reaches PutObject is caught.
func TestUploadToS3UsesConfiguredPrefix(t *testing.T) {
	tests := []struct {
		name     string
		prefix   string
		wantPath string
	}{
		{name: "default prefix", prefix: "backups", wantPath: "/test-bucket/backups/mydb-backup.sql.gz"},
		{name: "nested prefix", prefix: "prod/mydb", wantPath: "/test-bucket/prod/mydb/mydb-backup.sql.gz"},
		{name: "empty prefix uploads to bucket root", prefix: "", wantPath: "/test-bucket/mydb-backup.sql.gz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
			}))
			defer srv.Close()

			localPath := filepath.Join(t.TempDir(), "mydb-backup.sql.gz")
			if err := os.WriteFile(localPath, []byte("dump"), 0o600); err != nil {
				t.Fatalf("failed to write test backup file: %v", err)
			}

			client, err := storage.NewS3Client(context.Background(), storage.S3Config{
				Endpoint:        srv.URL,
				Region:          "nl-ams",
				Bucket:          "test-bucket",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				UsePathStyle:    true,
			})
			if err != nil {
				t.Fatalf("NewS3Client returned unexpected error: %v", err)
			}

			backup := &PostgresBackup{
				cfg:      &config.Config{S3: config.S3Config{Bucket: "test-bucket", Prefix: tt.prefix}},
				s3Client: client,
			}

			if err := backup.uploadToS3(localPath); err != nil {
				t.Fatalf("uploadToS3 returned unexpected error: %v", err)
			}
			if gotPath != tt.wantPath {
				t.Errorf("PUT path = %q, want %q", gotPath, tt.wantPath)
			}
		})
	}
}
