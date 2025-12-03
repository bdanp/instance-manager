# Makefile for AWS Instance Manager

.PHONY: build test clean install run help

# Variables
BINARY_NAME=instance-manager
BUILD_DIR=./bin
CMD_DIR=./cmd

# Default target
all: test build

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)/main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@go mod download

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -cover ./...

# Run tests with coverage report
test-coverage-html:
	@echo "Running tests with HTML coverage report..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	@go install $(CMD_DIR)/main.go

# Run the application (example usage)
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME) --help

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Run security scan
sec:
	@echo "Running security scan..."
	@gosec ./...

# Create example SSH key for testing
create-test-key:
	@echo "Creating test SSH key..."
	@mkdir -p ./test-keys
	@ssh-keygen -t rsa -b 2048 -f ./test-keys/test_key -N "" -C "test@example.com"
	@echo "Test key created: ./test-keys/test_key.pub"

# Run integration tests (requires AWS credentials)
test-integration:
	@echo "Running integration tests..."
	@go test -tags=integration ./test/...

# Show help
help:
	@echo "Available targets:"
	@echo "  build              - Build the application"
	@echo "  test               - Run unit tests"
	@echo "  test-coverage      - Run tests with coverage"
	@echo "  test-coverage-html - Generate HTML coverage report"
	@echo "  clean              - Clean build artifacts"
	@echo "  install            - Install binary to GOPATH/bin"
	@echo "  run                - Build and run the application"
	@echo "  fmt                - Format code"
	@echo "  lint               - Run linter"
	@echo "  sec                - Run security scan"
	@echo "  deps               - Install dependencies"
	@echo "  create-test-key    - Create test SSH key"
	@echo "  test-integration   - Run integration tests"
	@echo "  help               - Show this help message"
