# Hoster Dockerfile
# Multi-stage build for production deployment

# =============================================================================
# Stage 1: Build
# =============================================================================
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /app

# Copy go mod files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build minion binaries for remote node deployment
RUN GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" \
    -o internal/shell/docker/binaries/minion-linux-amd64 ./cmd/hoster-minion
RUN GOOS=linux GOARCH=arm64 go build -ldflags "-s -w" \
    -o internal/shell/docker/binaries/minion-linux-arm64 ./cmd/hoster-minion

# Build main hoster binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-s -w" -o hoster ./cmd/hoster

# =============================================================================
# Stage 2: Runtime
# =============================================================================
FROM alpine:3.19

# Install runtime dependencies
RUN apk --no-cache add ca-certificates wget docker-cli

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/hoster .

# Create data directory
RUN mkdir -p /data

# Expose ports
# 8080 - API server
# 9091 - App proxy
EXPOSE 8080 9091

# Default environment
ENV HOSTER_SERVER_PORT=8080
ENV HOSTER_DATA_DIR=/data
ENV HOSTER_APP_PROXY_ADDRESS=0.0.0.0:9091

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=10s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Run hoster
CMD ["./hoster"]
