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
		return "", lib.WrapError(err, lib.ErrCodeSystem, "resolve home dir")
	}

	dir := filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar")
	// #nosec G301 -- 0o755 lets the user read logs from Finder/Terminal without sudo; matches service.go convention
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return "", lib.WrapError(mkErr, lib.ErrCodeSystem, fmt.Sprintf("create logs dir %q", dir))
	}

	logPath := filepath.Join(dir, "stderr.log")
	// #nosec G304 -- logPath is derived from UserHomeDir + fixed app subpath, not user input
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return "", lib.WrapError(err, lib.ErrCodeSystem, fmt.Sprintf("open log file %q", logPath))
	}

	// Dup2 onto fd 1/2 so Go runtime panics and any third-party stderr writes
	// land in the same file as the structured logger output. Treat both fds
	// as mandatory: if stdout redirect fails we'd otherwise return success
	// but leave the global logger pinned to a half-redirected file.
	if err := syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd())); err != nil {
		_ = f.Close()
		return "", lib.WrapError(err, lib.ErrCodeSystem, "dup2 stderr")
	}
	if err := syscall.Dup2(int(f.Fd()), int(os.Stdout.Fd())); err != nil {
		_ = f.Close()
		return "", lib.WrapError(err, lib.ErrCodeSystem, "dup2 stdout")
	}

	// Future loggers (UsageService, TrayRunner) read the global default writer
	// at construction; updating it here means they all log to the same file.
	lib.SetGlobalOutput(f)
	return logPath, nil
}
