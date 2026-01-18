# Hoster - Modern Deployment Marketplace
# Build, test, and run commands

.PHONY: all build test test-unit test-integration test-e2e test-e2e-short test-all coverage run clean help

# Default target
all: test build

# Build the binary
build:
	@echo "Building hoster..."
	go build -o bin/hoster ./cmd/hoster

# Run all tests
test: test-unit test-integration

# Run unit tests (core/ - pure functions, no I/O)
test-unit:
	@echo "Running unit tests..."
	go test -v -race ./internal/core/...

# Run integration tests (shell/ - Docker, DB, API)
test-integration:
	@echo "Running integration tests..."
	go test -v -race ./internal/shell/...

# Run end-to-end tests (full suite, requires Docker)
test-e2e:
	@echo "Running E2E tests (requires Docker)..."
	go test -v -timeout 10m ./tests/e2e/...

# Run end-to-end smoke tests only (faster)
test-e2e-short:
	@echo "Running E2E smoke tests..."
	go test -v -short -timeout 5m ./tests/e2e/...

# Run all tests (unit + integration + e2e)
test-all: test-unit test-integration test-e2e

# Generate coverage report (core/ must be >90%)
coverage:
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./internal/core/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run the server
run: build
	./bin/hoster

# Run in development mode with auto-reload
dev:
	go run ./cmd/hoster

# Clean build artifacts
clean:
	rm -rf bin/ coverage.out coverage.html

# Download dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code
vet:
	go vet ./...

# Help
help:
	@echo "Hoster Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build            - Build the binary"
	@echo "  make test             - Run unit + integration tests"
	@echo "  make test-unit        - Run unit tests (core/)"
	@echo "  make test-integration - Run integration tests (shell/)"
	@echo "  make test-e2e         - Run E2E tests (requires Docker)"
	@echo "  make test-e2e-short   - Run E2E smoke tests only"
	@echo "  make test-all         - Run all tests (unit + integration + e2e)"
	@echo "  make coverage         - Generate coverage report"
	@echo "  make run              - Build and run the server"
	@echo "  make dev              - Run in development mode"
	@echo "  make clean            - Clean build artifacts"
	@echo "  make deps             - Download dependencies"
	@echo "  make fmt              - Format code"
	@echo "  make vet              - Vet code"
