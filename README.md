# postgres-backup

A lightweight, automated PostgreSQL backup service that creates compressed backups and uploads them to S3-compatible storage.

## Features

- **Scheduled Backups**: Configurable cron-style scheduling using `robfig/cron`
- **Compression**: Automatic gzip compression to save storage space
- **S3 Integration**: Upload backups to AWS S3 or S3-compatible storage (MinIO, etc.)
- **Retry Logic**: Automatic retry on S3 upload failures
- **Local Cleanup**: Removes local backup files after successful S3 upload
- **Flexible Configuration**: Configure via YAML file or environment variables
- **Docker Support**: Ready-to-use Docker container

## Quick Start

### Using Docker

```bash
docker run -v $(pwd)/config.yaml:/app/config.yaml \
  ghcr.io/010codingcollective/postgres-backup:latest
```

### Using Docker Compose

```yaml
version: '3.8'
services:
  postgres-backup:
    image: ghcr.io/010codingcollective/postgres-backup:latest
    volumes:
      - ./config.yaml:/app/config.yaml
    environment:
      - POSTGRES_PASSWORD=your_password
      - S3_SECRET_ACCESS_KEY=your_secret_key
```

## Configuration

Configuration can be provided via:
1. YAML file (`config.yaml`)
2. Environment variables (highest priority)

### Example Configuration

```yaml
schedule: "@daily"  # Cron format: @daily, @hourly, or "0 2 * * *"
run_at_startup: false  # Run backup immediately on startup
postgres_database: mydb
postgres_user: postgres
postgres_password: secret
postgres_host: localhost
postgres_port: "5432"
postgres_extra_opts: "--schema=public --blobs"

s3:
  endpoint: ""  # Leave empty for AWS S3, or set for MinIO/S3-compatible
  region: us-east-1
  bucket: my-backup-bucket
  access_key_id: AKIAIOSFODNN7EXAMPLE
  secret_access_key: wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
  use_path_style: false  # Set to true for MinIO and most S3-compatible services
```

### Environment Variables

All configuration can be overridden with environment variables:

```bash
SCHEDULE="@daily"
RUN_AT_STARTUP="false"
POSTGRES_DATABASE="mydb"
POSTGRES_USER="postgres"
POSTGRES_PASSWORD="secret"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5432"
POSTGRES_EXTRA_OPTS="--schema=public --blobs"
S3_ENDPOINT=""
S3_REGION="us-east-1"
S3_BUCKET="my-backup-bucket"
S3_ACCESS_KEY_ID="AKIAIOSFODNN7EXAMPLE"
S3_SECRET_ACCESS_KEY="wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
S3_USE_PATH_STYLE="false"
```

## Schedule Format

The `schedule` field uses cron format. Common examples:

- `@daily` - Run once per day at midnight
- `@hourly` - Run once per hour
- `@every 6h` - Run every 6 hours
- `0 2 * * *` - Run at 2:00 AM daily
- `0 */6 * * *` - Run every 6 hours
- `0 2 * * 0` - Run at 2:00 AM every Sunday

## Backup Storage

Backups are stored in S3 with the following structure:

```
s3://your-bucket/backups/{database}-backup-{timestamp}.sql.gz
```

Example: `backups/mydb-backup-20250128-020000.sql.gz`

## Building from Source

### Prerequisites

- Go 1.23+
- PostgreSQL client tools (`pg_dump`)

### Build

```bash
go build -o pg-backup .
```

### Run

```bash
./pg-backup
```

## Building Docker Image

```bash
docker build -t postgres-backup .
```

Or using buildah:

```bash
buildah bud -f Dockerfile -t postgres-backup .
```

## How It Works

1. **Scheduler** starts based on configured cron schedule
2. **pg_dump** creates a PostgreSQL backup
3. **Compression** compresses the backup with gzip
4. **S3 Upload** uploads the compressed backup to S3 (with retry)
5. **Cleanup** removes the local backup file after successful upload

## Error Handling

- If `pg_dump` fails, the backup job is aborted
- If S3 upload fails, it retries once
- If both S3 upload attempts fail, the local backup file is retained and logged
- If S3 is misconfigured, the application fails at startup

## License

MIT License - see [LICENSE](LICENSE) file for details

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.
