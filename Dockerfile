# Multi-stage build for pg-backup
# Runtime image includes pg_dump so the app can perform backups.

# ---------- Build stage ----------
FROM golang:1.25-alpine AS builder
WORKDIR /src

# Install build deps
RUN apk add --no-cache git ca-certificates

# Cache modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/pg-backup ./

# ---------- Runtime stage ----------
FROM alpine:3.20

WORKDIR /app

# pg_dump is required at runtime; also install certificates and tzdata
RUN apk add --no-cache postgresql-client ca-certificates tzdata

# Copy app binary
COPY --from=builder /out/pg-backup /usr/local/bin/pg-backup

# Provide a default config (can be overridden by mounting a config.yaml)
COPY config.yaml.example /app/config.yaml

# Expose a volume for optional persistence or mounting configs
VOLUME ["/app"]

# Use non-root user for safety
RUN adduser -D -u 10001 appuser && chown -R appuser:appuser /app
USER appuser

# Logs to stdout/stderr by default; the app reads /app/config.yaml
ENTRYPOINT ["/usr/local/bin/pg-backup"]
