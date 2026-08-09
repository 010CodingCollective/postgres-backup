package storage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNormalizeStorageClass(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty means provider default", input: "", want: ""},
		{name: "whitespace only means provider default", input: "   ", want: ""},
		{name: "canonical value passes through", input: "GLACIER", want: "GLACIER"},
		{name: "lowercase is upper-cased", input: "glacier", want: "GLACIER"},
		{name: "surrounding whitespace is trimmed", input: "  GLACIER  ", want: "GLACIER"},
		{name: "scaleway one zone class is accepted", input: "ONEZONE_IA", want: "ONEZONE_IA"},
		{name: "typo is rejected", input: "GLACIAR", wantErr: true},
		{name: "internal whitespace is rejected", input: "GLACIER IR", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeStorageClass(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("normalizeStorageClass(%q) = %q, want an error", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeStorageClass(%q) returned unexpected error: %v", tt.input, err)
			}
			if string(got) != tt.want {
				t.Errorf("normalizeStorageClass(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestUploadSetsStorageClassHeader checks the wire format: an unset storage class
// must omit x-amz-storage-class entirely so providers apply their own default.
func TestUploadSetsStorageClassHeader(t *testing.T) {
	tests := []struct {
		name       string
		configured string
		wantHeader string
	}{
		{name: "unset omits the header", configured: "", wantHeader: ""},
		{name: "configured value is sent normalized", configured: "glacier", wantHeader: "GLACIER"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("x-amz-storage-class")
			}))
			defer srv.Close()

			client, err := NewS3Client(context.Background(), S3Config{
				Endpoint:        srv.URL,
				Region:          "nl-ams",
				Bucket:          "test-bucket",
				AccessKeyID:     "key",
				SecretAccessKey: "secret",
				UsePathStyle:    true,
				StorageClass:    tt.configured,
			})
			if err != nil {
				t.Fatalf("NewS3Client returned unexpected error: %v", err)
			}

			if err := client.UploadReader(context.Background(), "backup.sql.gz", strings.NewReader("dump")); err != nil {
				t.Fatalf("UploadReader returned unexpected error: %v", err)
			}

			if got != tt.wantHeader {
				t.Errorf("x-amz-storage-class = %q, want %q", got, tt.wantHeader)
			}
		})
	}
}

// TestFileExists checks that only a genuine 404 is reported as absent. Any other
// failure must surface as an error rather than being mistaken for a missing object.
func TestFileExists(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		want    bool
		wantErr bool
	}{
		{name: "object present", status: http.StatusOK, want: true},
		{name: "object absent", status: http.StatusNotFound, want: false},
		{name: "access denied is an error", status: http.StatusForbidden, want: false, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()

			client, err := NewS3Client(context.Background(), S3Config{
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

			got, err := client.FileExists(context.Background(), "backup.sql.gz")
			if tt.wantErr && err == nil {
				t.Fatalf("FileExists on HTTP %d returned no error, want one", tt.status)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("FileExists on HTTP %d returned unexpected error: %v", tt.status, err)
			}
			if got != tt.want {
				t.Errorf("FileExists on HTTP %d = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
