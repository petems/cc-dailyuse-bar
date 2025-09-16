package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigDefaults(t *testing.T) {
	config := ConfigDefaults()

	assert.Equal(t, "ccusage", config.CCUsagePath)
	assert.Equal(t, 30, config.UpdateInterval)
	assert.Equal(t, 10.0, config.YellowThreshold)
	assert.Equal(t, 20.0, config.RedThreshold)
	assert.Equal(t, "INFO", config.DebugLevel)
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	config := &Config{
		CCUsagePath:     "/usr/local/bin/ccusage",
		UpdateInterval:  60,
		YellowThreshold: 8.0,
		RedThreshold:    15.0,
		DebugLevel:      "INFO",
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_EmptyFields(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected string
	}{
		{
			name: "empty ccusage path",
			config: &Config{
				CCUsagePath:     "",
				UpdateInterval:  30,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "INFO",
			},
			expected: "ccusage_path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestConfig_Validate_Thresholds(t *testing.T) {
	tests := []struct {
		name          string
		yellow        float64
		red           float64
		valid         bool
		expectedError string
	}{
		{"valid thresholds", 5.0, 10.0, true, ""},
		{"small values", 0.1, 0.2, true, ""},
		{"large values", 100.0, 200.0, true, ""},
		{"negative yellow", -1.0, 10.0, false, "yellow_threshold must be positive"},
		{"negative red", 5.0, -1.0, false, "red_threshold must be positive"},
		{"red equals yellow", 5.0, 5.0, false, "red_threshold must be greater than yellow_threshold"},
		{"red less than yellow", 10.0, 5.0, false, "red_threshold must be greater than yellow_threshold"},
		{"zero thresholds", 0.0, 0.0, false, "red_threshold must be greater than yellow_threshold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConfigDefaults()
			config.YellowThreshold = tt.yellow
			config.RedThreshold = tt.red

			err := config.Validate()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestConfig_Validate_UpdateInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		valid    bool
	}{
		{"valid interval 30", 30, true},
		{"minimum valid 10", 10, true},
		{"maximum valid 300", 300, true},
		{"too low 9", 9, false},
		{"too high 301", 301, false},
		{"zero", 0, false},
		{"negative", -5, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConfigDefaults()
			config.UpdateInterval = tt.interval

			err := config.Validate()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "update_interval must be between 10 and 300 seconds")
			}
		})
	}
}

func TestConfig_Validate_DebugLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		valid bool
	}{
		{"DEBUG", "DEBUG", true},
		{"INFO", "INFO", true},
		{"WARN", "WARN", true},
		{"ERROR", "ERROR", true},
		{"FATAL", "FATAL", true},
		{"lowercase debug", "debug", true},
		{"lowercase info", "info", true},
		{"mixed case", "Info", true},
		{"invalid level", "INVALID", false},
		{"empty level", "", false},
		{"numeric level", "1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ConfigDefaults()
			config.DebugLevel = tt.level

			err := config.Validate()
			if tt.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "debug_level must be one of")
			}
		})
	}
}

func TestConfig_GetLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected int
	}{
		{"DEBUG", "DEBUG", 0},
		{"INFO", "INFO", 1},
		{"WARN", "WARN", 2},
		{"ERROR", "ERROR", 3},
		{"FATAL", "FATAL", 4},
		{"lowercase debug", "debug", 0},
		{"lowercase info", "info", 1},
		{"mixed case", "Info", 1},
		{"invalid level", "INVALID", 1}, // Defaults to INFO
		{"empty level", "", 1},          // Defaults to INFO
		{"numeric level", "1", 1},       // Defaults to INFO
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{DebugLevel: tt.level}
			assert.Equal(t, tt.expected, config.GetLogLevel())
		})
	}
}

func TestConfig_Validate_MultipleErrors(t *testing.T) {
	// Test that validation returns the first error found
	config := &Config{
		CCUsagePath:     "",        // First error
		UpdateInterval:  5,         // Second error
		YellowThreshold: -1,        // Fourth error
		RedThreshold:    -1,        // Fifth error
		DebugLevel:      "INVALID", // Sixth error
	}

	err := config.Validate()
	assert.Error(t, err)
	// Should return the first error (empty ccusage_path)
	assert.Contains(t, err.Error(), "ccusage_path cannot be empty")
}

func TestConfig_Validate_EdgeCases(t *testing.T) {
	// Test boundary values
	config := &Config{
		CCUsagePath:     "ccusage",
		UpdateInterval:  10,   // Minimum valid
		YellowThreshold: 0.0,  // Minimum valid (zero)
		RedThreshold:    0.01, // Just above yellow
		DebugLevel:      "INFO",
	}

	err := config.Validate()
	assert.NoError(t, err)

	// Test maximum values
	config.UpdateInterval = 300 // Maximum valid
	config.YellowThreshold = 999.99
	config.RedThreshold = 1000.0

	err = config.Validate()
	assert.NoError(t, err)
}

func TestConfig_Validate_RealWorldScenarios(t *testing.T) {
	// Test realistic configuration scenarios
	scenarios := []struct {
		name   string
		config *Config
		valid  bool
	}{
		{
			name:   "default configuration",
			config: ConfigDefaults(),
			valid:  true,
		},
		{
			name: "high frequency monitoring",
			config: &Config{
				CCUsagePath:     "/usr/local/bin/ccusage",
				UpdateInterval:  10,
				YellowThreshold: 5.0,
				RedThreshold:    10.0,
				DebugLevel:      "WARN",
			},
			valid: true,
		},
		{
			name: "low frequency monitoring",
			config: &Config{
				CCUsagePath:     "ccusage",
				UpdateInterval:  300,
				YellowThreshold: 50.0,
				RedThreshold:    100.0,
				DebugLevel:      "ERROR",
			},
			valid: true,
		},
		{
			name: "custom path with spaces",
			config: &Config{
				CCUsagePath:     "/path with spaces/ccusage",
				UpdateInterval:  60,
				YellowThreshold: 1.0,
				RedThreshold:    2.0,
				DebugLevel:      "DEBUG",
			},
			valid: true,
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.valid {
				assert.NoError(t, err, "Configuration should be valid: %v", tt.config)
			} else {
				assert.Error(t, err, "Configuration should be invalid: %v", tt.config)
			}
		})
	}
}
