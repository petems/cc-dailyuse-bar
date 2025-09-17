package services

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
)

// ConfigService implements configuration management with XDG compliance
type ConfigService struct {
	logger     *lib.Logger
	configPath string // Override for testing

	mu         sync.Mutex
	cachedPath string
	cachedEnv  string
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

const fallbackMarkerName = "config.path"

func (cs *ConfigService) Load() (*models.Config, error) {
	configPath := cs.GetConfigPath()
	attemptedFallback := false

	for {
		if _, err := os.Stat(configPath); err != nil {
			if os.IsNotExist(err) {
				defaults := models.ConfigDefaults()
				cs.logInfo("Config file not found, using defaults", map[string]interface{}{
					"path": configPath,
				})
				return defaults, nil
			}

			if os.IsPermission(err) && !attemptedFallback {
				configPath = cs.useFallbackPath()
				attemptedFallback = true
				continue
			}

			return nil, err
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}

		var config models.Config
		if err := yaml.Unmarshal(data, &config); err != nil {
			cs.warn("Invalid configuration detected, attempting to restore defaults", map[string]interface{}{
				"error": err,
				"path":  configPath,
			})
			if restoreErr := cs.Save(models.ConfigDefaults()); restoreErr != nil {
				cs.warn("Failed to restore default configuration", map[string]interface{}{"error": restoreErr})
			}
			return nil, err
		}

		if err := cs.Validate(&config); err != nil {
			return nil, err
		}

		return &config, nil
	}
}

// Save persists configuration to XDG-compliant storage
// Creates directories if they don't exist
// Returns error for validation failures or write issues
func (cs *ConfigService) Save(config *models.Config) error {
	// Validate first
	if err := cs.Validate(config); err != nil {
		return err
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	configPath := cs.GetConfigPath()
	if err := cs.writeConfigFile(configPath, data); err != nil {
		if os.IsPermission(err) {
			fallbackPath := cs.useFallbackPath()
			if fallbackPath != configPath {
				if fallbackErr := cs.writeConfigFile(fallbackPath, data); fallbackErr == nil {
					return nil
				}
			}
		}
		return err
	}

	return nil
}

// Validate checks configuration values for correctness
// Returns error describing first validation failure found
func (cs *ConfigService) Validate(config *models.Config) error {
	return config.Validate()
}

// GetConfigPath returns the full path to the config file
// Useful for debugging and user information
func (cs *ConfigService) GetConfigPath() string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.configPath != "" {
		return cs.configPath
	}

	if persisted := cs.loadPersistedFallbackPath(); persisted != "" {
		cs.configPath = persisted
		return cs.configPath
	}

	currentEnv := os.Getenv("XDG_CONFIG_HOME")
	if currentEnv != cs.cachedEnv || cs.cachedPath == "" {
		if currentEnv != cs.cachedEnv {
			xdg.Reload()
		}

		cs.cachedEnv = currentEnv
		cs.cachedPath = filepath.Join(xdg.ConfigHome, "cc-dailyuse-bar", "config.yaml")
	}

	return cs.cachedPath
}

// SetConfigPath sets a custom config path for testing
func (cs *ConfigService) SetConfigPath(path string) {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	cs.configPath = path
	cs.cachedPath = ""
	cs.cachedEnv = ""
}

func (cs *ConfigService) fallbackConfigPath() string {
	return filepath.Join(os.TempDir(), "cc-dailyuse-bar", "config.yaml")
}

func (cs *ConfigService) useFallbackPath() string {
	fallback := cs.fallbackConfigPath()
	cs.SetConfigPath(fallback)
	cs.persistFallbackPath(fallback)
	cs.warn("Using fallback config path", map[string]interface{}{"path": fallback})
	return fallback
}

func (cs *ConfigService) writeConfigFile(path string, data []byte) error {
	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func (cs *ConfigService) warn(message string, context map[string]interface{}) {
	if cs.logger == nil {
		return
	}
	cs.logger.Warn(message, context)
}

func (cs *ConfigService) logInfo(message string, context map[string]interface{}) {
	if cs.logger == nil {
		return
	}
	cs.logger.Info(message, context)
}

func (cs *ConfigService) fallbackMarkerPath() string {
	return filepath.Join(os.TempDir(), "cc-dailyuse-bar", fallbackMarkerName)
}

func (cs *ConfigService) persistFallbackPath(path string) {
	markerPath := cs.fallbackMarkerPath()
	if err := os.MkdirAll(filepath.Dir(markerPath), 0755); err != nil {
		cs.warn("Failed to persist fallback path", map[string]interface{}{"error": err})
		return
	}
	if err := os.WriteFile(markerPath, []byte(path), 0600); err != nil {
		cs.warn("Failed to record fallback path", map[string]interface{}{"error": err})
	}
}

func (cs *ConfigService) loadPersistedFallbackPath() string {
	markerPath := cs.fallbackMarkerPath()
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(string(data))
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err != nil {
		return ""
	}
	return path
}
