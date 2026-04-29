package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
)

func TestPrintConfig_YAML(t *testing.T) {
	config := models.ConfigDefaults()
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := printConfig(cmd, config, "yaml")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "ccusage_path")
}

func TestPrintConfig_JSON(t *testing.T) {
	config := models.ConfigDefaults()
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := printConfig(cmd, config, "json")
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "CCUsagePath")
}

func TestPrintConfig_UnsupportedFormat(t *testing.T) {
	config := models.ConfigDefaults()
	cmd := &cobra.Command{}

	err := printConfig(cmd, config, "toml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestConfigValidateCmd(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	// Use a non-existent config path; Load() returns defaults for missing files
	cfgPath := filepath.Join(t.TempDir(), "nonexistent-cc-dailyuse-bar-test.yaml")
	RootCmd.SetArgs([]string{"config", "validate", "--config", cfgPath})

	err := RootCmd.Execute()
	require.NoError(t, err)
}

func TestConfigShowCmd_YAML(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	cfgPath := filepath.Join(t.TempDir(), "nonexistent-cc-dailyuse-bar-test.yaml")
	RootCmd.SetArgs([]string{"config", "show", "--config", cfgPath})

	err := RootCmd.Execute()
	require.NoError(t, err)
}

func TestConfigShowCmd_JSON(t *testing.T) {
	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	cfgPath := filepath.Join(t.TempDir(), "nonexistent-cc-dailyuse-bar-test.yaml")
	RootCmd.SetArgs([]string{"config", "show", "--format", "json", "--config", cfgPath})

	err := RootCmd.Execute()
	require.NoError(t, err)
}

// resetForceInit restores the package-level flag after each test that may
// have set --force, so test ordering can't leak state.
func resetForceInit(t *testing.T) {
	t.Helper()
	saved := forceInit
	t.Cleanup(func() {
		forceInit = saved
		RootCmd.SetArgs(nil)
	})
}

func TestConfigInitCmd_RefusesToOverwriteWithoutForce(t *testing.T) {
	resetForceInit(t)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	const sentinel = "ccusage_path: pre-existing-marker\n"
	require.NoError(t, os.WriteFile(cfgPath, []byte(sentinel), 0o644))

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetErr(buf)
	RootCmd.SetArgs([]string{"config", "init", "--config", cfgPath})

	err := RootCmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")

	// File must be untouched.
	contents, readErr := os.ReadFile(cfgPath) //nolint:gosec // cfgPath is a t.TempDir()-derived test path
	require.NoError(t, readErr)
	assert.Equal(t, sentinel, string(contents))
}

func TestConfigInitCmd_ForceOverwrites(t *testing.T) {
	resetForceInit(t)

	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(t, os.WriteFile(cfgPath, []byte("ccusage_path: pre-existing-marker\n"), 0o644))

	buf := new(bytes.Buffer)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"config", "init", "--config", cfgPath, "--force"})

	err := RootCmd.Execute()
	require.NoError(t, err)

	contents, readErr := os.ReadFile(cfgPath) //nolint:gosec // cfgPath is a t.TempDir()-derived test path
	require.NoError(t, readErr)
	// File should now hold defaults — sentinel is gone, default ccusage path is in.
	assert.NotContains(t, string(contents), "pre-existing-marker")
	assert.Contains(t, string(contents), "ccusage_path")
}
