// Package models contains domain models and configuration types.
package models

import (
	"strings"

	"cc-dailyuse-bar/src/lib"
)

const (
	defaultUpdateIntervalSeconds = 30
	defaultYellowThresholdUSD    = 10.00
	defaultRedThresholdUSD       = 20.00
	defaultCacheWindowSeconds    = 10
	defaultCmdTimeoutSeconds     = 5
)

// Config represents the application configuration structure.
type Config struct {
	CCUsagePath     string  `yaml:"ccusage_path"`
	UpdateInterval  int     `yaml:"update_interval"`
	YellowThreshold float64 `yaml:"yellow_threshold"`
	RedThreshold    float64 `yaml:"red_threshold"`
	DebugLevel      string  `yaml:"debug_level"`
	CacheWindow     int     `yaml:"cache_window"` // Cache window in seconds
	CmdTimeout      int     `yaml:"cmd_timeout"`  // Command timeout in seconds
}

// ConfigDefaults returns a Config struct with default values.
func ConfigDefaults() *Config {
	return &Config{
		CCUsagePath:     "ccusage",
		UpdateInterval:  defaultUpdateIntervalSeconds,
		YellowThreshold: defaultYellowThresholdUSD,
		RedThreshold:    defaultRedThresholdUSD,
		DebugLevel:      "INFO",
		CacheWindow:     defaultCacheWindowSeconds, // 10 seconds cache window
		CmdTimeout:      defaultCmdTimeoutSeconds,  // 5 seconds command timeout
	}
}

// Validate checks configuration values for correctness
// Returns error describing first validation failure found.
func (c *Config) Validate() error {
	// Validate required fields
	if c.CCUsagePath == "" {
		return lib.ValidationError("ccusage_path cannot be empty")
	}

	// Validate update interval
	if c.UpdateInterval < 10 || c.UpdateInterval > 300 {
		return lib.ValidationError("update_interval must be between 10 and 300 seconds")
	}

	// Validate thresholds
	if c.YellowThreshold < 0 {
		return lib.ValidationError("yellow_threshold must be positive")
	}
	if c.RedThreshold < 0 {
		return lib.ValidationError("red_threshold must be positive")
	}
	if c.RedThreshold <= c.YellowThreshold {
		return lib.ValidationError("red_threshold must be greater than yellow_threshold")
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
		return lib.ValidationError("debug_level must be one of: DEBUG, INFO, WARN, ERROR, FATAL")
	}

	// Validate cache window
	if c.CacheWindow < 1 || c.CacheWindow > 300 {
		return lib.ValidationError("cache_window must be between 1 and 300 seconds")
	}

	// Validate command timeout
	if c.CmdTimeout < 1 || c.CmdTimeout > 60 {
		return lib.ValidationError("cmd_timeout must be between 1 and 60 seconds")
	}

	return nil
}

// GetLogLevel converts the debug level string to a LogLevel enum
// Returns INFO level if the string is invalid.
func (c *Config) GetLogLevel() int {
	switch strings.ToUpper(c.DebugLevel) {
	case "DEBUG":
		return int(lib.DEBUG)
	case "INFO":
		return int(lib.INFO)
	case "WARN":
		return int(lib.WARN)
	case "ERROR":
		return int(lib.ERROR)
	case "FATAL":
		return int(lib.FATAL)
	default:
		return int(lib.INFO) // Default to INFO
	}
}
