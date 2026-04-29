package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

var daemonMode bool

var logger = lib.NewLogger("cmd-run")

// runTrayApp is set by the platform-specific run_tray.go file.
// It is nil when built with the "nogui" tag.
var runTrayApp func(cmd *cobra.Command, config *models.Config) error

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Launch the system tray application",
	Long: `Start the CC Daily Use Bar in the system tray.
This is the default mode if no command is specified.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Validate the parent process before forking a daemon — otherwise the
		// parent prints a success PID even when the child is guaranteed to fail
		// (no GUI build, bad config, invalid flags).
		if runTrayApp == nil {
			return lib.NewError(lib.ErrCodeSystem, "this binary was built without GUI support (use a build without the 'nogui' tag)")
		}

		configService := services.NewConfigService()
		if cfgFile != "" {
			configService.SetConfigPath(cfgFile)
		}

		// Load() already returns ConfigDefaults for a missing file; any error
		// here is a real failure (parse, permissions, validation). Don't mask it.
		config, err := configService.Load()
		if err != nil {
			return lib.WrapError(err, lib.ErrCodeConfig,
				fmt.Sprintf("failed to load configuration from %q; fix the file or run 'cc-dailyuse-bar config init --force' to reset to defaults",
					configService.GetConfigPath()))
		}

		if err := mergeConfig(config, cmd); err != nil {
			return lib.WrapError(err, lib.ErrCodeValidation, "invalid configuration after flag overrides")
		}

		if daemonMode {
			return runAsDaemon(cmd)
		}

		return runTrayApp(cmd, config)
	},
}

func init() {
	RootCmd.AddCommand(runCmd)

	// Local flags for run command
	runCmd.Flags().BoolVarP(&daemonMode, "daemon", "d", false, "Run as daemon (background process)")
	runCmd.Flags().Int("update-interval", 0, "Update interval in seconds")
	runCmd.Flags().Float64("yellow-threshold", 0, "Yellow alert threshold ($)")
	runCmd.Flags().Float64("red-threshold", 0, "Red alert threshold ($)")
	runCmd.Flags().String("ccusage-path", "", "Path to ccusage binary")
	runCmd.Flags().Int("cache-window", 0, "Cache window in seconds")
	runCmd.Flags().Int("cmd-timeout", 0, "Command timeout in seconds")
}

func mergeConfig(config *models.Config, cmd *cobra.Command) error {
	flags := cmd.Flags()

	if flags.Changed("update-interval") {
		v, _ := flags.GetInt("update-interval")
		config.UpdateInterval = v
	}
	if flags.Changed("yellow-threshold") {
		v, _ := flags.GetFloat64("yellow-threshold")
		config.YellowThreshold = v
	}
	if flags.Changed("red-threshold") {
		v, _ := flags.GetFloat64("red-threshold")
		config.RedThreshold = v
	}
	if flags.Changed("ccusage-path") {
		v, _ := flags.GetString("ccusage-path")
		config.CCUsagePath = v
	}
	if flags.Changed("cache-window") {
		v, _ := flags.GetInt("cache-window")
		config.CacheWindow = v
	}
	if flags.Changed("cmd-timeout") {
		v, _ := flags.GetInt("cmd-timeout")
		config.CmdTimeout = v
	}

	return config.Validate()
}

// buildDaemonArgs constructs the argument list for the daemon subprocess,
// stripping --daemon/-d flags and ensuring "run" is at the subcommand position.
// Bare "run" tokens that appear later in the list (e.g. as a flag value like
// `--config run`) are preserved verbatim.
func buildDaemonArgs(osArgs []string) []string {
	args := make([]string, 0, len(osArgs))

	for i := 1; i < len(osArgs); i++ {
		arg := osArgs[i]

		// Skip --daemon / -d in all forms
		if arg == "--daemon" || arg == "-d" {
			continue
		}
		if strings.HasPrefix(arg, "--daemon=") {
			continue
		}

		args = append(args, arg)
	}

	if len(args) == 0 || args[0] != "run" {
		args = append([]string{"run"}, args...)
	}

	return args
}

func runAsDaemon(cmd *cobra.Command) error {
	execPath, err := os.Executable()
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to get executable path")
	}

	args := buildDaemonArgs(os.Args)

	child := exec.Command(execPath, args...)
	// Detach the child from the parent's terminal — leaving Stdout/Stderr wired
	// up means closing the terminal sends SIGHUP to the daemon. Discarding to
	// /dev/null lets the parent exit cleanly without dragging the child down.
	devNull, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, "failed to open /dev/null")
	}
	child.Stdin = devNull
	child.Stdout = devNull
	child.Stderr = devNull

	startErr := child.Start()
	// The child has dup'd its own fds; the parent's devNull is no longer needed.
	if cerr := devNull.Close(); cerr != nil && startErr == nil {
		startErr = cerr
	}
	if startErr != nil {
		return lib.WrapError(startErr, lib.ErrCodeSystem, "failed to start daemon")
	}

	// Write through the cobra command so callers using cmd.SetOut() (tests) can
	// capture this output, and so deferred cleanup in the caller still runs.
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "CC Daily Use Bar started as daemon (PID: %d)\n", child.Process.Pid)
	fmt.Fprintf(out, "To stop: kill %d\n", child.Process.Pid)

	return nil
}
