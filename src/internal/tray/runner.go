package tray

import (
	"fmt"
	"time"

	"github.com/getlantern/systray"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

// Runner handles the system tray UI and logic
type Runner struct {
	config       *models.Config
	usageService *services.UsageService
	menuItems    []*systray.MenuItem
	logger       *lib.Logger
	stopFallback chan struct{} // signals the fallback polling goroutine to stop
}

// NewRunner creates a new instance of Runner
func NewRunner(config *models.Config, usageService *services.UsageService) *Runner {
	return &Runner{
		config:       config,
		usageService: usageService,
		menuItems:    make([]*systray.MenuItem, 0),
		logger:       lib.NewLogger("tray-runner"),
	}
}

// Run starts the system tray application
// This blocks until the application exits
func (tr *Runner) Run() {
	systray.Run(tr.onReady, tr.onExit)
}

func (tr *Runner) emojiForStatus(status models.AlertStatus) string {
	switch status {
	case models.Green:
		return "🟢"
	case models.Yellow:
		return "🟡"
	case models.Red:
		return "🔴"
	case models.Unknown:
		return "⚪️"
	default:
		return "⚪️"
	}
}

func (tr *Runner) onReady() {
	systray.SetTitle("CC Loading...")
	systray.SetTooltip("Claude Code Daily Usage Monitor")

	// Create placeholder menu items (will be dynamically updated)
	for i := 0; i < 10; i++ {
		tr.menuItems = append(tr.menuItems, systray.AddMenuItem("Loading...", "Loading..."))
	}

	systray.AddSeparator()
	mSettings := systray.AddMenuItem("Settings", "Open settings")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the application")

	// Initial update
	tr.updateStatus()

	// Use the service's polling mechanism
	err := tr.usageService.StartPolling(tr.config.UpdateInterval, func(state *models.UsageState) {
		tr.updateUIFromState(state)
	})
	if err != nil {
		tr.logger.Warn("Failed to start polling, falling back to manual updates", map[string]interface{}{
			"error": err.Error(),
		})
		tr.stopFallback = make(chan struct{})
		go func() {
			ticker := time.NewTicker(time.Duration(tr.config.UpdateInterval) * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					tr.updateStatus()
				case <-tr.stopFallback:
					return
				}
			}
		}()
	}

	// Handle menu clicks in a separate goroutine
	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				tr.showSettings()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func (tr *Runner) updateUIFromState(state *models.UsageState) {
	if state == nil {
		systray.SetTitle("CC Error")
		tr.updateMenuItems([]string{"❌ No data available"})
		return
	}

	if !state.IsAvailable || state.Status == models.Unknown {
		systray.SetTitle("CC ⚪️ Unknown")
		tr.updateMenuItems([]string{"⚠️ Usage data unavailable"})
		return
	}

	// Compute status based on configured thresholds
	state.UpdateStatus(tr.config.YellowThreshold, tr.config.RedThreshold)
	emoji := tr.emojiForStatus(state.Status)

	// Update compact title
	systray.SetTitle(fmt.Sprintf("CC %s $%.2f", emoji, state.DailyCost))

	// Update detailed menu items
	detailedInfo := []string{
		fmt.Sprintf("💰 Daily Cost: $%.2f", state.DailyCost),
		fmt.Sprintf("🎯 API Calls: %d", state.DailyCount),
		fmt.Sprintf("📅 Last Update: %s", state.LastUpdate.Format("2006-01-02 15:04:05")),
	}
	tr.updateMenuItems(detailedInfo)
}

func (tr *Runner) updateStatus() {
	// Force a fresh update from ccusage
	usage, err := tr.usageService.UpdateUsage()
	if err != nil {
		tr.logger.Error("Error getting usage data", map[string]interface{}{
			"error": err.Error(),
		})
		systray.SetTitle("CC Error")
		tr.updateMenuItems([]string{"❌ Failed to fetch data"})
		return
	}

	tr.updateUIFromState(usage)
}

func (tr *Runner) updateMenuItems(info []string) {
	for i, item := range tr.menuItems {
		if i < len(info) {
			if info[i] == "" {
				item.Hide()
			} else {
				item.Show()
				item.SetTitle(info[i])
			}
		} else {
			item.Hide()
		}
	}
}

func (tr *Runner) showSettings() {
	// Show settings in the tray title temporarily
	settingsTitle := fmt.Sprintf("Settings: %ds, $%.1f/$%.1f",
		tr.config.UpdateInterval, tr.config.YellowThreshold, tr.config.RedThreshold)
	systray.SetTitle(settingsTitle)

	// Log full settings
	tr.logger.Info("Current Settings", map[string]interface{}{
		"ccusage_path":     tr.config.CCUsagePath,
		"update_interval":  tr.config.UpdateInterval,
		"yellow_threshold": tr.config.YellowThreshold,
		"red_threshold":    tr.config.RedThreshold,
		"debug_level":      tr.config.DebugLevel,
	})

	// Reset title after 3 seconds
	go func() {
		time.Sleep(3 * time.Second)
		// Get current usage to restore proper title
		usage, err := tr.usageService.GetDailyUsage()
		if err == nil && usage != nil && usage.IsAvailable {
			// Recalculate status before reading it to avoid stale emoji
			usage.UpdateStatus(tr.config.YellowThreshold, tr.config.RedThreshold)
			emoji := tr.emojiForStatus(usage.Status)
			systray.SetTitle(fmt.Sprintf("CC %s $%.2f", emoji, usage.DailyCost))
		} else {
			systray.SetTitle("CC Loading...")
		}
	}()
}

func (tr *Runner) onExit() {
	// Stop the fallback polling goroutine if it's running
	if tr.stopFallback != nil {
		close(tr.stopFallback)
	}

	// Ensure background goroutines stop cleanly
	if tr.usageService != nil {
		tr.usageService.StopPolling()
	}
}
