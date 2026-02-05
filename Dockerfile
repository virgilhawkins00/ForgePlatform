# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /forge ./cmd/forge

# Runtime stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata sqlite

# Create non-root user
RUN adduser -D -g '' forge

# Create directories
RUN mkdir -p /home/forge/.forge/data /home/forge/.forge/plugins /home/forge/.forge/logs \
    && chown -R forge:forge /home/forge/.forge

# Copy binary from builder
COPY --from=builder /forge /usr/local/bin/forge

# Switch to non-root user
USER forge

# Set working directory
WORKDIR /home/forge

# Expose ports for HTTP API and metrics
EXPOSE 8080 9090

# Health check using liveness probe
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD forge health liveness || exit 1

# Environment variables
ENV FORGE_DATA_DIR=/home/forge/.forge/data
ENV FORGE_LOG_LEVEL=info

# Default command - start in foreground mode
ENTRYPOINT ["forge"]
CMD ["start", "--foreground"]
