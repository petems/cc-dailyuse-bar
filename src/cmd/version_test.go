package cmd

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionCmd_Default(t *testing.T) {
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)

	// Reset flags to avoid leakage
	shortVersion = false
	jsonVersion = false

	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"version"})

	err := RootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "CC Daily Use Bar")
	assert.Contains(t, buf.String(), Version)
}

func TestVersionCmd_Short(t *testing.T) {
	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)

	// Reset flags
	shortVersion = false
	jsonVersion = false

	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"version", "--short"})

	err := RootCmd.Execute()
	require.NoError(t, err)
	assert.Contains(t, buf.String(), Version)
}

func TestVersionCmd_JSON(t *testing.T) {
	// Reset flags
	shortVersion = false
	jsonVersion = false

	buf := new(bytes.Buffer)
	versionCmd.SetOut(buf)
	RootCmd.SetOut(buf)
	RootCmd.SetArgs([]string{"version", "--json"})

	err := RootCmd.Execute()
	require.NoError(t, err)

	// Parse the actual command output as JSON
	var info map[string]string
	err = json.Unmarshal(buf.Bytes(), &info)
	require.NoError(t, err, "version --json should produce valid JSON, got: %s", buf.String())

	assert.Equal(t, Version, info["version"])
	assert.Equal(t, Commit, info["commit"])
	assert.Equal(t, Date, info["date"])
	assert.NotEmpty(t, info["go_version"])
	assert.NotEmpty(t, info["os"])
	assert.NotEmpty(t, info["arch"])
}

func TestVersionVars_Defaults(t *testing.T) {
	assert.Equal(t, "dev", Version)
	assert.Equal(t, "none", Commit)
	assert.Equal(t, "unknown", Date)
}
