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

# Expose ports (if needed for future HTTP/gRPC API)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD forge status || exit 1

# Default command
ENTRYPOINT ["forge"]
CMD ["--help"]

