package cmd

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/lib"
)

func TestSetupLogging(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected lib.LogLevel
	}{
		{"debug level", "DEBUG", lib.DEBUG},
		{"info level", "INFO", lib.INFO},
		{"warn level", "WARN", lib.WARN},
		{"error level", "ERROR", lib.ERROR},
		{"fatal level", "FATAL", lib.FATAL},
		{"case insensitive", "debug", lib.DEBUG},
		{"mixed case", "Warn", lib.WARN},
		{"invalid defaults to INFO", "INVALID", lib.INFO},
		{"empty defaults to INFO", "", lib.INFO},
	}

	savedLogLevel := logLevel
	t.Cleanup(func() {
		logLevel = savedLogLevel
		setupLogging()
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logLevel = tt.input
			setupLogging()
			assert.Equal(t, tt.expected, lib.GetGlobalLevel())
		})
	}
}

// Errors returned from a subcommand's RunE must go to cobra's ErrOrStderr,
// not OutOrStdout — scripts parsing the CLI's stdout would break otherwise.
// Pins the policy from commit ccd28d7.
func TestRootCmd_ErrorsRouteToStderr(t *testing.T) {
	stdoutBuf := new(bytes.Buffer)
	stderrBuf := new(bytes.Buffer)

	savedFormat := showFormat
	t.Cleanup(func() {
		RootCmd.SetArgs(nil)
		RootCmd.SetOut(nil)
		RootCmd.SetErr(nil)
		showFormat = savedFormat
	})

	cfgPath := filepath.Join(t.TempDir(), "missing.yaml")
	RootCmd.SetOut(stdoutBuf)
	RootCmd.SetErr(stderrBuf)
	RootCmd.SetArgs([]string{"config", "show", "--format", "toml", "--config", cfgPath})

	err := RootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
	assert.Contains(t, stderrBuf.String(), "unsupported format",
		"command errors must be written to stderr")
	assert.NotContains(t, stdoutBuf.String(), "unsupported format",
		"command errors must not leak into stdout")
}
