//go:build darwin

package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/lib"
)

//go:embed templates/launchagent.plist
var launchAgentPlist string

const (
	launchAgentLabel    = "com.cc-dailyuse-bar"
	launchAgentFileName = "com.cc-dailyuse-bar.plist"
)

// execLaunchctl is overridable in tests so we can assert calls without
// shelling out to a real launchctl binary.
var execLaunchctl = func(args ...string) ([]byte, error) {
	return exec.Command("launchctl", args...).CombinedOutput()
}

var (
	serviceBinPath    string
	servicePurgeLogs  bool
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the macOS LaunchAgent for autostart at login",
	Long: `Install, uninstall, or check the status of the user-level
LaunchAgent that starts cc-dailyuse-bar automatically when you log in.

The LaunchAgent runs as your user (not root) and respects the menu bar /
LSUIElement bundle config — no Dock icon. Logs go to
~/Library/Logs/cc-dailyuse-bar/.`,
}

var serviceInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and load the LaunchAgent",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServiceInstall(cmd, serviceBinPath)
	},
}

var serviceUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Unload the LaunchAgent and remove its plist",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServiceUninstall(cmd, servicePurgeLogs)
	},
}

var serviceStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Report whether the LaunchAgent is loaded",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runServiceStatus(cmd)
	},
}

func init() {
	serviceInstallCmd.Flags().StringVar(&serviceBinPath, "bin-path", "",
		"override the binary path written into the LaunchAgent (default: resolved from os.Executable)")
	serviceUninstallCmd.Flags().BoolVar(&servicePurgeLogs, "purge-logs", false,
		"also delete ~/Library/Logs/cc-dailyuse-bar (logs are kept by default)")

	serviceCmd.AddCommand(serviceInstallCmd, serviceUninstallCmd, serviceStatusCmd)
	RootCmd.AddCommand(serviceCmd)
}

// resolveBinPath returns the absolute, symlink-resolved path that should be
// written into the LaunchAgent's ProgramArguments. When --bin-path is empty,
// this is the running binary's resolved path (which for cask installs walks
// Homebrew's bin shim back to /Applications/CC Daily Use Bar.app/.../cc-dailyuse-bar).
func resolveBinPath(override string) (string, error) {
	if override != "" {
		abs, err := filepath.Abs(override)
		if err != nil {
			return "", lib.WrapError(err, lib.ErrCodeSystem, "failed to resolve --bin-path")
		}
		return abs, nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", lib.WrapError(err, lib.ErrCodeSystem, "failed to get executable path")
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		// Fall back to the unresolved path; better than failing outright.
		return exe, nil
	}
	return resolved, nil
}

// renderLaunchAgent substitutes the __HOME__ and __CC_DAILYUSE_BAR_BIN__
// tokens in the embedded plist template. Kept pure so tests can pin the
// substitution behaviour without filesystem touches.
func renderLaunchAgent(home, binPath string) string {
	return strings.NewReplacer(
		"__HOME__", home,
		"__CC_DAILYUSE_BAR_BIN__", binPath,
	).Replace(launchAgentPlist)
}

func launchAgentPath(home string) string {
	return filepath.Join(home, "Library", "LaunchAgents", launchAgentFileName)
}

func logsDir(home string) string {
	return filepath.Join(home, "Library", "Logs", "cc-dailyuse-bar")
}

func runServiceInstall(cmd *cobra.Command, binOverride string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to resolve home directory")
	}

	binPath, err := resolveBinPath(binOverride)
	if err != nil {
		return err
	}

	plistPath := launchAgentPath(home)
	logs := logsDir(home)

	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to create LaunchAgents directory")
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to create logs directory")
	}

	rendered := renderLaunchAgent(home, binPath)

	tmp := plistPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(rendered), 0o644); err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to write plist tempfile")
	}
	if err := os.Rename(tmp, plistPath); err != nil {
		_ = os.Remove(tmp)
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to install plist")
	}

	// Best-effort unload first so re-running install doesn't leave stale state.
	_, _ = execLaunchctl("unload", plistPath)
	if out, err := execLaunchctl("load", plistPath); err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem,
			fmt.Sprintf("launchctl load failed: %s", strings.TrimSpace(string(out))))
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "LaunchAgent installed: %s\n", plistPath)
	fmt.Fprintf(w, "Binary:                %s\n", binPath)
	fmt.Fprintf(w, "Logs:                  %s\n", logs)
	fmt.Fprintln(w, "Disable autostart with `cc-dailyuse-bar service uninstall`.")
	return nil
}

func runServiceUninstall(cmd *cobra.Command, purgeLogs bool) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to resolve home directory")
	}

	plistPath := launchAgentPath(home)

	// launchctl unload is idempotent enough — swallow non-zero (already unloaded).
	_, _ = execLaunchctl("unload", plistPath)

	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to remove plist")
	}

	w := cmd.OutOrStdout()
	fmt.Fprintf(w, "LaunchAgent removed:   %s\n", plistPath)

	if purgeLogs {
		logs := logsDir(home)
		if err := os.RemoveAll(logs); err != nil {
			return lib.WrapError(err, lib.ErrCodeSystem, "failed to purge logs directory")
		}
		fmt.Fprintf(w, "Logs purged:           %s\n", logs)
	} else {
		fmt.Fprintln(w, "Logs preserved (pass --purge-logs to delete them too).")
	}
	return nil
}

func runServiceStatus(cmd *cobra.Command) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to resolve home directory")
	}

	plistPath := launchAgentPath(home)
	w := cmd.OutOrStdout()

	if _, err := os.Stat(plistPath); os.IsNotExist(err) {
		fmt.Fprintf(w, "Not installed (no plist at %s)\n", plistPath)
		return nil
	} else if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to stat plist")
	}

	out, err := execLaunchctl("list", launchAgentLabel)
	if err != nil {
		fmt.Fprintf(w, "Plist present at %s but not loaded\n", plistPath)
		fmt.Fprintln(w, "Run `cc-dailyuse-bar service install` to load it.")
		return nil
	}
	fmt.Fprintf(w, "LaunchAgent loaded: %s\n", plistPath)
	fmt.Fprintln(w, strings.TrimRight(string(out), "\n"))
	return nil
}
