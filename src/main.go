package main

import (
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
		// Use defaults if config loading fails
		config = models.ConfigDefaults()
	}

	usageService = services.NewUsageService()
	_ = usageService.SetCCUsagePath(config.CCUsagePath)

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

	// Start the application in the background
	cmd := exec.Command(execPath)
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

	// Start the update loop
	go func() {
		log.Printf("Starting update loop with interval: %d seconds", config.UpdateInterval)
		ticker := time.NewTicker(time.Duration(config.UpdateInterval) * time.Second)
		defer ticker.Stop()

		// Initial update
		updateStatus()

		for {
			select {
			case <-ticker.C:
				updateStatus()
			case <-mSettings.ClickedCh:
				showSettings()
			case <-mQuit.ClickedCh:
				systray.Quit()
				return
			}
		}
	}()
}

func updateStatus() {
	// Force a fresh update from ccusage
	usage, err := usageService.UpdateUsage()
	if err != nil {
		log.Printf("Error getting usage data: %v", err)
		systray.SetTitle(fmt.Sprintf("CC %s", t("Error", "エラー")))
		updateMenuItems([]string{fmt.Sprintf("❌ %s", t("Failed to fetch data", "データを取得できませんでした"))})
		return
	}

	if usage == nil || !usage.IsAvailable {
		systray.SetTitle(fmt.Sprintf("CC %s $0.00", "⚪️"))
		updateMenuItems([]string{fmt.Sprintf("⚠️ %s", t("ccusage unavailable", "ccusage を利用できません"))})
		return
	}

	// Compute status based on configured thresholds
	usage.UpdateStatus(config.YellowThreshold, config.RedThreshold)
	emoji := emojiForStatus(usage.Status)

	// Update compact title
	systray.SetTitle(fmt.Sprintf("CC %s $%.2f", emoji, usage.DailyCost))

	// Update detailed menu items
	detailedInfo := []string{
		fmt.Sprintf("💰 %s: $%.2f", t("Daily Cost", "1日コスト"), usage.DailyCost),
		fmt.Sprintf("🎯 %s: %d", t("API Calls", "API呼び出し"), usage.DailyCount),
		fmt.Sprintf("📅 %s: %s", t("Last Update", "最終更新"), usage.LastUpdate.Format("2006-01-02 15:04:05")),
	}
	updateMenuItems(detailedInfo)
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
	settingsTitle := fmt.Sprintf("Settings: %ds, $%.1f/$%.1f",
		config.UpdateInterval, config.YellowThreshold, config.RedThreshold)
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
		time.Sleep(3 * time.Second)
		// Get current usage to restore proper title
		usage, err := usageService.GetDailyUsage()
		if err == nil && usage != nil && usage.IsAvailable {
			emoji := emojiForStatus(usage.Status)
			systray.SetTitle(fmt.Sprintf("CC %s $%.2f", emoji, usage.DailyCost))
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
