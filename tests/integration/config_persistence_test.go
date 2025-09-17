package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

// T010: Integration test for configuration persistence and validation
// This test verifies config can be saved, loaded, and validated end-to-end
// MUST FAIL initially (RED phase) until ConfigService is implemented

func TestConfigurationPersistence(t *testing.T) {
	// Arrange - Clean test environment
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-config-test")
	os.RemoveAll(testConfigDir)

	originalConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", testConfigDir)
	defer func() {
		if originalConfigHome == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigHome)
		}
		os.RemoveAll(testConfigDir)
	}()

	configService := &services.ConfigService{}

	// Act - Save custom configuration
	customConfig := &models.Config{
		CCUsagePath:     "/custom/path/ccusage",
		UpdateInterval:  60,
		YellowThreshold: 7.5,
		RedThreshold:    15.0,
		DebugLevel:      "DEBUG",
		CacheWindow:     25,
		CmdTimeout:      8,
	}

	err := configService.Save(customConfig)
	require.NoError(t, err, "Should be able to save custom configuration")

	// Act - Load configuration
	loadedConfig, err := configService.Load()
	require.NoError(t, err, "Should be able to load saved configuration")

	// Assert - All fields should match
	assert.Equal(t, customConfig.CCUsagePath, loadedConfig.CCUsagePath)
	assert.Equal(t, customConfig.UpdateInterval, loadedConfig.UpdateInterval)
	assert.Equal(t, customConfig.YellowThreshold, loadedConfig.YellowThreshold)
	assert.Equal(t, customConfig.RedThreshold, loadedConfig.RedThreshold)
	assert.Equal(t, customConfig.DebugLevel, loadedConfig.DebugLevel)
	assert.Equal(t, customConfig.CacheWindow, loadedConfig.CacheWindow)
	assert.Equal(t, customConfig.CmdTimeout, loadedConfig.CmdTimeout)

	// Assert - Config file should exist
	configPath := configService.GetConfigPath()
	assert.FileExists(t, configPath, "Config file should be created")
}

func TestConfigurationYAMLFormat(t *testing.T) {
	// Arrange
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-yaml-test")
	os.RemoveAll(testConfigDir)

	originalConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", testConfigDir)
	defer func() {
		if originalConfigHome == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigHome)
		}
		os.RemoveAll(testConfigDir)
	}()

	configService := &services.ConfigService{}

	// Act - Save configuration
	config := &models.Config{
		CCUsagePath:     "ccusage",
		UpdateInterval:  45,
		YellowThreshold: 6.0,
		RedThreshold:    12.0,
		DebugLevel:      "INFO",
		CacheWindow:     15,
		CmdTimeout:      6,
	}

	err := configService.Save(config)
	require.NoError(t, err)

	// Assert - Read raw file and verify YAML format
	configPath := configService.GetConfigPath()
	content, err := os.ReadFile(configPath)
	require.NoError(t, err)

	yamlContent := string(content)
	assert.Contains(t, yamlContent, "ccusage_path: ccusage")
	assert.Contains(t, yamlContent, "update_interval: 45")
	assert.Contains(t, yamlContent, "yellow_threshold: 6")
	assert.Contains(t, yamlContent, "red_threshold: 12")
	assert.Contains(t, yamlContent, "debug_level: INFO")
}

func TestConfigurationValidation(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}

	testCases := []struct {
		config    *models.Config
		name      string
		expectErr bool
	}{
		{
			name: "Valid configuration",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  30,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			},
			expectErr: false,
		},
		{
			name: "Update interval too low",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  5,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			},
			expectErr: true,
		},
		{
			name: "Update interval too high",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  500,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			},
			expectErr: true,
		},
		{
			name: "Red threshold lower than yellow",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  30,
				YellowThreshold: 10.0,
				RedThreshold:    5.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			},
			expectErr: true,
		},
		{
			name: "Negative thresholds",
			config: &models.Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  30,
				YellowThreshold: -1.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			},
			expectErr: true,
		},
		{
			name: "Invalid template syntax",
			config: &models.Config{
				UpdateInterval:  30,
				CCUsagePath:     "", // Empty path to trigger validation error
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
				CacheWindow:     10,
				CmdTimeout:      5,
			},
			expectErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := configService.Validate(tc.config)

			// Assert
			if tc.expectErr {
				assert.Error(t, err, "Validation should fail for: %s", tc.name)
			} else {
				assert.NoError(t, err, "Validation should pass for: %s", tc.name)
			}
		})
	}
}

func TestConfigurationDefaults(t *testing.T) {
	// Arrange - Clean environment with no existing config
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-defaults-test")
	os.RemoveAll(testConfigDir)

	originalConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", testConfigDir)
	defer func() {
		if originalConfigHome == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigHome)
		}
		os.RemoveAll(testConfigDir)
	}()

	configService := &services.ConfigService{}

	// Act - Load configuration when no file exists
	config, err := configService.Load()

	// Assert - Should return default configuration
	require.NoError(t, err, "Loading non-existent config should return defaults")
	assert.NotNil(t, config)

	// Note: Load() returns defaults when no config file exists, or saved config if it exists
	assert.NotEmpty(t, config.CCUsagePath)
	assert.Greater(t, config.UpdateInterval, 0)
	assert.Greater(t, config.YellowThreshold, 0.0)
	assert.Greater(t, config.RedThreshold, config.YellowThreshold)
	assert.NotEmpty(t, config.DebugLevel)

	// Assert - Default config should be valid
	err = configService.Validate(config)
	assert.NoError(t, err, "Default configuration should be valid")
}
