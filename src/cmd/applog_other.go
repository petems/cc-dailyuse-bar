//go:build !darwin

package cmd

// setupAppLogFile is a no-op outside macOS. The launchd-induced "logs vanish
// when the .app is launched from /Applications" problem this addresses is
// macOS-specific.
func setupAppLogFile() (string, error) {
	return "", nil
}

// appLogPath returns "" on non-darwin: the app log is macOS-only because the
// LaunchAgent and LaunchServices flows that motivate it are macOS-only.
func appLogPath() string {
	return ""
}
