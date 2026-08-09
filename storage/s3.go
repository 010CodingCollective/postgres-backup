package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Config holds configuration for S3 and S3-compatible storage services
type S3Config struct {
	Endpoint        string // Custom endpoint for S3-compatible services (e.g., "https://minio.example.com")
	Region          string // AWS region or any string for S3-compatible services
	Bucket          string // S3 bucket name
	AccessKeyID     string // Access key ID
	SecretAccessKey string // Secret access key
	UsePathStyle    bool   // Set to true for most S3-compatible services (MinIO, etc.)
	StorageClass    string // Storage class for uploaded objects (e.g., "GLACIER"); empty uses the provider default
}

// S3Client wraps the AWS S3 client for uploading backups
type S3Client struct {
	client       *s3.Client
	bucket       string
	storageClass types.StorageClass
}

// normalizeStorageClass upper-cases a configured storage class and checks the SDK
// recognises it. Empty is valid and means "let the provider pick its default".
func normalizeStorageClass(s string) (types.StorageClass, error) {
	sc := types.StorageClass(strings.ToUpper(strings.TrimSpace(s)))
	if sc == "" {
		return "", nil
	}
	if !slices.Contains(sc.Values(), sc) {
		return "", fmt.Errorf("unknown storage class %q, expected one of %v", s, sc.Values())
	}
	return sc, nil
}

// NewS3Client creates a new S3 client that works with AWS S3 and S3-compatible services
func NewS3Client(ctx context.Context, cfg S3Config) (*S3Client, error) {
	// Fail at startup on an invalid storage class rather than at the first upload
	storageClass, err := normalizeStorageClass(cfg.StorageClass)
	if err != nil {
		return nil, err
	}

	// Create custom credentials provider
	creds := credentials.NewStaticCredentialsProvider(
		cfg.AccessKeyID,
		cfg.SecretAccessKey,
		"", // session token (not needed for most cases)
	)

	// Build AWS config options
	opts := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(creds),
	}

	// Load the SDK configuration
	awsCfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with optional custom endpoint
	s3Opts := []func(*s3.Options){}

	// Use custom endpoint for S3-compatible services
	if cfg.Endpoint != "" {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		})
	}

	// Use path-style addressing (required for most S3-compatible services)
	if cfg.UsePathStyle {
		s3Opts = append(s3Opts, func(o *s3.Options) {
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, s3Opts...)

	return &S3Client{
		client:       client,
		bucket:       cfg.Bucket,
		storageClass: storageClass,
	}, nil
}

// UploadFile uploads a file to S3
func (c *S3Client) UploadFile(ctx context.Context, key string, file *os.File) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		Body:         file,
		StorageClass: c.storageClass,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}
	return nil
}

// UploadReader uploads from an io.Reader to S3
func (c *S3Client) UploadReader(ctx context.Context, key string, reader io.Reader) error {
	_, err := c.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:       aws.String(c.bucket),
		Key:          aws.String(key),
		Body:         reader,
		StorageClass: c.storageClass,
	})
	if err != nil {
		return fmt.Errorf("failed to upload to S3: %w", err)
	}
	return nil
}

// DownloadFile downloads a file from S3
func (c *S3Client) DownloadFile(ctx context.Context, key string) (io.ReadCloser, error) {
	output, err := c.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}
	return output.Body, nil
}

// DeleteFile deletes a file from S3
func (c *S3Client) DeleteFile(ctx context.Context, key string) error {
	_, err := c.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}
	return nil
}

// ListFiles lists files in the S3 bucket with an optional prefix
func (c *S3Client) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(c.bucket),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	paginator := s3.NewListObjectsV2Paginator(c.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list files from S3: %w", err)
		}
		for _, obj := range page.Contents {
			files = append(files, *obj.Key)
		}
	}

	return files, nil
}

// FileExists checks if a file exists in the S3 bucket
func (c *S3Client) FileExists(ctx context.Context, key string) (bool, error) {
	_, err := c.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a "not found" error
		return false, nil
	}
	return true, nil
}

// GetBucket returns the configured bucket name
func (c *S3Client) GetBucket() string {
	return c.bucket
}

// GetStorageClass returns the normalized storage class sent with uploads,
// or an empty string when the provider default is used
func (c *S3Client) GetStorageClass() string {
	return string(c.storageClass)
}
