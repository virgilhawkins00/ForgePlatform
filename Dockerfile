# Build stage - compile with CGO for SQLite support
FROM golang:1.24-alpine AS builder

# Install build dependencies for CGO
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files first for caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with CGO enabled for SQLite (architecture determined by docker platform)
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o forge ./cmd/forge

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite-libs

# Create non-root user
RUN adduser -D -g '' forge

# Create directories
RUN mkdir -p /home/forge/.forge/data /home/forge/.forge/plugins /home/forge/.forge/logs \
    && chown -R forge:forge /home/forge/.forge

# Copy binary from builder
COPY --from=builder /app/forge /usr/local/bin/forge
RUN chmod +x /usr/local/bin/forge

# Switch to non-root user
USER forge

# Set working directory
WORKDIR /home/forge

# Expose ports for HTTP API and metrics
EXPOSE 8080 9090

# Health check using HTTP liveness probe (PORT env var is set by Cloud Run)
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:${PORT:-8080}/health/liveness || exit 1

# =============================================================================
# Environment Variables
# =============================================================================
# These are default values. Override them at runtime using:
#   - docker run -e FORGE_VAR=value
#   - docker run --env-file .env
#   - Cloud Run environment variables
#   - Kubernetes ConfigMaps/Secrets
#
# SECURITY: Never bake secrets into the image!
# Use runtime environment variables or secret managers.
# =============================================================================

# Core configuration
ENV FORGE_DATA_DIR=/home/forge/.forge/data
ENV FORGE_LOG_LEVEL=info
ENV FORGE_HTTP_PORT=8080

# Database configuration
ENV FORGE_DB_MAX_CONNECTIONS=10
ENV FORGE_DB_CACHE_SIZE=64000

# GCP configuration (set at runtime, not in image)
# ENV FORGE_GCP_PROJECT_ID=
# ENV FORGE_GCP_CREDENTIALS_PATH=
ENV FORGE_GCP_REGION=southamerica-east1
ENV FORGE_GCP_METRIC_PREFIX=custom.googleapis.com/forge
ENV FORGE_GCP_FLUSH_INTERVAL=60
ENV FORGE_GCP_BATCH_SIZE=200

# AI configuration
ENV FORGE_OLLAMA_URL=http://localhost:11434
ENV FORGE_AI_MODEL=llama3.2

# Development (never enable in production!)
ENV FORGE_DEBUG=false
ENV FORGE_PROFILING_ENABLED=false

# Default command - start daemon (runs in foreground by default)
ENTRYPOINT ["forge"]
CMD ["start"]
