# cc-dailyuse-bar

A system tray application that monitors your daily Claude Code usage and displays real-time cost information in your menu bar.

## Features

- **Real-time monitoring**: Displays current daily usage cost and API call count
- **Status indicators**: Color-coded status (🟢 Green, 🟡 Yellow, 🔴 Red) based on configurable thresholds
- **System tray integration**: Runs in the background with menu bar access
- **Automatic updates**: Configurable polling interval for fresh data
- **XDG compliance**: Stores configuration in standard XDG directories
- **Multi-language support**: English and Japanese localization

## Screenshots

The application displays in your system tray as:
- `CC 🟢 $0.45` - Normal usage (below yellow threshold)
- `CC 🟡 $12.50` - High usage (above yellow threshold)
- `CC 🔴 $25.00` - Critical usage (above red threshold)

## Installation

### Prerequisites

- Go 1.19 or later
- `ccusage` binary installed and accessible in PATH
- macOS, Linux, or Windows

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
display_format: "Claude: {{.Count}} ({{.Status}})"
yellow_threshold: 10.00
red_threshold: 20.00
debug_level: "INFO"
```

### Configuration Options

- `ccusage_path`: Path to the ccusage binary (default: "ccusage")
- `update_interval`: Polling interval in seconds (10-300, default: 30)
- `display_format`: Template for display format (default: "Claude: {{.Count}} ({{.Status}})")
- `yellow_threshold`: Cost threshold for yellow warning (default: $10.00)
- `red_threshold`: Cost threshold for red alert (default: $20.00)
- `debug_level`: Logging level - DEBUG, INFO, WARN, ERROR, or FATAL (default: "INFO")

## Usage

### Running the Application

```bash
# Run directly
make run

# Or build and run
make build
./cc-dailyuse-bar
```

### System Tray Menu

Right-click the tray icon to access:
- **Usage Information**: Daily cost, API calls, last update time
- **Settings**: View current configuration
- **Quit**: Exit the application

### Status Indicators

- 🟢 **Green**: Usage below yellow threshold (normal)
- 🟡 **Yellow**: Usage above yellow but below red threshold (warning)
- 🔴 **Red**: Usage above red threshold (critical)
- ⚪️ **Gray**: ccusage unavailable or error state

## Development

### Project Structure

```
src/
├── main.go                 # Application entry point
├── models/                 # Data models
│   ├── alert_status.go     # Status enumeration
│   ├── config.go          # Configuration model
│   ├── template_data.go   # Template data structures
│   └── usage_state.go     # Usage state model
├── services/              # Business logic
│   ├── config_service.go  # Configuration management
│   └── usage_service.go   # Usage data service
├── lib/                   # Utilities
│   ├── errors.go          # Error handling
│   ├── logger.go          # Structured logging
│   └── template_engine.go # Template processing
└── resources/             # Static resources
    └── icons.go           # Tray icons
```

### Available Make Targets

```bash
make help                 # Show all available targets
make build               # Build the binary
make run                 # Run the application
make test                # Run tests
make lint                # Run linter
make clean               # Clean build artifacts
make deps                # Download dependencies
make coverage            # Run tests with coverage
make dev-setup           # Set up development environment
```

### Testing

```bash
# Run all tests
make test

# Run tests with coverage
make coverage

# Run tests with race detection
make test-race

# Run benchmarks
make bench
```

### Linting

```bash
# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Format code
make format
```

## Dependencies

- [github.com/getlantern/systray](https://github.com/getlantern/systray) - Cross-platform system tray support
- [github.com/adrg/xdg](https://github.com/adrg/xdg) - XDG Base Directory support
- [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) - YAML configuration parsing

## Requirements

- **ccusage**: The application requires the `ccusage` binary to be installed and accessible
- **Go**: Version 1.19 or later for building from source
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

## License

[Add your license information here]

## Changelog

### v1.0.0
- Initial release
- System tray integration
- Real-time usage monitoring
- Configurable thresholds
- XDG-compliant configuration
- Multi-language support
