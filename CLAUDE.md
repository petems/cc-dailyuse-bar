# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

cc-dailyuse-bar is a Go-based system tray application that monitors daily Claude Code usage via the `ccusage` binary. It displays real-time cost information with color-coded status indicators (🟢 Green, 🟡 Yellow, 🔴 Red) based on configurable thresholds.

## Architecture

The application follows a clean architecture pattern with clear separation of concerns:

### Core Components

- **main.go**: Application entry point with system tray integration using `getlantern/systray`
- **models/**: Data models and business logic
  - `Config`: Configuration management with validation
  - `UsageState`: Current usage state with status calculation
  - `AlertStatus`: Status enumeration (Green/Yellow/Red/Gray)
  - `TemplateData`: Template data structures for display formatting
- **services/**: Business logic layer
  - `ConfigService`: XDG-compliant configuration management
  - `UsageService`: Integration with `ccusage` binary, polling, and state management
- **lib/**: Utilities and shared functionality
  - `Logger`: Structured logging with configurable levels
  - `template_engine`: Template processing for display formats
  - `errors`: Custom error types and wrapping

### Key Design Patterns

- **Service Layer**: Business logic is encapsulated in services with clear interfaces
- **Dependency Injection**: Services are injected at initialization
- **Error Wrapping**: Custom error types with categorization (ErrCodeCCUsage, ErrCodeValidation, etc.)
- **Polling Architecture**: Configurable polling intervals with retry logic
- **XDG Compliance**: Configuration stored in standard XDG directories
- **Internationalization**: Built-in English/Japanese localization support

## Common Development Commands

### Building and Running
```bash
make build          # Build the binary
make run            # Run the application directly
make daemon         # Run as background daemon
```

### Testing
```bash
make test           # Run all tests
make test-race      # Run tests with race detection
make coverage       # Generate coverage report (HTML and terminal)
make coverage-func  # Show coverage by function
make bench          # Run benchmarks
```

#### Test Structure
- **Unit tests**: `src/*/test.go` - Individual component tests
- **Contract tests**: `tests/contract/` - Service interface contracts  
- **Integration tests**: `tests/integration/` - End-to-end workflows
- **Test categories**: `tests/unit/` - Additional unit tests

### Code Quality
```bash
make lint           # Run golangci-lint
make lint-fix       # Run linter with auto-fix
make fmt            # Format Go code
make vet            # Run go vet
make format         # Run fmt + lint-fix together
```

### Dependencies
```bash
make deps           # Download and tidy dependencies
make deps-update    # Update dependencies
```

### Development Setup
```bash
make dev-setup      # Install golangci-lint and setup environment
make clean          # Clean build artifacts
```

## Configuration

The application uses XDG-compliant configuration:
- **Linux/macOS**: `~/.config/cc-dailyuse-bar/config.yaml`
- **Windows**: `%APPDATA%/cc-dailyuse-bar/config.yaml`

Key configuration fields:
- `ccusage_path`: Path to ccusage binary (default: "ccusage")
- `update_interval`: Polling interval in seconds (10-300, default: 30)
- `yellow_threshold`/`red_threshold`: Cost thresholds for status colors
- `debug_level`: Logging level (DEBUG, INFO, WARN, ERROR, FATAL)

## Integration with ccusage

The application depends on the `ccusage` binary being installed and accessible. It expects JSON output from `ccusage daily --json` with the structure:
```json
{
  "daily": [{"date": "2023-XX-XX", "totalTokens": X, "totalCost": X.XX}],
  "totals": {"totalTokens": X, "totalCost": X.XX}
}
```

When `ccusage` is unavailable, the application falls back to simulated data for development.

## Error Handling

The application uses structured error handling with custom error types:
- `CCUsageError`: Issues with ccusage binary interaction
- `ValidationError`: Configuration or input validation failures
- `ConfigError`: Configuration file issues

All errors are logged with structured context using the built-in logger.

## Testing Strategy

- **Mocking**: Uses testify/mock for service layer testing
- **Fixtures**: Test configuration files in XDG-compliant paths
- **Integration**: Full workflow tests including ccusage simulation
- **Coverage**: Comprehensive test coverage across all layers
- **Race Detection**: All tests run with race detection in CI

## Performance Considerations

- **Caching**: UsageService caches ccusage results for 10 seconds to avoid excessive calls
- **Polling**: Configurable polling intervals with sensible defaults
- **Resource Management**: Proper cleanup of goroutines and tickers
- **Error Resilience**: Retry logic for ccusage failures with exponential backoff