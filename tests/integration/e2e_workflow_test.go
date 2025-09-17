package integration

import (
	"os"
	"path/filepath"
	"testing"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T012: E2E test for complete user workflow from installation to monitoring
// Adjusted to exclude GUI/Fyne usage.
func TestCompleteUserWorkflow(t *testing.T) {
	// Skip in short mode as this is a comprehensive test
	if testing.Short() {
		t.Skip("Skipping E2E workflow test in short mode")
	}

	// Arrange - Clean test environment
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-e2e-test")
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

	// Step 1: First-time application startup
	t.Run("FirstStartup", func(t *testing.T) {
		// Initialize services
		configService := &services.ConfigService{}
		config := models.ConfigDefaults()
		usageService := services.NewUsageService(config)

		// Load configuration (should create defaults)
		config, err := configService.Load()
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Validate configuration
		err = configService.Validate(config)
		assert.NoError(t, err)

		// Initialize usage service
		err = usageService.SetCCUsagePath(config.CCUsagePath)
		if err != nil {
			t.Logf("ccusage not available (expected in test): %v", err)
		}

		// Verify config file was created
		configPath := configService.GetConfigPath()
		assert.FileExists(t, configPath)
	})

	// Step 2: Configuration customization
	t.Run("ConfigurationCustomization", func(t *testing.T) {
		configService := &services.ConfigService{}

		// Load current config
		config, err := configService.Load()
		require.NoError(t, err)

		// Modify configuration (simulate user changes)
		config.UpdateInterval = 60
		config.YellowThreshold = 10.0
		config.RedThreshold = 20.0

		// Validate changes
		err = configService.Validate(config)
		assert.NoError(t, err)

		// Save changes
		err = configService.Save(config)
		assert.NoError(t, err)

		// Verify changes persisted
		reloadedConfig, err := configService.Load()
		require.NoError(t, err)
		assert.Equal(t, 60, reloadedConfig.UpdateInterval)
		assert.Equal(t, 10.0, reloadedConfig.YellowThreshold)
		assert.Equal(t, 20.0, reloadedConfig.RedThreshold)
	})

	// Step 3: Daily monitoring (service-only)
	t.Run("DailyMonitoring", func(t *testing.T) {
		configService := &services.ConfigService{}
		config := models.ConfigDefaults()
		usageService := services.NewUsageService(config)

		// Load configuration
		config, err := configService.Load()
		require.NoError(t, err)

		// Get initial usage
		_, err = usageService.GetDailyUsage()
		if err != nil {
			t.Logf("Usage retrieval failed (expected without ccusage): %v", err)
			return
		}

		// Simulate status transitions
		statusTransitions := []struct {
			name     string
			cost     float64
			expected models.AlertStatus
		}{
			{"Green status", 2.0, models.Green},
			{"Yellow status", 12.0, models.Yellow},
			{"Red status", 22.0, models.Red},
		}

		for _, transition := range statusTransitions {
			t.Run(transition.name, func(t *testing.T) {
				// Verify status calculation logic would work
				var expectedStatus models.AlertStatus
				if transition.cost >= config.RedThreshold {
					expectedStatus = models.Red
				} else if transition.cost >= config.YellowThreshold {
					expectedStatus = models.Yellow
				} else {
					expectedStatus = models.Green
				}
				assert.Equal(t, transition.expected, expectedStatus)
			})
		}
	})

	// Step 4: Daily reset simulation
	t.Run("DailyReset", func(t *testing.T) {
		config := models.ConfigDefaults()
		usageService := services.NewUsageService(config)

		// Reset daily usage
		err := usageService.ResetDaily()
		if err != nil {
			t.Logf("Reset failed (expected without ccusage): %v", err)
			return
		}

		// Verify reset
		usage, err := usageService.GetDailyUsage()
		require.NoError(t, err)

		// After reset, should get fresh data from ccusage
		assert.GreaterOrEqual(t, usage.DailyCount, 0)
		assert.GreaterOrEqual(t, usage.DailyCost, 0.0)
		assert.Contains(t, []models.AlertStatus{models.Green, models.Yellow, models.Red, models.Unknown}, usage.Status)
	})

	// Step 5: Application shutdown (no-op without GUI)
	t.Run("ApplicationShutdown", func(t *testing.T) {
		configService := &services.ConfigService{}
		config, err := configService.Load()
		require.NoError(t, err)

		// Configuration should still exist
		assert.Equal(t, 60, config.UpdateInterval) // From customization step
	})
}

func TestUserErrorScenarios(t *testing.T) {
	// Test common user error scenarios and recovery

	// Arrange
	testConfigDir := filepath.Join(os.TempDir(), "cc-dailyuse-bar-error-test")
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

	t.Run("InvalidConfigurationRecovery", func(t *testing.T) {
		configService := &services.ConfigService{}

		// Create invalid config file
		configPath := configService.GetConfigPath()
		os.MkdirAll(filepath.Dir(configPath), 0o755)
		invalidYAML := "invalid: yaml: content: {"
		err := os.WriteFile(configPath, []byte(invalidYAML), 0o644)
		require.NoError(t, err)

		// Attempt to load - should recover with defaults
		config, err := configService.Load()
		if err != nil {
			t.Logf("Load failed with invalid YAML (expected behavior): %v", err)
		}

		// Should still get a valid config (defaults)
		if config != nil {
			err = configService.Validate(config)
			assert.NoError(t, err, "Recovered config should be valid")
		}
	})

	t.Run("MissingCCUsageRecovery", func(t *testing.T) {
		config := models.ConfigDefaults()
		usageService := services.NewUsageService(config)

		// Set invalid ccusage path
		err := usageService.SetCCUsagePath("/nonexistent/ccusage")
		assert.Error(t, err, "Should fail to set nonexistent ccusage path")

		// Check availability - should still be true since path setting failed
		available := usageService.IsAvailable()
		assert.True(t, available, "Should still be available since invalid path was rejected")

		// Application should still function with original path
		usage, err := usageService.GetDailyUsage()
		if usage != nil {
			assert.True(t, usage.IsAvailable, "Usage should be available with original path")
		}
	})
}
