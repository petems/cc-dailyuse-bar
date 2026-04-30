//go:build darwin

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEmbeddedPlistMatchesRepoRoot guards against drift between the plist
// the Go binary embeds and the one the Makefile install-service-macos
// target reads from the repo root. They must stay byte-identical so the two
// install paths produce the same LaunchAgent.
func TestEmbeddedPlistMatchesRepoRoot(t *testing.T) {
	repoRoot, err := os.ReadFile(filepath.Join("..", "..", "com.cc-dailyuse-bar.plist"))
	require.NoError(t, err, "could not read repo-root plist")
	assert.Equal(t, string(repoRoot), launchAgentPlist,
		"src/cmd/templates/launchagent.plist drifted from com.cc-dailyuse-bar.plist; sync them")
}

func TestRenderLaunchAgent_SubstitutesTokens(t *testing.T) {
	rendered := renderLaunchAgent("/Users/alice", "/Applications/CC Daily Use Bar.app/Contents/MacOS/cc-dailyuse-bar")

	assert.NotContains(t, rendered, "__HOME__", "__HOME__ token should be fully substituted")
	assert.NotContains(t, rendered, "__CC_DAILYUSE_BAR_BIN__", "__CC_DAILYUSE_BAR_BIN__ token should be fully substituted")
	assert.Contains(t, rendered, "/Users/alice/Library/Logs/cc-dailyuse-bar/stdout.log")
	assert.Contains(t, rendered, "/Users/alice/Library/Logs/cc-dailyuse-bar/stderr.log")
	assert.Contains(t, rendered, "/Applications/CC Daily Use Bar.app/Contents/MacOS/cc-dailyuse-bar")
	assert.Contains(t, rendered, "<string>com.cc-dailyuse-bar</string>", "Label should be preserved")
}

func TestResolveBinPath_OverrideTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "my-binary")
	require.NoError(t, os.WriteFile(target, []byte("#!/bin/sh\n"), 0o755))

	got, err := resolveBinPath(target)
	require.NoError(t, err)
	assert.Equal(t, target, got)
}

func TestResolveBinPath_OverrideMakesAbsolute(t *testing.T) {
	got, err := resolveBinPath("./relative/path")
	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(got), "expected absolute path, got %q", got)
}

func TestResolveBinPath_DefaultResolvesSymlinks(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real-binary")
	require.NoError(t, os.WriteFile(real, []byte{}, 0o755))
	link := filepath.Join(dir, "symlink-to-binary")
	require.NoError(t, os.Symlink(real, link))

	// macOS's /var is itself a symlink to /private/var, so resolve both
	// sides through EvalSymlinks before comparing.
	got, err := filepath.EvalSymlinks(link)
	require.NoError(t, err)
	wantResolved, err := filepath.EvalSymlinks(real)
	require.NoError(t, err)
	assert.Equal(t, wantResolved, got, "EvalSymlinks should walk to the real binary")
}

func TestLaunchAgentPath(t *testing.T) {
	got := launchAgentPath("/Users/alice")
	assert.Equal(t, "/Users/alice/Library/LaunchAgents/com.cc-dailyuse-bar.plist", got)
}

func TestRunServiceInstall_WritesPlistAndCallsLaunchctl(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Capture launchctl invocations without shelling out.
	var calls [][]string
	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	t.Cleanup(func() { execLaunchctl = prev })

	buf := new(bytes.Buffer)
	cmd := serviceInstallCmd
	cmd.SetOut(buf)

	err := runServiceInstall(cmd, "/path/to/cc-dailyuse-bar")
	require.NoError(t, err)

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.cc-dailyuse-bar.plist")
	body, err := os.ReadFile(plistPath)
	require.NoError(t, err)
	assert.Contains(t, string(body), "/path/to/cc-dailyuse-bar")
	assert.Contains(t, string(body), home+"/Library/Logs/cc-dailyuse-bar/stdout.log")
	assert.NotContains(t, string(body), "__HOME__")
	assert.NotContains(t, string(body), "__CC_DAILYUSE_BAR_BIN__")

	logsDirPath := filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar")
	stat, err := os.Stat(logsDirPath)
	require.NoError(t, err, "logs dir should be created")
	assert.True(t, stat.IsDir())

	require.Len(t, calls, 2, "expected unload + load")
	assert.Equal(t, []string{"unload", plistPath}, calls[0])
	assert.Equal(t, []string{"load", plistPath}, calls[1])

	output := buf.String()
	assert.Contains(t, output, plistPath)
	assert.Contains(t, output, "/path/to/cc-dailyuse-bar")
}

func TestRunServiceInstall_IsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { execLaunchctl = prev })

	cmd := serviceInstallCmd
	cmd.SetOut(new(bytes.Buffer))

	require.NoError(t, runServiceInstall(cmd, "/bin1"))
	require.NoError(t, runServiceInstall(cmd, "/bin2"), "second install should not error")

	body, err := os.ReadFile(filepath.Join(home, "Library", "LaunchAgents", "com.cc-dailyuse-bar.plist"))
	require.NoError(t, err)
	assert.Contains(t, string(body), "/bin2", "second install should overwrite the first")
}

func TestRunServiceUninstall_RemovesPlistAndKeepsLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.cc-dailyuse-bar.plist")
	require.NoError(t, os.MkdirAll(filepath.Dir(plistPath), 0o755))
	require.NoError(t, os.WriteFile(plistPath, []byte("seed"), 0o644))

	logs := filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar")
	require.NoError(t, os.MkdirAll(logs, 0o755))
	logFile := filepath.Join(logs, "stdout.log")
	require.NoError(t, os.WriteFile(logFile, []byte("hi"), 0o644))

	var calls [][]string
	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) {
		calls = append(calls, args)
		return nil, nil
	}
	t.Cleanup(func() { execLaunchctl = prev })

	cmd := serviceUninstallCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	require.NoError(t, runServiceUninstall(cmd, false))

	_, err := os.Stat(plistPath)
	assert.True(t, os.IsNotExist(err), "plist should be removed")
	_, err = os.Stat(logFile)
	assert.NoError(t, err, "logs should be preserved by default")

	require.Len(t, calls, 1)
	assert.Equal(t, []string{"unload", plistPath}, calls[0])
	assert.Contains(t, buf.String(), "Logs preserved")
}

func TestRunServiceUninstall_PurgeLogs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	logs := filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar")
	require.NoError(t, os.MkdirAll(logs, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(logs, "stdout.log"), []byte("hi"), 0o644))

	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { execLaunchctl = prev })

	cmd := serviceUninstallCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	require.NoError(t, runServiceUninstall(cmd, true))

	_, err := os.Stat(logs)
	assert.True(t, os.IsNotExist(err), "logs dir should be purged")
	assert.Contains(t, buf.String(), "Logs purged")
}

func TestRunServiceUninstall_NoStateIsFine(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { execLaunchctl = prev })

	cmd := serviceUninstallCmd
	cmd.SetOut(new(bytes.Buffer))
	assert.NoError(t, runServiceUninstall(cmd, false), "uninstall with nothing installed should be a no-op")
}

func TestRunServiceStatus_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) { return nil, nil }
	t.Cleanup(func() { execLaunchctl = prev })

	buf := new(bytes.Buffer)
	cmd := serviceStatusCmd
	cmd.SetOut(buf)
	require.NoError(t, runServiceStatus(cmd))
	assert.Contains(t, strings.ToLower(buf.String()), "not installed")
}

func TestRunServiceStatus_LoadedReportsLaunchctlOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	plistPath := filepath.Join(home, "Library", "LaunchAgents", "com.cc-dailyuse-bar.plist")
	require.NoError(t, os.MkdirAll(filepath.Dir(plistPath), 0o755))
	require.NoError(t, os.WriteFile(plistPath, []byte("seed"), 0o644))

	prev := execLaunchctl
	execLaunchctl = func(args ...string) ([]byte, error) {
		return []byte(`{
	"PID" = 12345;
	"Label" = "com.cc-dailyuse-bar";
};`), nil
	}
	t.Cleanup(func() { execLaunchctl = prev })

	buf := new(bytes.Buffer)
	cmd := serviceStatusCmd
	cmd.SetOut(buf)
	require.NoError(t, runServiceStatus(cmd))
	assert.Contains(t, buf.String(), "loaded")
	assert.Contains(t, buf.String(), "12345")
}
