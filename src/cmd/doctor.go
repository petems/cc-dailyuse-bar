package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Aliases: []string{"status"},
	Short:   "Check the health of the application and dependencies",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "Running health checks...")
		hasWarnings := false

		svc := services.NewConfigService()
		if cfgFile != "" {
			svc.SetConfigPath(cfgFile)
		}

		// 1. Config Check
		config, err := svc.Load()
		if err != nil {
			return fmt.Errorf("config: failed to load configuration from %q; fix the file or run 'cc-dailyuse-bar config init --force' to reset to defaults: %w",
				svc.GetConfigPath(), err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Config: Valid (loaded from %s)\n", svc.GetConfigPath())

		// 2. Binary Check — uses the same resolver the running tray uses
		// (PATH lookup, with macOS Homebrew fallback for bare names) so
		// `doctor` matches runtime behaviour. Without this, the tray could
		// fail when launched from /Applications under launchd's stripped
		// PATH while doctor (run from a shell) reports everything fine.
		path, fallback, err := services.ResolveCCUsagePath(config.CCUsagePath)
		if err != nil {
			return fmt.Errorf("binary: 'ccusage' not found at %q; install ccusage or set 'ccusage_path' to an absolute path in config: %w", config.CCUsagePath, err)
		}

		// On non-Windows, verify the file is executable via permission bits.
		// On Windows, executability is determined by file extension and PATHEXT,
		// so LookPath success is sufficient.
		if runtime.GOOS != "windows" {
			info, statErr := os.Stat(path)
			if statErr != nil {
				return fmt.Errorf("binary: '%s' is not accessible: %w", path, statErr)
			}
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("binary: '%s' is not executable", path)
			}
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Binary: Found at '%s'\n", path)
		if fallback {
			fmt.Fprintf(cmd.OutOrStdout(),
				"Binary: WARNING — resolved via Homebrew fallback because %q is not on PATH.\n"+
					"        macOS .app bundles launched from /Applications run with a minimal PATH\n"+
					"        (no /opt/homebrew/bin), so set 'ccusage_path: %s' in config to be safe.\n",
				config.CCUsagePath, path)
			hasWarnings = true
		}

		// 3. Connectivity Check (One-shot poll)
		fmt.Fprintf(cmd.OutOrStdout(), "Connectivity: Testing API connection (timeout: %ds)...\n", config.CmdTimeout)
		usageService := services.NewUsageService(config)

		state, err := usageService.UpdateUsage()
		if err != nil {
			return fmt.Errorf("connectivity: failed to fetch usage data: %w", err)
		}

		if state.Status == models.Unknown && !state.IsAvailable {
			fmt.Fprintln(cmd.OutOrStdout(), "Connectivity: Data unavailable (API returned no data)")
			hasWarnings = true
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Connectivity: Success! (Cost: $%.2f, Count: %d)\n", state.DailyCost, state.DailyCount)
		}

		if hasWarnings {
			fmt.Fprintln(cmd.OutOrStdout(), "\nSome checks had warnings.")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "\nAll checks passed!")
		}
		return nil
	},
}

func init() {
	RootCmd.AddCommand(doctorCmd)
}
