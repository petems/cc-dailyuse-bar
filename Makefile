# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=cc-dailyuse-bar
BINARY_UNIX=$(BINARY_NAME)_unix

# Linting
GOLANGCI_LINT=golangci-lint
LINT_TIMEOUT=5m

# Build flags
LDFLAGS=-ldflags "-s -w"
LDFLAGS_GUI=-ldflags "-s -w -H windowsgui"
BUILD_FLAGS=-v

.PHONY: all build clean test coverage coverage-html coverage-func deps lint fmt vet help run install gofumpt

# Default target
all: clean deps lint test build

# Build the binary (console)
build:
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) -v ./src

# Build for Windows (console)
build-windows:
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME).exe -v ./src

# Build console version (for debugging)
build-console:
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME)-console -v ./src

# Build for Linux
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_UNIX) -v ./src

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run tests
test:
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# Show coverage percentage
coverage-func:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -func=coverage.out

# Generate HTML coverage report
coverage-html:
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	$(GOTEST) -v -race ./...

# Run benchmarks
bench:
	$(GOTEST) -v -bench=. ./...

# Download dependencies
deps:
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
deps-update:
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Run linter
lint:
	$(GOLANGCI_LINT) run --timeout=$(LINT_TIMEOUT)

# Run linter with fix
lint-fix:
	$(GOLANGCI_LINT) run --timeout=$(LINT_TIMEOUT) --fix

# Format code
fmt:
	@command -v gofumpt >/dev/null 2>&1 || { echo "Installing gofumpt..."; $(GOCMD) install mvdan.cc/gofumpt@latest; }
	gofumpt -l -w .
	$(GOCMD) fmt ./...

# Format code with gofumpt only (write changes)
gofumpt:
	@command -v gofumpt >/dev/null 2>&1 || { echo "Installing gofumpt..."; $(GOCMD) install mvdan.cc/gofumpt@latest; }
	gofumpt -l -w .

# Run go vet
vet:
	$(GOCMD) vet ./...

# Run the application
run:
	$(GOCMD) run ./src

# Run as daemon (background process)
daemon:
	$(GOCMD) run ./src --daemon

# Install the binary
install:
	$(GOCMD) install ./src

# Install systemd service
install-service: build
	sudo cp $(BINARY_NAME) /usr/local/bin/
	sudo cp cc-dailyuse-bar.service /etc/systemd/system/
	sudo systemctl daemon-reload
	sudo systemctl enable cc-dailyuse-bar
	@echo "Service installed. Start with: sudo systemctl start cc-dailyuse-bar"

# Uninstall systemd service
uninstall-service:
	sudo systemctl stop cc-dailyuse-bar || true
	sudo systemctl disable cc-dailyuse-bar || true
	sudo rm -f /etc/systemd/system/cc-dailyuse-bar.service
	sudo rm -f /usr/local/bin/cc-dailyuse-bar
	sudo systemctl daemon-reload
	@echo "Service uninstalled"

# Run formatters
format:
	@command -v gofumpt >/dev/null 2>&1 || { echo "Installing gofumpt..."; $(GOCMD) install mvdan.cc/gofumpt@latest; }
	gofumpt -l -w .
	$(GOCMD) fmt ./...
	$(GOLANGCI_LINT) run --timeout=$(LINT_TIMEOUT) --fix

# Check for security vulnerabilities
security:
	$(GOCMD) list -json -deps ./... | nancy sleuth

# Generate mocks (if using mockgen)
mocks:
	@echo "Generating mocks..."
	@find . -name "*.go" -exec grep -l "//go:generate mockgen" {} \; | xargs -I {} sh -c 'cd $$(dirname {}) && go generate'

# Run all checks (lint, test, build)
check: lint test build

# Development setup
dev-setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v1.54.2; \
	fi

# CI pipeline
ci: deps lint test build

# Docker build (if you have a Dockerfile)
docker-build:
	docker build -t $(BINARY_NAME) .

# Docker run (if you have a Dockerfile)
docker-run:
	docker run --rm -p 8080:8080 $(BINARY_NAME)

# Show help
help:
	@echo "Available targets:"
	@echo "  all          - Clean, deps, lint, test, and build"
	@echo "  build        - Build the binary"
	@echo "  build-linux  - Build for Linux"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  coverage     - Run tests with coverage report"
	@echo "  coverage-func- Show coverage percentage by function"
	@echo "  coverage-html- Generate HTML coverage report"
	@echo "  test-race    - Run tests with race detection"
	@echo "  bench        - Run benchmarks"
	@echo "  deps         - Download and tidy dependencies"
	@echo "  deps-update  - Update dependencies"
	@echo "  lint         - Run linter"
	@echo "  lint-fix     - Run linter with auto-fix"
	@echo "  fmt          - Format Go code"
	@echo "  gofumpt      - Format Go code with gofumpt (writes changes)"
	@echo "  vet          - Run go vet"
	@echo "  run          - Run the application"
	@echo "  daemon       - Run as daemon (background process)"
	@echo "  install      - Install the binary"
	@echo "  install-service - Install as systemd service"
	@echo "  uninstall-service - Remove systemd service"
	@echo "  format       - Format code (fmt + lint-fix)"
	@echo "  security     - Check for security vulnerabilities"
	@echo "  mocks        - Generate mocks"
	@echo "  check        - Run lint, test, and build"
	@echo "  dev-setup    - Set up development environment"
	@echo "  ci           - CI pipeline (deps, lint, test, build)"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help message"
