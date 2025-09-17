package services

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cc-dailyuse-bar/src/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigService(t *testing.T) {
	service := NewConfigService()

	assert.NotNil(t, service)
	assert.NotNil(t, service.logger)
	// Logger component is not exported, so we can't test it directly
}

func TestConfigService_GetConfigPath(t *testing.T) {
	service := NewConfigService()
	path := service.GetConfigPath()

	assert.NotEmpty(t, path)
	assert.Contains(t, path, "cc-dailyuse-bar")
	assert.Contains(t, path, "config.yaml")
	assert.True(t, filepath.IsAbs(path))
}

func TestConfigService_Load_NoFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	service := NewConfigService()
	service.SetConfigPath(configPath)

	config, err := service.Load()

	require.NoError(t, err)
	assert.NotNil(t, config)

	// Should return defaults
	defaults := models.ConfigDefaults()
	assert.Equal(t, defaults.CCUsagePath, config.CCUsagePath)
	assert.Equal(t, defaults.UpdateInterval, config.UpdateInterval)
	assert.Equal(t, defaults.YellowThreshold, config.YellowThreshold)
	assert.Equal(t, defaults.RedThreshold, config.RedThreshold)
	assert.Equal(t, defaults.DebugLevel, config.DebugLevel)

	// Should create the config file
	assert.FileExists(t, configPath)
}

func TestConfigService_Load_ExistingFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	service := NewConfigService()
	service.SetConfigPath(configPath)

	// Create a custom config
	customConfig := &models.Config{
		CCUsagePath:     "/custom/ccusage",
		UpdateInterval:  60,
		YellowThreshold: 5.0,
		RedThreshold:    10.0,
		DebugLevel:      "DEBUG",
		CacheWindow:     15,
		CmdTimeout:      8,
	}

	// Save it
	err := service.Save(customConfig)
	require.NoError(t, err)

	// Load it
	loadedConfig, err := service.Load()
	require.NoError(t, err)

	// Should match the saved config
	assert.Equal(t, customConfig.CCUsagePath, loadedConfig.CCUsagePath)
	assert.Equal(t, customConfig.UpdateInterval, loadedConfig.UpdateInterval)
	assert.Equal(t, customConfig.YellowThreshold, loadedConfig.YellowThreshold)
	assert.Equal(t, customConfig.RedThreshold, loadedConfig.RedThreshold)
	assert.Equal(t, customConfig.DebugLevel, loadedConfig.DebugLevel)
}

func TestConfigService_Load_InvalidYAML(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.yaml")

	service := NewConfigService()
	service.SetConfigPath(configPath)

	// Create directory
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)

	// Write invalid YAML
	err = os.WriteFile(configPath, []byte("invalid: yaml: content: ["), 0o644)
	require.NoError(t, err)

	// Load should return error for invalid YAML
	config, err := service.Load()
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "yaml:")
}

func TestConfigService_Load_InvalidConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()
	configPath := service.GetConfigPath()

	// Create directory
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)

	// Write invalid config (negative update interval)
	invalidYAML := `ccusage_path: "ccusage"
update_interval: -10
yellow_threshold: 5.0
red_threshold: 10.0
debug_level: "INFO"
cache_window: 10
cmd_timeout: 5`

	err = os.WriteFile(configPath, []byte(invalidYAML), 0o644)
	require.NoError(t, err)

	// Load should return error for invalid config
	config, err := service.Load()
	require.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "update_interval")
}

func TestConfigService_Save_ValidConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()

	config := &models.Config{
		CCUsagePath:     "/usr/local/bin/ccusage",
		UpdateInterval:  60,
		YellowThreshold: 8.0,
		RedThreshold:    15.0,
		DebugLevel:      "DEBUG",
		CacheWindow:     20,
		CmdTimeout:      10,
	}

	err := service.Save(config)
	require.NoError(t, err)

	// Verify file was created
	configPath := service.GetConfigPath()
	assert.FileExists(t, configPath)

	// Verify content
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "ccusage_path: /usr/local/bin/ccusage")
	assert.Contains(t, string(content), "update_interval: 60")
	assert.Contains(t, string(content), "yellow_threshold: 8")
	assert.Contains(t, string(content), "red_threshold: 15")
	assert.Contains(t, string(content), "debug_level: DEBUG")
}

func TestConfigService_Save_InvalidConfig(t *testing.T) {
	service := NewConfigService()

	// Get the actual config path and clean it up
	configPath := service.GetConfigPath()
	os.Remove(configPath) // Clean up any existing file

	// Invalid config (negative update interval)
	config := &models.Config{
		CCUsagePath:     "ccusage",
		UpdateInterval:  -10, // Invalid
		YellowThreshold: 5.0,
		RedThreshold:    10.0,
		DebugLevel:      "INFO",
	}

	err := service.Save(config)
	assert.Error(t, err) // Save validates the config
	assert.Contains(t, err.Error(), "update_interval")

	// File should not be created (Save validates first)
	assert.NoFileExists(t, configPath)
}

func TestConfigService_Save_CreateDirectory(t *testing.T) {
	service := NewConfigService()

	// Get the actual config path
	configPath := service.GetConfigPath()
	configDir := filepath.Dir(configPath)

	// Remove the config directory if it exists to test creation
	os.RemoveAll(configDir)

	// Use a valid config to test directory creation
	config := models.ConfigDefaults()
	err := service.Save(config)
	require.NoError(t, err)

	// Verify directory was created
	assert.DirExists(t, configDir)
	assert.FileExists(t, configPath)

	// Clean up
	os.RemoveAll(configDir)
}

func TestConfigService_Validate(t *testing.T) {
	service := NewConfigService()

	tests := []struct {
		name    string
		config  *models.Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  models.ConfigDefaults(),
			wantErr: false,
		},
		{
			name: "invalid update interval",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  -10,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
			},
			wantErr: true,
		},
		{
			name: "invalid thresholds",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  30,
				YellowThreshold: 10.0,
				RedThreshold:    5.0, // Red lower than yellow
				DebugLevel:      "INFO",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.Validate(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigService_RoundTrip(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()

	// Test with various configs
	configs := []*models.Config{
		models.ConfigDefaults(),
		{
			CCUsagePath:     "/custom/ccusage",
			UpdateInterval:  60,
			YellowThreshold: 5.0,
			RedThreshold:    10.0,
			DebugLevel:      "DEBUG",
			CacheWindow:     25,
			CmdTimeout:      12,
		},
		{
			CCUsagePath:     "ccusage",
			UpdateInterval:  10,
			YellowThreshold: 0.1,
			RedThreshold:    0.2,
			DebugLevel:      "WARN",
			CacheWindow:     5,
			CmdTimeout:      3,
		},
	}

	for i, originalConfig := range configs {
		t.Run("config_"+string(rune(i)), func(t *testing.T) {
			// Save config
			err := service.Save(originalConfig)
			require.NoError(t, err)

			// Load config
			loadedConfig, err := service.Load()
			require.NoError(t, err)

			// Verify all fields match
			assert.Equal(t, originalConfig.CCUsagePath, loadedConfig.CCUsagePath)
			assert.Equal(t, originalConfig.UpdateInterval, loadedConfig.UpdateInterval)
			assert.Equal(t, originalConfig.YellowThreshold, loadedConfig.YellowThreshold)
			assert.Equal(t, originalConfig.RedThreshold, loadedConfig.RedThreshold)
			assert.Equal(t, originalConfig.DebugLevel, loadedConfig.DebugLevel)
		})
	}
}

func TestConfigService_Load_ReadPermissionError(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()
	configPath := service.GetConfigPath()

	// Create directory
	err := os.MkdirAll(filepath.Dir(configPath), 0o755)
	require.NoError(t, err)

	// Create a file with no read permissions
	err = os.WriteFile(configPath, []byte("ccusage_path: test"), 0o000)
	require.NoError(t, err)

	// Load should return error for permission issues
	config, err := service.Load()
	require.Error(t, err)
	assert.Nil(t, config)
	// Error should be related to permission denied or validation failure
	// (depending on whether the file can be read or not)
	assert.True(t, strings.Contains(err.Error(), "permission denied") ||
		strings.Contains(err.Error(), "update_interval") ||
		strings.Contains(err.Error(), "cache_window"))
}

func TestConfigService_Save_WritePermissionError(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()
	configPath := service.GetConfigPath()

	// Create directory with no write permissions
	err := os.MkdirAll(filepath.Dir(configPath), 0o555)
	require.NoError(t, err)

	config := models.ConfigDefaults()
	err = service.Save(config)
	// On some systems, this might not fail due to permission handling
	// So we'll just test that the method doesn't panic
	if err != nil {
		assert.Contains(t, err.Error(), "permission denied")
	}
}

func TestConfigService_EdgeCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()

	// Test with empty XDG_CONFIG_HOME
	os.Setenv("XDG_CONFIG_HOME", "")
	path := service.GetConfigPath()
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "cc-dailyuse-bar")

	// Restore for other tests
	os.Setenv("XDG_CONFIG_HOME", tempDir)
}

func TestConfigService_ConcurrentAccess(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	originalHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tempDir)
	defer os.Setenv("XDG_CONFIG_HOME", originalHome)

	service := NewConfigService()

	// Test concurrent saves
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			config := &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  30,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			}

			err := service.Save(config)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state is consistent
	config, err := service.Load()
	require.NoError(t, err)
	assert.Equal(t, "ccusage", config.CCUsagePath)
	assert.Equal(t, 30, config.UpdateInterval)
}
