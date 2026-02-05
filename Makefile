# Forge Platform Makefile

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/forge-platform/forge/internal/adapters/cli.Version=$(VERSION) \
	-X github.com/forge-platform/forge/internal/adapters/cli.Commit=$(COMMIT) \
	-X github.com/forge-platform/forge/internal/adapters/cli.BuildDate=$(BUILD_DATE)"

# Go variables
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := gofmt
GOLINT := golangci-lint

# Binary name
BINARY_NAME := forge
BINARY_PATH := bin/$(BINARY_NAME)

# Directories
CMD_DIR := ./cmd/forge
INTERNAL_DIR := ./internal
PKG_DIR := ./pkg

.PHONY: all build clean test lint fmt deps run install help

# Default target
all: deps lint test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) $(CMD_DIR)
	@echo "Built $(BINARY_PATH)"

# Build for all platforms
build-all: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)

build-darwin:
	@echo "Building for macOS..."
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)

build-windows:
	@echo "Building for Windows..."
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out
	$(GOMOD) tidy

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage report
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run linter
lint:
	@echo "Running linter..."
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run the application
run: build
	./$(BINARY_PATH)

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installed to $(GOPATH)/bin/$(BINARY_NAME)"

# Initialize Forge (create config directory)
init: build
	./$(BINARY_PATH) init

# Start the daemon
start: build
	./$(BINARY_PATH) start

# Open the TUI
ui: build
	./$(BINARY_PATH) ui

# Generate protobuf files (for gRPC)
proto:
	@echo "Generating protobuf files..."
	@if [ -d "api/proto" ]; then \
		protoc --go_out=. --go-grpc_out=. api/proto/*.proto; \
	fi

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t forge-platform:$(VERSION) .

# Docker run
docker-run:
	docker run -it --rm forge-platform:$(VERSION)

# Help
help:
	@echo "Forge Platform - Build Commands"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          - Run deps, lint, test, and build"
	@echo "  build        - Build the binary"
	@echo "  build-all    - Build for all platforms"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  deps         - Download dependencies"
	@echo "  run          - Build and run"
	@echo "  install      - Install to GOPATH/bin"
	@echo "  init         - Initialize Forge configuration"
	@echo "  start        - Start the daemon"
	@echo "  ui           - Open the TUI"
	@echo "  docker-build - Build Docker image"
	@echo "  help         - Show this help"

