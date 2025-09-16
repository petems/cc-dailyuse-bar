package contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

func TestConfigService_Load(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}

	// Act - Test loading defaults when no config file exists
	config, err := configService.Load()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, config)
	// Note: Load() returns defaults when no config file exists
	// The actual values depend on whether a config file was previously saved
	assert.NotEmpty(t, config.CCUsagePath)
	assert.Greater(t, config.UpdateInterval, 0)
	assert.Greater(t, config.YellowThreshold, 0.0)
	assert.Greater(t, config.RedThreshold, config.YellowThreshold)
	assert.NotEmpty(t, config.DebugLevel)
}

func TestConfigService_Save(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}
	config := &models.Config{
		CCUsagePath:     "/custom/ccusage",
		UpdateInterval:  60,
		YellowThreshold: 8.0,
		RedThreshold:    15.0,
		DebugLevel:      "DEBUG",
	}

	// Act
	err := configService.Save(config)

	// Assert
	assert.NoError(t, err)

	// Verify by loading
	loadedConfig, err := configService.Load()
	require.NoError(t, err)
	assert.Equal(t, config.CCUsagePath, loadedConfig.CCUsagePath)
	assert.Equal(t, config.UpdateInterval, loadedConfig.UpdateInterval)
	assert.Equal(t, config.YellowThreshold, loadedConfig.YellowThreshold)
	assert.Equal(t, config.RedThreshold, loadedConfig.RedThreshold)
	assert.Equal(t, config.DebugLevel, loadedConfig.DebugLevel)
}

func TestConfigService_Validate_Valid(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}
	config := &models.Config{
		CCUsagePath:     "ccusage",
		UpdateInterval:  30,
		YellowThreshold: 5.0,
		RedThreshold:    10.0,
		DebugLevel:      "INFO",
	}

	// Act
	err := configService.Validate(config)

	// Assert
	assert.NoError(t, err)
}

func TestConfigService_Validate_Invalid(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}

	testCases := []struct {
		config *models.Config
		name   string
	}{
		{
			name: "UpdateInterval too low",
			config: &models.Config{
				UpdateInterval:  5,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
			},
		},
		{
			name: "UpdateInterval too high",
			config: &models.Config{
				UpdateInterval:  400,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
			},
		},
		{
			name: "RedThreshold lower than YellowThreshold",
			config: &models.Config{
				UpdateInterval:  30,
				YellowThreshold: 10.0,
				RedThreshold:    5.0,
			},
		},
		{
			name: "Negative thresholds",
			config: &models.Config{
				UpdateInterval:  30,
				YellowThreshold: -1.0,
				RedThreshold:    10.0,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := configService.Validate(tc.config)

			// Assert
			assert.Error(t, err)
		})
	}
}

func TestConfigService_GetConfigPath(t *testing.T) {
	// Arrange
	configService := &services.ConfigService{}

	// Act
	path := configService.GetConfigPath()

	// Assert
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "cc-dailyuse-bar")
	assert.Contains(t, path, "config.yaml")
}
