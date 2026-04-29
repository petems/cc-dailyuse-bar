package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
)

func TestMergeConfig_NoFlagsChanged(t *testing.T) {
	config := models.ConfigDefaults()
	original := *config

	cmd := &cobra.Command{}
	cmd.Flags().Int("update-interval", 0, "")
	cmd.Flags().Float64("yellow-threshold", 0, "")
	cmd.Flags().Float64("red-threshold", 0, "")
	cmd.Flags().String("ccusage-path", "", "")
	cmd.Flags().Int("cache-window", 0, "")
	cmd.Flags().Int("cmd-timeout", 0, "")

	err := mergeConfig(config, cmd)
	require.NoError(t, err)
	assert.Equal(t, original, *config)
}

func TestMergeConfig_OverridesApplied(t *testing.T) {
	config := models.ConfigDefaults()

	cmd := &cobra.Command{}
	cmd.Flags().Int("update-interval", 0, "")
	cmd.Flags().Float64("yellow-threshold", 0, "")
	cmd.Flags().Float64("red-threshold", 0, "")
	cmd.Flags().String("ccusage-path", "", "")
	cmd.Flags().Int("cache-window", 0, "")
	cmd.Flags().Int("cmd-timeout", 0, "")

	// Simulate flags being set via command line
	require.NoError(t, cmd.Flags().Set("update-interval", "60"))
	require.NoError(t, cmd.Flags().Set("yellow-threshold", "15.0"))
	require.NoError(t, cmd.Flags().Set("red-threshold", "25.0"))

	err := mergeConfig(config, cmd)
	require.NoError(t, err)

	assert.Equal(t, 60, config.UpdateInterval)
	assert.Equal(t, 15.0, config.YellowThreshold)
	assert.Equal(t, 25.0, config.RedThreshold)
}

func TestMergeConfig_ValidationFailure(t *testing.T) {
	config := models.ConfigDefaults()

	cmd := &cobra.Command{}
	cmd.Flags().Int("update-interval", 0, "")
	cmd.Flags().Float64("yellow-threshold", 0, "")
	cmd.Flags().Float64("red-threshold", 0, "")
	cmd.Flags().String("ccusage-path", "", "")
	cmd.Flags().Int("cache-window", 0, "")
	cmd.Flags().Int("cmd-timeout", 0, "")

	// Set an invalid interval (below minimum of 10)
	require.NoError(t, cmd.Flags().Set("update-interval", "1"))

	err := mergeConfig(config, cmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "update_interval")
}

func TestBuildDaemonArgs_Basic(t *testing.T) {
	tests := []struct {
		name     string
		osArgs   []string
		expected []string
	}{
		{
			name:     "strips --daemon",
			osArgs:   []string{"./app", "run", "--daemon"},
			expected: []string{"run"},
		},
		{
			name:     "strips -d",
			osArgs:   []string{"./app", "run", "-d"},
			expected: []string{"run"},
		},
		{
			name:     "strips --daemon=true",
			osArgs:   []string{"./app", "run", "--daemon=true"},
			expected: []string{"run"},
		},
		{
			name:     "strips --daemon=false",
			osArgs:   []string{"./app", "run", "--daemon=false"},
			expected: []string{"run"},
		},
		{
			name:     "preserves other flags",
			osArgs:   []string{"./app", "run", "--daemon", "--update-interval", "60"},
			expected: []string{"run", "--update-interval", "60"},
		},
		{
			name:     "adds run if not present",
			osArgs:   []string{"./app", "--daemon"},
			expected: []string{"run"},
		},
		{
			name:     "avoids duplicate run",
			osArgs:   []string{"./app", "run", "--daemon"},
			expected: []string{"run"},
		},
		{
			name:     "preserves config flag",
			osArgs:   []string{"./app", "run", "-d", "--config", "/tmp/config.yaml"},
			expected: []string{"run", "--config", "/tmp/config.yaml"},
		},
		{
			// Regression: a flag value that happens to equal "run" must not be stripped.
			name:     "preserves run as a flag value",
			osArgs:   []string{"./app", "--config", "run", "--daemon"},
			expected: []string{"run", "--config", "run"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildDaemonArgs(tt.osArgs)
			assert.Equal(t, tt.expected, result)
		})
	}
}
