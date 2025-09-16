package models

import (
	"errors"
	"strings"
)

// Config represents the application configuration structure
type Config struct {
	CCUsagePath     string  `yaml:"ccusage_path"`
	UpdateInterval  int     `yaml:"update_interval"`
	YellowThreshold float64 `yaml:"yellow_threshold"`
	RedThreshold    float64 `yaml:"red_threshold"`
	DebugLevel      string  `yaml:"debug_level"`
}

// ConfigDefaults returns a Config struct with default values
func ConfigDefaults() *Config {
	return &Config{
		CCUsagePath:     "ccusage",
		UpdateInterval:  30,
		YellowThreshold: 10.00,
		RedThreshold:    20.00,
		DebugLevel:      "INFO",
	}
}

// Validate checks configuration values for correctness
// Returns error describing first validation failure found
func (c *Config) Validate() error {
	// Validate required fields
	if c.CCUsagePath == "" {
		return errors.New("ccusage_path cannot be empty")
	}

	// Validate update interval
	if c.UpdateInterval < 10 || c.UpdateInterval > 300 {
		return errors.New("update_interval must be between 10 and 300 seconds")
	}

	// Validate thresholds
	if c.YellowThreshold < 0 {
		return errors.New("yellow_threshold must be positive")
	}
	if c.RedThreshold < 0 {
		return errors.New("red_threshold must be positive")
	}
	if c.RedThreshold <= c.YellowThreshold {
		return errors.New("red_threshold must be greater than yellow_threshold")
	}

	// Validate debug level
	validLevels := []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"}
	upperLevel := strings.ToUpper(c.DebugLevel)
	valid := false
	for _, level := range validLevels {
		if upperLevel == level {
			valid = true
			break
		}
	}
	if !valid {
		return errors.New("debug_level must be one of: DEBUG, INFO, WARN, ERROR, FATAL")
	}

	return nil
}

// GetLogLevel converts the debug level string to a LogLevel enum
// Returns INFO level if the string is invalid
func (c *Config) GetLogLevel() int {
	switch strings.ToUpper(c.DebugLevel) {
	case "DEBUG":
		return 0 // DEBUG
	case "INFO":
		return 1 // INFO
	case "WARN":
		return 2 // WARN
	case "ERROR":
		return 3 // ERROR
	case "FATAL":
		return 4 // FATAL
	default:
		return 1 // Default to INFO
	}
}
