package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

// T032: Unit tests for Config validation

func TestConfig_Defaults(t *testing.T) {
	config := models.ConfigDefaults()

	assert.Equal(t, "ccusage", config.CCUsagePath)
	assert.Equal(t, 30, config.UpdateInterval)
	assert.Equal(t, "Claude: {{.Count}} ({{.Status}})", config.DisplayFormat)
	assert.Equal(t, 10.0, config.YellowThreshold)
	assert.Equal(t, 20.0, config.RedThreshold)
	assert.Equal(t, "INFO", config.DebugLevel)
}

func TestConfigService_Validate_ValidConfig(t *testing.T) {
	service := services.NewConfigService()
	config := &models.Config{
		CCUsagePath:     "/usr/local/bin/ccusage",
		UpdateInterval:  60,
		DisplayFormat:   "Usage: {{.Count}}",
		YellowThreshold: 8.0,
		RedThreshold:    15.0,
		DebugLevel:      "INFO",
	}

	err := service.Validate(config)
	assert.NoError(t, err)
}

func TestConfigService_Validate_InvalidUpdateInterval(t *testing.T) {
	service := services.NewConfigService()

	testCases := []struct {
		name     string
		interval int
	}{
		{"zero interval", 0},
		{"negative interval", -10},
		{"too low", 4},
		{"too high", 3601},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := models.ConfigDefaults()
			config.UpdateInterval = tc.interval

			err := service.Validate(config)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "update_interval")
		})
	}
}

func TestConfigService_Validate_InvalidThresholds(t *testing.T) {
	service := services.NewConfigService()

	testCases := []struct {
		name   string
		yellow float64
		red    float64
	}{
		{"negative yellow", -1.0, 10.0},
		{"negative red", 5.0, -1.0},
		{"red lower than yellow", 10.0, 5.0},
		{"equal thresholds", 5.0, 5.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := models.ConfigDefaults()
			config.YellowThreshold = tc.yellow
			config.RedThreshold = tc.red

			err := service.Validate(config)
			assert.Error(t, err)
		})
	}
}

func TestConfigService_Validate_EmptyFields(t *testing.T) {
	service := services.NewConfigService()

	testCases := []struct {
		name     string
		modifier func(*models.Config)
	}{
		{name: "empty ccusage path", modifier: func(c *models.Config) { c.CCUsagePath = "" }},
		{name: "empty display format", modifier: func(c *models.Config) { c.DisplayFormat = "" }},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := models.ConfigDefaults()
			tc.modifier(config)

			err := service.Validate(config)
			assert.Error(t, err)
		})
	}
}

func TestConfigService_Validate_ValidThresholds(t *testing.T) {
	service := services.NewConfigService()

	testCases := []struct {
		name   string
		yellow float64
		red    float64
	}{
		{"default thresholds", 5.0, 10.0},
		{"small values", 0.1, 0.2},
		{"large values", 100.0, 200.0},
		{"large difference", 1.0, 50.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := models.ConfigDefaults()
			config.YellowThreshold = tc.yellow
			config.RedThreshold = tc.red

			err := service.Validate(config)
			assert.NoError(t, err)
		})
	}
}

func TestConfigService_Validate_DisplayFormatValidation(t *testing.T) {
	service := services.NewConfigService()

	testCases := []struct {
		name   string
		format string
		valid  bool
	}{
		{"valid template", "{{.Count}}: ${{.Cost}}", true},
		{"simple text", "Claude Usage", true},
		{"all fields", "{{.Count}} {{.Cost}} {{.Status}} {{.Date}} {{.Time}}", true},
		{"invalid template", "{{.InvalidField}}", true}, // Template validation is lenient
		{"empty format", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config := models.ConfigDefaults()
			config.DisplayFormat = tc.format

			err := service.Validate(config)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestConfig_RoundTripValidation(t *testing.T) {
	// Test that a valid config remains valid after serialization
	service := services.NewConfigService()
	original := models.ConfigDefaults()

	// Validate original
	err := service.Validate(original)
	require.NoError(t, err)

	// Save and load
	err = service.Save(original)
	require.NoError(t, err)

	loaded, err := service.Load()
	require.NoError(t, err)

	// Validate loaded
	err = service.Validate(loaded)
	assert.NoError(t, err)

	// Values should match
	assert.Equal(t, original.CCUsagePath, loaded.CCUsagePath)
	assert.Equal(t, original.UpdateInterval, loaded.UpdateInterval)
	assert.Equal(t, original.DisplayFormat, loaded.DisplayFormat)
	assert.Equal(t, original.YellowThreshold, loaded.YellowThreshold)
	assert.Equal(t, original.RedThreshold, loaded.RedThreshold)
}
