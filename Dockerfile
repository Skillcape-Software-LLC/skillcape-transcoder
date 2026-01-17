# Build stage
FROM golang:1.22-bookworm AS builder

WORKDIR /app

# Install build dependencies (sqlite required for mattn/go-sqlite3)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Copy go mod files
COPY go.mod go.sum* ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -o server ./cmd/server

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ffmpeg ca-certificates tzdata

# Create non-root user
RUN adduser -D -g '' appuser

# Create directories
RUN mkdir -p /data /config && chown -R appuser:appuser /data /config

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server /usr/local/bin/server

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set default environment variables
ENV PORT=8080 \
    WORKER_COUNT=2 \
    TEMP_DIR=/data \
    WEBHOOK_RETRY_COUNT=3

# Run the application
CMD ["server"]
