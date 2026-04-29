//go:build !nogui

package cmd

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/getlantern/systray"
	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/internal/tray"
	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

func init() {
	runTrayApp = startTrayApp
}

func startTrayApp(cmd *cobra.Command, config *models.Config) error {
	// Initialize Usage Service
	usageService := services.NewUsageService(config)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received signal, shutting down gracefully", map[string]interface{}{
			"signal": sig.String(),
		})
		usageService.StopPolling()
		systray.Quit()
	}()

	// Initialize Tray Runner
	runner := tray.NewRunner(config, usageService)

	// Start the application (blocks until exit)
	runner.Run()
	return nil
}
