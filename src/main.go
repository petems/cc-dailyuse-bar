package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/getlantern/systray"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

var (
	configService *services.ConfigService
	config        *models.Config
	menuItems     []*systray.MenuItem
	isJapanese    bool
	usageService  *services.UsageService
	daemonMode    bool
)

func init() {
	// Detect system locale
	lang := os.Getenv("LANG")
	if lang == "" {
		// Fallback to checking other environment variables
		lang = os.Getenv("LC_ALL")
		if lang == "" {
			lang = os.Getenv("LC_MESSAGES")
		}
	}

	// Check if Japanese locale is set
	isJapanese = strings.HasPrefix(strings.ToLower(lang), "ja")

	// Initialize services
	configService = services.NewConfigService()
	var err error
	config, err = configService.Load()
	if err != nil {
		// Log the error and use defaults if config loading fails
		log.Printf("Failed to load configuration: %v", err)
		log.Printf("Using default configuration")
		config = models.ConfigDefaults()
	}

	usageService = services.NewUsageService(config)

	// Set logging level from config
	lib.SetGlobalLevel(lib.LogLevel(config.GetLogLevel()))

	// Debug: Log the loaded config
	log.Printf("Loaded config - UpdateInterval: %d, DebugLevel: %s", config.UpdateInterval, config.DebugLevel)
}

func t(en, ja string) string {
	if isJapanese {
		return ja
	}
	return en
}

func emojiForStatus(status models.AlertStatus) string {
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

func main() {
	// Parse command line flags
	flag.BoolVar(&daemonMode, "daemon", false, "Run as daemon (background process)")
	flag.Parse()

	if daemonMode {
		runAsDaemon()
		return
	}

	systray.Run(onReady, onExit)
}

func runAsDaemon() {
	// Get the current executable path
	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}

	// Start the application in the background; validate path resolution first
	resolved, err := exec.LookPath(execPath)
	if err != nil {
		log.Fatalf("Failed to resolve executable: %v", err)
	}
	cmd := exec.CommandContext(context.Background(), resolved) // #nosec G204 validated via LookPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the process
	if err := cmd.Start(); err != nil {
		log.Fatalf("Failed to start daemon: %v", err)
	}

	fmt.Printf("CC Daily Use Bar started as daemon (PID: %d)\n", cmd.Process.Pid)
	fmt.Printf("To stop: kill %d\n", cmd.Process.Pid)

	// Exit the parent process
	os.Exit(0)
}

func onReady() {
	systray.SetTitle(fmt.Sprintf("CC %s...", t("Loading", "読み込み中")))
	systray.SetTooltip("Claude Code Daily Usage Monitor")

	// Create placeholder menu items (will be dynamically updated)
	for i := 0; i < 10; i++ {
		menuItems = append(menuItems, systray.AddMenuItem(t("Loading...", "読み込み中..."), "Loading..."))
	}

	systray.AddSeparator()
	mSettings := systray.AddMenuItem(t("Settings", "設定"), "Open settings")
	systray.AddSeparator()
	mQuit := systray.AddMenuItem(t("Quit", "終了"), "Quit the application")

	// Initial update
	updateStatus()

	// Use the service's polling mechanism instead of creating our own ticker
	err := usageService.StartPolling(config.UpdateInterval, func(state *models.UsageState) {
		// This callback will be called by the service on each update
		updateUIFromState(state)
	})
	if err != nil {
		log.Printf("Failed to start polling: %v", err)
		// Fallback to manual updates if polling fails
		go func() {
			ticker := time.NewTicker(time.Duration(config.UpdateInterval) * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				updateStatus()
			}
		}()
	}

	// Handle menu clicks in a separate goroutine
	go func() {
		for {
			select {
			case <-mSettings.ClickedCh:
				showSettings()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

// updateUIFromState updates the UI based on the state provided by the service.
func updateUIFromState(state *models.UsageState) {
	if state == nil {
		systray.SetTitle("CC " + t("Error", "エラー"))
		updateMenuItems([]string{"❌ " + t("No data available", "データがありません")})
		return
	}

	if !state.IsAvailable || state.Status == models.Unknown {
		systray.SetTitle("CC " + "⚪️" + " " + t("Unknown", "不明"))
		updateMenuItems([]string{"⚠️ " + t("Usage data unavailable", "使用データを利用できません")})
		return
	}

	// Compute status based on configured thresholds
	state.UpdateStatus(config.YellowThreshold, config.RedThreshold)
	emoji := emojiForStatus(state.Status)

	// Update compact title
	systray.SetTitle("CC " + emoji + " $" + fmt.Sprintf("%.2f", state.DailyCost))

	// Update detailed menu items
	detailedInfo := []string{
		fmt.Sprintf("💰 %s: $%.2f", t("Daily Cost", "1日コスト"), state.DailyCost),
		fmt.Sprintf("🎯 %s: %d", t("API Calls", "API呼び出し"), state.DailyCount),
		fmt.Sprintf("📅 %s: %s", t("Last Update", "最終更新"), state.LastUpdate.Format("2006-01-02 15:04:05")),
	}
	updateMenuItems(detailedInfo)
}

func updateStatus() {
	// Force a fresh update from ccusage
	usage, err := usageService.UpdateUsage()
	if err != nil {
		log.Printf("Error getting usage data: %v", err)
		systray.SetTitle("CC " + t("Error", "エラー"))
		updateMenuItems([]string{"❌ " + t("Failed to fetch data", "データを取得できませんでした")})
		return
	}

	// Use the new updateUIFromState function
	updateUIFromState(usage)
}

func updateMenuItems(info []string) {
	for i, item := range menuItems {
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

func showSettings() {
	// Show settings in the tray title temporarily
	settingsTitle := fmt.Sprintf("Settings: %ds, $%.1f/$%.1f", config.UpdateInterval, config.YellowThreshold, config.RedThreshold)
	systray.SetTitle(settingsTitle)

	// Log full settings to console
	log.Printf("Current Settings:")
	log.Printf("  ccusage Path: %s", config.CCUsagePath)
	log.Printf("  Update Interval: %d seconds", config.UpdateInterval)
	log.Printf("  Yellow Threshold: $%.2f", config.YellowThreshold)
	log.Printf("  Red Threshold: $%.2f", config.RedThreshold)
	log.Printf("  Debug Level: %s", config.DebugLevel)
	log.Printf("  Config file: ~/.config/cc-dailyuse-bar/config.yaml")

	// Reset title after 3 seconds
	go func() {
		const settingsTitleRestoreDelaySeconds = 3
		time.Sleep(settingsTitleRestoreDelaySeconds * time.Second)
		// Get current usage to restore proper title
		usage, err := usageService.GetDailyUsage()
		if err == nil && usage != nil && usage.IsAvailable {
			emoji := emojiForStatus(usage.Status)
			systray.SetTitle("CC " + emoji + " $" + fmt.Sprintf("%.2f", usage.DailyCost))
		} else {
			systray.SetTitle("CC Loading...")
		}
	}()
}

func onExit() {
	// Ensure background goroutines stop cleanly
	if usageService != nil {
		usageService.StopPolling()
	}
}
