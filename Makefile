# Valhalla Makefile

# Build variables
BINARY_NAME=valhalla
VERSION?=dev
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE?=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt

# Directories
BUILD_DIR=bin
DIST_DIR=dist
DOCS_DIR=docs

# OS and Architecture detection
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)

.PHONY: all build clean test deps fmt lint install help

# Default target
all: clean deps fmt test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "Built $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: clean deps
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 .
	
	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 .
	
	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	
	# macOS AMD64
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 .
	
	# macOS ARM64 (Apple Silicon)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 .
	
	@echo "Multi-platform build complete in $(DIST_DIR)/"

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

# Lint code (requires golangci-lint)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

# Install binary to $GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html

# Run the application with development settings
run: build
	./$(BUILD_DIR)/$(BINARY_NAME) --help

# Run with example discovery
run-example: build
	./$(BUILD_DIR)/$(BINARY_NAME) discover --provider vmware --dry-run

# Generate documentation
docs:
	@echo "Generating documentation..."
	@mkdir -p $(DOCS_DIR)/cli
	./$(BUILD_DIR)/$(BINARY_NAME) --help > $(DOCS_DIR)/cli/help.txt
	@echo "Documentation generated in $(DOCS_DIR)/"

# Development setup
dev-setup:
	@echo "Setting up development environment..."
	# Install development tools
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/air-verse/air@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	@echo "Development tools installed"

# Live reload during development (requires air)
dev:
	@which air > /dev/null || (echo "Installing air..." && go install github.com/air-verse/air@latest)
	air

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t valhalla:$(VERSION) .

# Docker run
docker-run: docker-build
	docker run --rm -it valhalla:$(VERSION) --help

# Create release archives
release: build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)/archives
	
	# Linux AMD64
	tar -czf $(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-linux-amd64 README.md LICENSE
	
	# Linux ARM64
	tar -czf $(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-linux-arm64 README.md LICENSE
	
	# Windows AMD64
	zip -j $(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe README.md LICENSE
	
	# macOS AMD64
	tar -czf $(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-darwin-amd64 README.md LICENSE
	
	# macOS ARM64
	tar -czf $(DIST_DIR)/archives/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz -C $(DIST_DIR) $(BINARY_NAME)-darwin-arm64 README.md LICENSE
	
	@echo "Release archives created in $(DIST_DIR)/archives/"

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	@which govulncheck > /dev/null || (echo "Installing govulncheck..." && go install golang.org/x/vuln/cmd/govulncheck@latest)
	govulncheck ./...

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Show help
help:
	@echo "Valhalla - Hypervisor Infrastructure Discovery and IaC Generation Tool"
	@echo ""
	@echo "Available targets:"
	@echo "  build        Build the binary for current platform"
	@echo "  build-all    Build for all supported platforms"
	@echo "  deps         Download and tidy dependencies"
	@echo "  test         Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  fmt          Format code"
	@echo "  lint         Run linter"
	@echo "  install      Install binary to GOPATH/bin"
	@echo "  clean        Clean build artifacts"
	@echo "  run          Build and run with --help"
	@echo "  run-example  Run example discovery command"
	@echo "  docs         Generate documentation"
	@echo "  dev-setup    Setup development environment"
	@echo "  dev          Run with live reload (requires air)"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Build and run Docker container"
	@echo "  release      Create release archives"
	@echo "  security     Check for security vulnerabilities"
	@echo "  bench        Run benchmark tests"
	@echo "  help         Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION      Version string (default: dev)"
	@echo "  GOOS         Target OS (default: current OS)"
	@echo "  GOARCH       Target architecture (default: current arch)"
	@echo ""
	@echo "Examples:"
	@echo "  make build VERSION=1.0.0"
	@echo "  make build-all VERSION=1.0.0"
	@echo "  make build GOOS=linux GOARCH=amd64"
	@echo ""
	@echo "Prerequisites:"
	@echo "  Go 1.18+ is required"
