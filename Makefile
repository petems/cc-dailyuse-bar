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

# Versioning
VERSION ?= $(shell git describe --tags --always --dirty || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD || echo none)
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ" || echo unknown)

# Build flags
LDFLAGS_FLAGS=-s -w -X cc-dailyuse-bar/src/cmd.Version=$(VERSION) -X cc-dailyuse-bar/src/cmd.Commit=$(COMMIT) -X cc-dailyuse-bar/src/cmd.Date=$(DATE)
LDFLAGS=-ldflags "$(LDFLAGS_FLAGS)"
LDFLAGS_GUI=-ldflags "$(LDFLAGS_FLAGS) -H windowsgui"
BUILD_FLAGS=-v

.PHONY: all build clean test coverage coverage-html coverage-func deps lint fmt vet help run install \
	install-service-macos uninstall-service-macos bundle-macos

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
	$(GOCMD) fmt ./...

# Run go vet
vet:
	$(GOCMD) vet ./...

# Run the application
run:
	$(GOCMD) run ./src run

# Run as daemon (background process)
daemon:
	$(GOCMD) run ./src run --daemon

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

# macOS LaunchAgent variables
LAUNCHAGENT_LABEL=com.cc-dailyuse-bar
LAUNCHAGENT_PLIST=$(HOME)/Library/LaunchAgents/$(LAUNCHAGENT_LABEL).plist
LAUNCHAGENT_LOG_DIR=$(HOME)/Library/Logs/cc-dailyuse-bar
MACOS_BIN_DIR?=$(HOME)/.local/bin

# Install macOS LaunchAgent (run at login)
install-service-macos: build
	@echo "Installing cc-dailyuse-bar as macOS LaunchAgent..."
	mkdir -p $(HOME)/Library/LaunchAgents
	mkdir -p $(LAUNCHAGENT_LOG_DIR)
	mkdir -p "$(MACOS_BIN_DIR)"
	cp "$(BINARY_NAME)" "$(MACOS_BIN_DIR)/$(BINARY_NAME)"
	sed -e 's|__HOME__|$(HOME)|g' \
	    -e 's|__CC_DAILYUSE_BAR_BIN__|$(MACOS_BIN_DIR)/$(BINARY_NAME)|g' \
	    com.cc-dailyuse-bar.plist > "$(LAUNCHAGENT_PLIST)"
	launchctl load $(LAUNCHAGENT_PLIST)
	@echo "LaunchAgent installed and loaded."
	@echo "Logs: $(LAUNCHAGENT_LOG_DIR)/"
	@echo "To stop: make uninstall-service-macos"

# Uninstall macOS LaunchAgent
uninstall-service-macos:
	-launchctl unload $(LAUNCHAGENT_PLIST)
	rm -f $(LAUNCHAGENT_PLIST)
	rm -f "$(MACOS_BIN_DIR)/$(BINARY_NAME)"
	@echo "LaunchAgent uninstalled."
	@echo "Logs preserved at: $(LAUNCHAGENT_LOG_DIR)/"

# macOS .app bundle variables
APP_NAME=CC Daily Use Bar
APP_BUNDLE=$(APP_NAME).app

# Build macOS .app bundle
bundle-macos: build
	@echo "Verifying binary is a macOS Mach-O executable..."
	@file $(BINARY_NAME) | grep -q Mach-O || { echo "Error: $(BINARY_NAME) is not a macOS binary. Run on macOS or cross-compile for darwin."; exit 1; }
	@echo "Creating macOS .app bundle..."
	rm -rf "$(APP_BUNDLE)"
	mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	mkdir -p "$(APP_BUNDLE)/Contents/Resources"
	cp $(BINARY_NAME) "$(APP_BUNDLE)/Contents/MacOS/"
	cp packaging/macos/Info.plist "$(APP_BUNDLE)/Contents/"
	@echo "Bundle created: $(APP_BUNDLE)"
	@echo "To sign: codesign --deep --force --options=runtime --entitlements=packaging/macos/entitlements.plist --sign 'Developer ID Application: YOUR_NAME' '$(APP_BUNDLE)'"

# Run formatters
format:
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
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.10.1; \
	fi
	@if command -v pre-commit >/dev/null 2>&1; then \
		echo "Installing pre-commit hooks..."; \
		pre-commit install; \
	else \
		echo "pre-commit not found. Install it with: pip install pre-commit (or brew install pre-commit)"; \
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
	@echo "  vet          - Run go vet"
	@echo "  run          - Run the application"
	@echo "  daemon       - Run as daemon (background process)"
	@echo "  install      - Install the binary"
	@echo "  install-service - Install as systemd service (Linux)"
	@echo "  uninstall-service - Remove systemd service (Linux)"
	@echo "  install-service-macos - Install as macOS LaunchAgent"
	@echo "  uninstall-service-macos - Remove macOS LaunchAgent"
	@echo "  bundle-macos   - Build macOS .app bundle"
	@echo "  format       - Format code (fmt + lint-fix)"
	@echo "  security     - Check for security vulnerabilities"
	@echo "  mocks        - Generate mocks"
	@echo "  check        - Run lint, test, and build"
	@echo "  dev-setup    - Set up development environment"
	@echo "  ci           - CI pipeline (deps, lint, test, build)"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  help         - Show this help message"
