package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

var (
	forceInit  bool
	showFormat string
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration",
	Long:  `Inspect, validate, and initialize the configuration file.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	Long:  `Create a default configuration file at the standard XDG location.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := services.NewConfigService()
		if cfgFile != "" {
			svc.SetConfigPath(cfgFile)
		}

		path := svc.GetConfigPath()

		// Check if file exists
		if _, err := os.Stat(path); err == nil && !forceInit {
			return fmt.Errorf("config file already exists at %s (use --force to overwrite)", path)
		}

		defaults := models.ConfigDefaults()
		if err := svc.Save(defaults); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Configuration initialized at %s\n", path)
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := services.NewConfigService()
		if cfgFile != "" {
			svc.SetConfigPath(cfgFile)
		}

		config, err := svc.Load()
		if err != nil {
			return fmt.Errorf("configuration is invalid: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "✅ Configuration at %s is valid.\n", svc.GetConfigPath())
		fmt.Fprintln(cmd.OutOrStdout(), "Current values:")
		return printConfig(cmd, config, "yaml")
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show the effective configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		svc := services.NewConfigService()
		if cfgFile != "" {
			svc.SetConfigPath(cfgFile)
		}

		config, err := svc.Load()
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		return printConfig(cmd, config, showFormat)
	},
}

func init() {
	RootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)

	configInitCmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing config")
	configShowCmd.Flags().StringVar(&showFormat, "format", "yaml", "Output format (yaml or json)")
}

func printConfig(cmd *cobra.Command, config *models.Config, format string) error {
	var data []byte
	var err error

	switch format {
	case "json":
		data, err = json.MarshalIndent(config, "", "  ")
	case "yaml":
		data, err = yaml.Marshal(config)
	default:
		return fmt.Errorf("unsupported format %q (use yaml or json)", format)
	}

	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}
