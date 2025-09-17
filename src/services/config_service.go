package services

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
)

// ConfigService implements configuration management with XDG compliance
type ConfigService struct {
	logger     *lib.Logger
	configPath string // Override for testing
	readFile   func(string) ([]byte, error)
}

// NewConfigService creates a new ConfigService instance
func NewConfigService() *ConfigService {
	return &ConfigService{
		logger:   lib.NewLogger("config-service"),
		readFile: os.ReadFile,
	}
}

// Load reads configuration from XDG-compliant storage
// Returns default config if file doesn't exist
// Returns error for permission/system issues, corrupted files, or invalid configurations
func (cs *ConfigService) Load() (*models.Config, error) {
	configPath := cs.GetConfigPath()

	data, err := cs.readFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return models.ConfigDefaults(), nil
		}
		return nil, err
	}

	// Parse YAML - propagate parsing errors (corrupted file)
	var config models.Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	// Validate the loaded config - propagate validation errors (invalid config)
	if err := cs.Validate(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Validate checks configuration values for correctness
// Returns error describing first validation failure found
func (cs *ConfigService) Validate(config *models.Config) error {
	return config.Validate()
}

// GetConfigPath returns the full path to the config file
// Useful for debugging and user information
func (cs *ConfigService) GetConfigPath() string {
	if cs.configPath != "" {
		return cs.configPath
	}
	return filepath.Join(xdg.ConfigHome, "cc-dailyuse-bar", "config.yaml")
}

// SetConfigPath sets a custom config path for testing
func (cs *ConfigService) SetConfigPath(path string) {
	cs.configPath = path
}

// SetReadFile allows tests to override the file reader logic.
func (cs *ConfigService) SetReadFile(reader func(string) ([]byte, error)) {
	if reader == nil {
		cs.readFile = os.ReadFile
		return
	}
	cs.readFile = reader
}
