package cmd

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Version is the application version, set by -ldflags during build.
	Version = "dev"
	// Commit is the git commit hash, set by -ldflags during build.
	Commit = "none"
	// Date is the build date, set by -ldflags during build.
	Date = "unknown"
)

var (
	shortVersion bool
	jsonVersion  bool
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version info",
	RunE: func(cmd *cobra.Command, args []string) error {
		w := cmd.OutOrStdout()

		if shortVersion {
			fmt.Fprintln(w, Version)
			return nil
		}

		if jsonVersion {
			info := map[string]string{
				"version":    Version,
				"commit":     Commit,
				"date":       Date,
				"go_version": runtime.Version(),
				"os":         runtime.GOOS,
				"arch":       runtime.GOARCH,
			}
			data, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal version info: %w", err)
			}
			fmt.Fprintln(w, string(data))
			return nil
		}

		fmt.Fprintf(w, "CC Daily Use Bar %s\n", Version)
		fmt.Fprintf(w, "Commit: %s\n", Commit)
		fmt.Fprintf(w, "Date:   %s\n", Date)
		fmt.Fprintf(w, "Go:     %s %s/%s\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	RootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolVarP(&shortVersion, "short", "s", false, "Print only the version string")
	versionCmd.Flags().BoolVarP(&jsonVersion, "json", "j", false, "Print version info as JSON")
}
