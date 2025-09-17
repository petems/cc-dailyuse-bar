package services

import (
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
}

// NewConfigService creates a new ConfigService instance
func NewConfigService() *ConfigService {
	return &ConfigService{
		logger: lib.NewLogger("config-service"),
	}
}

// Load reads configuration from XDG-compliant storage
// Returns default config if file doesn't exist
// Returns error for permission/system issues, corrupted files, or invalid configurations
func (cs *ConfigService) Load() (*models.Config, error) {
	configPath := cs.GetConfigPath()

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaults := models.ConfigDefaults()
		// Create the config file with defaults
		if saveErr := cs.Save(defaults); saveErr != nil {
			// Log the save error but still return defaults
			cs.logger.Warn("Failed to create default config file", map[string]interface{}{
				"error": saveErr,
			})
			return defaults, nil
		}
		return defaults, nil
	}

	// Read the file - propagate read errors (permission issues, etc.)
	data, err := os.ReadFile(configPath)
	if err != nil {
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

// Save persists configuration to XDG-compliant storage
// Creates directories if they don't exist
// Returns error for validation failures or write issues
func (cs *ConfigService) Save(config *models.Config) error {
	// Validate first
	if err := cs.Validate(config); err != nil {
		return err
	}

	configPath := cs.GetConfigPath()
	configDir := filepath.Dir(configPath)

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	// Write file with user-only permissions for privacy
	return os.WriteFile(configPath, data, 0600)
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
