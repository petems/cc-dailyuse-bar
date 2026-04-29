package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCmd_InvalidConfig(t *testing.T) {
	// Create a temp file with invalid YAML config content
	tmpDir := t.TempDir()
	badConfig := filepath.Join(tmpDir, "config.yaml")
	err := os.WriteFile(badConfig, []byte("update_interval: -5\nyellow_threshold: -1\n"), 0644)
	require.NoError(t, err)

	savedCfgFile := cfgFile
	t.Cleanup(func() {
		cfgFile = savedCfgFile
		RootCmd.SetArgs(nil)
	})

	RootCmd.SetArgs([]string{"doctor", "--config", badConfig})
	err = RootCmd.Execute()
	// Doctor should fail because config validation will fail
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VALIDATION_ERROR")
}

func TestDoctorCmd_ShortDescription(t *testing.T) {
	assert.Contains(t, doctorCmd.Short, "health")
}

func TestDoctorCmd_Registration(t *testing.T) {
	assert.NotNil(t, doctorCmd)
	assert.Equal(t, "doctor", doctorCmd.Use)
	assert.Contains(t, doctorCmd.Aliases, "status")
}

// writeBinaryConfig writes a complete, validation-passing config YAML
// pointing ccusage_path at the given binary location.
func writeBinaryConfig(t *testing.T, dir, binaryPath string) string {
	t.Helper()
	cfgPath := filepath.Join(dir, "config.yaml")
	body := fmt.Sprintf(`ccusage_path: %q
update_interval: 30
yellow_threshold: 10.0
red_threshold: 20.0
debug_level: INFO
cache_window: 10
cmd_timeout: 5
`, binaryPath)
	require.NoError(t, os.WriteFile(cfgPath, []byte(body), 0o644))
	return cfgPath
}

func TestDoctorCmd_BinaryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	missing := filepath.Join(tmpDir, "definitely-not-here")
	cfgPath := writeBinaryConfig(t, tmpDir, missing)

	savedCfgFile := cfgFile
	t.Cleanup(func() {
		cfgFile = savedCfgFile
		RootCmd.SetArgs(nil)
	})

	RootCmd.SetArgs([]string{"doctor", "--config", cfgPath})
	err := RootCmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "binary")
	assert.Contains(t, err.Error(), "not found")
}

func TestDoctorCmd_BinaryNotExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows determines executability via PATHEXT, not Unix mode bits")
	}

	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "ccusage")
	require.NoError(t, os.WriteFile(binPath, []byte("#!/bin/sh\nexit 0\n"), 0o644))
	cfgPath := writeBinaryConfig(t, tmpDir, binPath)

	savedCfgFile := cfgFile
	t.Cleanup(func() {
		cfgFile = savedCfgFile
		RootCmd.SetArgs(nil)
	})

	RootCmd.SetArgs([]string{"doctor", "--config", cfgPath})
	err := RootCmd.Execute()

	require.Error(t, err)
	// exec.LookPath rejects non-executable absolute paths with "not found";
	// the explicit mode-bits check in doctor.go is a defense-in-depth fallback.
	assert.Contains(t, err.Error(), "binary")
}
