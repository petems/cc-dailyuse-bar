package cmd

import (
	"fmt"
	"os"
	"os/exec"
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

		// 2. Binary Check
		path, err := exec.LookPath(config.CCUsagePath)
		if err != nil {
			return fmt.Errorf("binary: 'ccusage' not found at %q; install ccusage or update 'ccusage_path' in config", config.CCUsagePath)
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

		// 3. Connectivity Check (One-shot poll)
		fmt.Fprintln(cmd.OutOrStdout(), "Connectivity: Testing API connection...")
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
