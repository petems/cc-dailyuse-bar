//go:build darwin

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"cc-dailyuse-bar/src/lib"
)

// appLogPath returns the absolute path of the application log file used by
// setupAppLogFile and read by the `logs` subcommand. Empty string means the
// home dir could not be resolved.
func appLogPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar", "stderr.log")
}

// setupAppLogFile redirects stderr (and stdout, and the structured logger)
// into ~/Library/Logs/cc-dailyuse-bar/stderr.log when running on macOS as a
// GUI .app bundle. LaunchServices wires GUI fds to /dev/null, so without
// this the structured logger has nowhere to write and Unknown-status issues
// are impossible to diagnose post-hoc.
//
// No-op when stderr is already a regular file — the LaunchAgent plist's
// StandardErrorPath already redirects, and shell users may have their own
// redirect — so we don't double-write or fight the existing target.
func setupAppLogFile() (string, error) {
	if info, err := os.Stderr.Stat(); err == nil && info.Mode().IsRegular() {
		return "", nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	dir := filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create logs dir %q: %w", dir, err)
	}

	logPath := filepath.Join(dir, "stderr.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return "", fmt.Errorf("open log file %q: %w", logPath, err)
	}

	// Dup2 onto fd 1/2 so Go runtime panics and any third-party stderr writes
	// land in the same file as the structured logger output.
	if err := syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd())); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("dup2 stderr: %w", err)
	}
	_ = syscall.Dup2(int(f.Fd()), int(os.Stdout.Fd()))

	// Future loggers (UsageService, TrayRunner) read the global default writer
	// at construction; updating it here means they all log to the same file.
	lib.SetGlobalOutput(f)
	return logPath, nil
}
