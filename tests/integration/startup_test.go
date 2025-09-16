package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/adrg/xdg"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/services"
)

// T007: Integration test for application startup and config loading
// This test verifies the complete startup sequence works end-to-end
// MUST FAIL initially (RED phase) until services are implemented

func TestApplicationStartup(t *testing.T) {
	// Arrange - Clean test environment
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-test")
	os.RemoveAll(testConfigDir) // Clean up any previous test data

	// Override XDG config directory for test
	originalConfigHome := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", testConfigDir)
	defer func() {
		if originalConfigHome == "" {
			os.Unsetenv("XDG_CONFIG_HOME")
		} else {
			os.Setenv("XDG_CONFIG_HOME", originalConfigHome)
		}
		os.RemoveAll(testConfigDir) // Clean up test data
	}()

	// Act - Simulate application startup sequence
	configService := &services.ConfigService{}

	// Step 1: Load configuration (should create defaults if missing)
	config, err := configService.Load()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Step 2: Validate loaded configuration
	err = configService.Validate(config)
	assert.NoError(t, err)

	// Step 3: Initialize usage service with config
	usageService := services.NewUsageService()
	err = usageService.SetCCUsagePath(config.CCUsagePath)
	if err != nil {
		t.Logf("ccusage not available at default path (expected in test environment): %v", err)
	}

	// Assert - Configuration should be persisted
	configPath := configService.GetConfigPath()
	assert.FileExists(t, configPath)

	// Assert - Configuration should contain expected defaults
	assert.Equal(t, "ccusage", config.CCUsagePath)
	assert.Equal(t, 30, config.UpdateInterval)
	assert.Contains(t, config.DisplayFormat, "{{.Count}}")
	assert.Equal(t, 10.0, config.YellowThreshold)
	assert.Equal(t, 20.0, config.RedThreshold)
	assert.Equal(t, "INFO", config.DebugLevel)
}

func TestStartup_ConfigDirectoryCreation(t *testing.T) {
	// Arrange - Use a completely new directory
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-startup-test")
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

	// Act
	configService := &services.ConfigService{}
	config, err := configService.Load()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Verify directory was created
	expectedDir := filepath.Join(xdg.ConfigHome, "cc-dailyuse-bar")
	assert.DirExists(t, expectedDir)
}

func TestStartup_XDGCompliance(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}

	// Act
	configPath := configService.GetConfigPath()

	// Assert - Path should be XDG compliant
	assert.Contains(t, configPath, "cc-dailyuse-bar")
	assert.Contains(t, configPath, "config.yaml")

	// Should use XDG_CONFIG_HOME or fallback to ~/.config
	homeDir, _ := os.UserHomeDir()
	expectedPaths := []string{
		xdg.ConfigHome,
		filepath.Join(homeDir, ".config"),
	}

	foundValidPath := false
	for _, expectedPath := range expectedPaths {
		if len(configPath) > len(expectedPath) && configPath[:len(expectedPath)] == expectedPath {
			foundValidPath = true
			break
		}
	}
	assert.True(t, foundValidPath, "Config path should use XDG-compliant location: %s", configPath)
}
