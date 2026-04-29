package services

import (
	"errors"
	"fmt"
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
	writeFile  func(string, []byte, os.FileMode) error
	mkdirAll   func(string, os.FileMode) error
}

// NewConfigService creates a new ConfigService instance
func NewConfigService() *ConfigService {
	return &ConfigService{
		logger:    lib.NewLogger("config-service"),
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
		mkdirAll:  os.MkdirAll,
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

// Save writes the configuration to disk
func (cs *ConfigService) Save(config *models.Config) error {
	// Validate before saving
	if err := cs.Validate(config); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	configPath := cs.GetConfigPath()

	// Ensure directory exists
	if err := cs.EnsureConfigDir(); err != nil {
		return err
	}

	if err := cs.writeFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// EnsureConfigDir ensures the configuration directory exists
func (cs *ConfigService) EnsureConfigDir() error {
	dir := filepath.Dir(cs.GetConfigPath())
	if err := cs.mkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return nil
}

// Validate checks configuration values for correctness
// Returns error describing first validation failure found
func (cs *ConfigService) Validate(config *models.Config) error {
	if config == nil {
		return lib.ValidationError("config is nil")
	}
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

// SetWriteFile allows tests to override the file writer logic.
func (cs *ConfigService) SetWriteFile(writer func(string, []byte, os.FileMode) error) {
	if writer == nil {
		cs.writeFile = os.WriteFile
		return
	}
	cs.writeFile = writer
}

// SetMkdirAll allows tests to override the mkdir logic.
func (cs *ConfigService) SetMkdirAll(mkdir func(string, os.FileMode) error) {
	if mkdir == nil {
		cs.mkdirAll = os.MkdirAll
		return
	}
	cs.mkdirAll = mkdir
}
