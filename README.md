# cc-dailyuse-bar

![Screenshot](docs/screenshot.png)

A system tray application that monitors your daily Claude Code usage and displays real-time cost information in your menu bar. The project was inspired by [sivchari/ccowl](https://github.com/sivchari/ccowl), but optimised for my usecase: for enterprise plans where spend is tracked per day rather than the five-hour windows Claude Pro/Max.

## Features

- **Real-time monitoring**: Displays current daily usage cost and API call count
- **Daily spend focus**: Tailored for enterprise accounts that care about day-level usage, not five-hour Pro/Max windows
- **Status indicators**: Color-coded status (🟢 Green, 🟡 Yellow, 🔴 Red) based on configurable thresholds
- **System tray integration**: Runs in the background with menu bar access
- **Automatic updates**: Configurable polling interval with resilient polling service
- **XDG compliance**: Stores configuration in standard XDG directories
- **Multi-language support**: English and Japanese localization
- **Smart caching**: Avoids hitting `ccusage` more often than necessary and surfaces health issues clearly


The application displays in your system tray as:
- `CC 🟢 $0.45` - Normal usage (below yellow threshold)
- `CC 🟡 $12.50` - High usage (above yellow threshold)
- `CC 🔴 $25.00` - Critical usage (above red threshold)
- `CC 🟢 $0.00` - No usage data for today (ccusage works but no data)
- `CC ⚪️ Unknown` - ccusage unavailable or error state

## Installation

### Prerequisites

- Go 1.21 or later
- `ccusage` binary installed and accessible in PATH
- macOS, Linux, or Windows

### Ubuntu/Linux Additional Requirements

For Ubuntu/Linux systems, install the required dependencies:

```bash
sudo apt update
sudo apt install -y libayatana-appindicator3-dev pkg-config
```

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd cc-dailyuse-bar

# Install dependencies
make deps

# Build the application
make build

./cc-dailyuse-bar
```

## Configuration

The application uses XDG-compliant configuration storage:

- **Linux/macOS**: `~/.config/cc-dailyuse-bar/config.yaml`
- **Windows**: `%APPDATA%/cc-dailyuse-bar/config.yaml`

### Default Configuration

```yaml
ccusage_path: "ccusage"
update_interval: 30
yellow_threshold: 10.00
red_threshold: 20.00
debug_level: "INFO"
cache_window: 10
cmd_timeout: 5
```

### Configuration Options

- `ccusage_path`: Path to the ccusage binary (default: "ccusage")
- `update_interval`: Polling interval in seconds (10-300, default: 30)
- `yellow_threshold`: Cost threshold for yellow warning (default: $10.00)
- `red_threshold`: Cost threshold for red alert (default: $20.00)
- `debug_level`: Logging level - DEBUG, INFO, WARN, ERROR, or FATAL (default: "INFO")
- `cache_window`: Number of seconds to reuse a cached ccusage response when it reports healthy data (default: 10)
- `cmd_timeout`: Number of seconds before a ccusage command run is aborted (default: 5)

## Usage

### Running the Application

```bash
# Run directly
make run

# Run as daemon (background process)
make daemon

# Or build and run
make build
./cc-dailyuse-bar

# Run as daemon with command line flag
./cc-dailyuse-bar --daemon
```

### System Tray Menu

Right-click the tray icon to access:
- **Usage Information**: Daily cost, API calls, last update time
- **Settings**: View current configuration
- **Quit**: Exit the application

### Status Indicators

- 🟢 **Green**: Usage below yellow threshold (normal) or no data for today ($0.00)
- 🟡 **Yellow**: Usage above yellow but below red threshold (warning)
- 🔴 **Red**: Usage above red threshold (critical)
- ⚪️ **Unknown**: ccusage binary unavailable, command failed, or data parsing error

**Important**: The application now distinguishes between two zero-cost scenarios:
- `CC 🟢 $0.00` - ccusage is working but you haven't used Claude Code today
- `CC ⚪️ Unknown` - ccusage binary is unavailable or not functioning properly

## Development

### Project Structure

```
src/
├── main.go                 # Application entry point with systray integration
├── models/                 # Config, alert status, template data, usage state
├── services/               # Configuration + ccusage polling services
└── lib/                    # Logging, error helpers, template engine

docs/
└── screenshot.png          # UI reference used in this README

tests live alongside the code as `*_test.go` files under each package.
```

### Available Make Targets

```bash
make help                 # Show all available targets
make build               # Build the binary
make run                 # Run the application
make daemon              # Run as daemon (background process)
make test                # Run tests
make test-race           # Run tests with race detection
make bench               # Run benchmarks
make lint                # Run linter
make lint-fix            # Run linter with auto-fix
make fmt                 # Format Go code
make vet                 # Run go vet
make format              # Run fmt + lint-fix together
make clean               # Clean build artifacts
make deps                # Download dependencies
make deps-update         # Update dependencies
make coverage            # Run tests with coverage report
make coverage-func       # Show coverage percentage by function
make coverage-html       # Generate HTML coverage report
make dev-setup           # Set up development environment
make install             # Install the binary
make install-service     # Install as systemd service (Linux)
make uninstall-service   # Remove systemd service (Linux)
make security            # Check for security vulnerabilities
make check               # Run lint, test, and build
make ci                  # CI pipeline (deps, lint, test, build)
```

### Testing

The project includes comprehensive test coverage with multiple test types:

```bash
# Run all tests
make test

# Run tests with coverage report
make coverage

# Run tests with coverage (HTML report)
make coverage-html

# Show coverage by function
make coverage-func

# Run tests with race detection
make test-race

# Run benchmarks
make bench
```

`make coverage` and `make coverage-html` both emit `coverage.html` so you can open a local report after the tests finish.

#### Test Layout
- Tests live next to their implementation under `src/**/**/*_test.go`
- Table-driven coverage exercises the config, service, and model layers
- Use `make coverage-func` to spot gaps before sending a PR

### Code Quality

```bash
# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Format code
make fmt

# Run go vet
make vet

# Format + lint-fix combined
make format

# Check for security vulnerabilities
make security

# Run all quality checks (lint, test, build)
make check
```

## Dependencies

- [github.com/getlantern/systray](https://github.com/getlantern/systray) - Cross-platform system tray support
- [github.com/adrg/xdg](https://github.com/adrg/xdg) - XDG Base Directory support  
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML configuration parsing
- [github.com/stretchr/testify](https://github.com/stretchr/testify) - Testing toolkit

## Requirements

- **ccusage**: The application requires the `ccusage` binary to be installed and accessible
- **Go**: Version 1.21 or later for building from source
- **Platform**: macOS, Linux, or Windows

## Troubleshooting

### Tray Icon Not Visible

1. Check if the application is running: `ps aux | grep cc-dailyuse-bar`
2. Look for the icon in your system tray/menu bar
3. On macOS, check System Preferences > Security & Privacy > Privacy > Accessibility
4. Try running with debug logging: `go run ./src`

### ccusage Not Found

1. Ensure `ccusage` is installed and in your PATH
2. Check the configuration file for the correct path
3. Test manually: `ccusage daily --json`

If ccusage is completely unavailable, the app will show `CC ⚪️ Unknown`

### Understanding Status Display

The application shows different indicators based on data availability:

**Green Status ($0.00)**: 
- ccusage is working properly
- No Claude Code usage recorded for today
- Normal state for new days or days without usage

**Unknown Status (⚪️)**:
- ccusage binary not found or not executable  
- ccusage command fails or returns invalid data
- Network/permission issues preventing ccusage execution

**Testing Status**:
```bash
# Test if ccusage works (should show Green $0.00 or actual usage)
ccusage daily --json

# Test what happens when ccusage fails (should show Unknown)
# Temporarily set an invalid `ccusage_path` in your config file, then rerun:
./cc-dailyuse-bar
```

### Configuration Issues

1. Check the config file location: `~/.config/cc-dailyuse-bar/config.yaml`
2. Validate YAML syntax
3. Check file permissions

### Debug Logging

Enable debug logging by setting the `debug_level` in your configuration file:

```yaml
debug_level: "DEBUG"
```

Available log levels:
- `DEBUG`: Detailed debugging information
- `INFO`: General information (default)
- `WARN`: Warning messages
- `ERROR`: Error messages only
- `FATAL`: Fatal errors only

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Run the test suite: `make test`
6. Run the linter: `make lint`
7. Submit a pull request
