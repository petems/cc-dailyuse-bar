//go:build !darwin

package cmd

import (
	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/lib"
)

var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Manage the macOS LaunchAgent (darwin only)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return lib.NewError(lib.ErrCodeSystem,
			"`service` is darwin-only; this binary was built for a non-darwin target")
	},
}

func init() {
	RootCmd.AddCommand(serviceCmd)
}
