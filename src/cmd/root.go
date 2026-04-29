package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/lib"
)

var (
	cfgFile  string
	logLevel string
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "cc-dailyuse-bar",
	Short: "A menu bar app for monitoring Anthropic Claude Code usage",
	Long: `CC Daily Use Bar is a system tray application that monitors your
Anthropic Claude Code usage and costs in real-time.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		setupLogging()
	},
	// Default to run command when no subcommand is specified
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCmd.RunE(runCmd, args)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the RootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	RootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $XDG_CONFIG_HOME/cc-dailyuse-bar/config.yaml)")
	RootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "INFO", "log level (DEBUG, INFO, WARN, ERROR, FATAL)")
}

func setupLogging() {
	var level lib.LogLevel
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		level = lib.DEBUG
	case "INFO":
		level = lib.INFO
	case "WARN":
		level = lib.WARN
	case "ERROR":
		level = lib.ERROR
	case "FATAL":
		level = lib.FATAL
	default:
		level = lib.INFO
	}
	lib.SetGlobalLevel(level)
}
