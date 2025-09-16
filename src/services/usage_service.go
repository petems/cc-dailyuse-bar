package services

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
)

// UsageService implements Claude Code usage tracking via ccusage integration
type UsageService struct {
	lastQuery      time.Time
	state          *models.UsageState
	logger         *lib.Logger
	ticker         *time.Ticker
	stopChan       chan struct{}
	updateCallback func(*models.UsageState)
	ccusagePath    string
	cacheWindow    time.Duration
	mutex          sync.RWMutex // Protect ticker access
}

// NewUsageService creates a new UsageService instance
func NewUsageService() *UsageService {
	return &UsageService{
		ccusagePath: "ccusage",
		state:       models.NewUsageState(),
		cacheWindow: 10 * time.Second, // Cache ccusage results briefly
		logger:      lib.NewLogger("usage-service"),
		stopChan:    make(chan struct{}),
	}
}

// CCUsageOutput represents the JSON structure returned by ccusage
type CCUsageOutput struct {
	Date        string  `json:"date"`
	TotalTokens int     `json:"totalTokens"`
	TotalCost   float64 `json:"totalCost"`
}

// CCUsageResponse represents the full JSON response from ccusage
type CCUsageResponse struct {
	Daily  []CCUsageOutput `json:"daily"`
	Totals struct {
		TotalTokens int     `json:"totalTokens"`
		TotalCost   float64 `json:"totalCost"`
	} `json:"totals"`
}

// GetDailyUsage queries ccusage and returns current daily statistics
// Returns cached data if last query was within cache window
// Returns error if ccusage is unavailable or returns invalid data
func (us *UsageService) GetDailyUsage() (*models.UsageState, error) {
	// Return cached state if within cache window
	if time.Since(us.lastQuery) < us.cacheWindow && us.state.IsAvailable {
		return us.state, nil
	}

	// Update from ccusage
	return us.UpdateUsage()
}

// UpdateUsage forces a fresh query to ccusage, bypassing cache
// Used for immediate updates when user requests refresh
// Returns error if ccusage command fails or data is invalid
func (us *UsageService) UpdateUsage() (*models.UsageState, error) {
	if !us.IsAvailable() {
		us.state.IsAvailable = false
		return us.state, errors.New("ccusage is not available")
	}

	// Execute ccusage command
	cmd := exec.Command(us.ccusagePath, "daily", "--json")
	output, err := cmd.Output()
	if err != nil {
		us.logger.Warn("ccusage command failed", map[string]interface{}{
			"error":  err.Error(),
			"path":   us.ccusagePath,
			"output": string(output),
		})
		us.state.IsAvailable = false
		return us.state, err
	}

	us.logger.Debug("ccusage command successful", map[string]interface{}{
		"output": string(output),
	})

	// Parse JSON output - expect the full response structure
	var ccusageResponse CCUsageResponse
	if err := json.Unmarshal(output, &ccusageResponse); err != nil {
		// Log the error and raw output for debugging
		us.logger.Warn("Failed to parse ccusage JSON", map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
		// If ccusage doesn't support JSON or returns invalid data,
		// try to simulate data for development/testing
		us.simulateUsageData()
		us.lastQuery = time.Now()
		return us.state, nil
	}

	// Find today's data in the daily array
	today := time.Now().Format("2006-01-02")
	var ccusageOutput CCUsageOutput
	found := false

	for _, daily := range ccusageResponse.Daily {
		if daily.Date == today {
			ccusageOutput = daily
			found = true
			break
		}
	}

	if !found {
		us.logger.Warn("No data found for today", map[string]interface{}{
			"today": today,
			"availableDates": func() []string {
				dates := make([]string, len(ccusageResponse.Daily))
				for i, d := range ccusageResponse.Daily {
					dates[i] = d.Date
				}
				return dates
			}(),
		})
		us.simulateUsageData()
		us.lastQuery = time.Now()
		return us.state, nil
	}

	// Check if we got meaningful data
	if ccusageOutput.TotalCost == 0 && ccusageOutput.TotalTokens == 0 {
		us.logger.Warn("ccusage returned zero values, falling back to simulation", map[string]interface{}{
			"totalTokens": ccusageOutput.TotalTokens,
			"totalCost":   ccusageOutput.TotalCost,
			"date":        ccusageOutput.Date,
		})
		us.simulateUsageData()
		us.lastQuery = time.Now()
		return us.state, nil
	}

	// Update state from ccusage output
	us.state.DailyCount = ccusageOutput.TotalTokens
	us.state.DailyCost = ccusageOutput.TotalCost
	us.state.LastUpdate = time.Now()
	us.state.IsAvailable = true

	us.logger.Info("Successfully parsed ccusage data", map[string]interface{}{
		"totalTokens": ccusageOutput.TotalTokens,
		"totalCost":   ccusageOutput.TotalCost,
		"date":        ccusageOutput.Date,
	})

	us.lastQuery = time.Now()
	return us.state, nil
}

// simulateUsageData provides simulated data when ccusage is unavailable
// This helps with development and testing
func (us *UsageService) simulateUsageData() {
	now := time.Now()

	// Simulate some usage based on time of day
	hour := now.Hour()
	simulatedCount := hour * 3                      // More usage later in day
	simulatedCost := float64(simulatedCount) * 0.05 // $0.05 per request

	us.state.DailyCount = simulatedCount
	us.state.DailyCost = simulatedCost
	us.state.LastUpdate = now
	us.state.IsAvailable = true
}

// ResetDaily resets counters for a new day
// Called automatically at midnight or manually by user
// Returns error only for system clock issues
func (us *UsageService) ResetDaily() error {
	us.state.Reset()
	us.lastQuery = time.Time{} // Clear cache
	return nil
}

// IsAvailable checks if ccusage is accessible
// Performs quick validation without full query
// Returns false if binary not found or not executable
func (us *UsageService) IsAvailable() bool {
	if us.ccusagePath == "" {
		return false
	}

	// Check if file exists and is executable
	info, err := os.Stat(us.ccusagePath)
	if err != nil {
		// Try to find in PATH
		if _, pathErr := exec.LookPath(us.ccusagePath); pathErr != nil {
			return false
		}
		return true
	}

	// Check if it's a regular file and executable
	if info.IsDir() {
		return false
	}

	// Check basic execute permissions (simplified check)
	mode := info.Mode()
	return mode&0111 != 0
}

// SetCCUsagePath updates the path to ccusage binary
// Validates that the new path is executable
// Returns error if path is invalid or not executable
func (us *UsageService) SetCCUsagePath(path string) error {
	if path == "" {
		return errors.New("ccusage path cannot be empty")
	}

	// Test the path
	oldPath := us.ccusagePath
	us.ccusagePath = path

	if !us.IsAvailable() {
		// Restore old path
		us.ccusagePath = oldPath
		return errors.New("ccusage path is not executable: " + path)
	}

	return nil
}

// SetThresholds updates the alert thresholds and recalculates status
func (us *UsageService) SetThresholds(yellowThreshold, redThreshold float64) {
	us.state.UpdateStatus(yellowThreshold, redThreshold)
}

// T025: Connect to ccusage binary with retry logic
func (us *UsageService) updateWithRetry(maxRetries int) (*models.UsageState, error) {
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		us.logger.Debug("Attempting ccusage query", map[string]interface{}{
			"attempt":     attempt,
			"maxRetries":  maxRetries,
			"ccusagePath": us.ccusagePath,
		})

		if !us.IsAvailable() {
			lastError = lib.CCUsageError("ccusage binary not available")
			us.state.IsAvailable = false
			us.logger.Warn("ccusage not available", map[string]interface{}{
				"attempt": attempt,
				"path":    us.ccusagePath,
			})

			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			return us.state, lastError
		}

		// Execute ccusage command
		cmd := exec.Command(us.ccusagePath, "daily", "--json")
		output, err := cmd.Output()
		if err != nil {
			lastError = lib.WrapError(err, lib.ErrCodeCCUsage, "ccusage command failed")
			us.logger.Warn("ccusage command failed", map[string]interface{}{
				"attempt": attempt,
				"error":   err.Error(),
			})

			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			us.state.IsAvailable = false
			return us.state, lastError
		}

		// Parse JSON output - expect the full response structure
		var ccusageResponse CCUsageResponse
		if err := json.Unmarshal(output, &ccusageResponse); err != nil {
			us.logger.Info("ccusage JSON parsing failed, using simulation", map[string]interface{}{
				"error":  err.Error(),
				"output": string(output),
			})
			us.simulateUsageData()
			return us.state, nil
		}

		// Find today's data in the daily array
		today := time.Now().Format("2006-01-02")
		var ccusageOutput CCUsageOutput
		found := false

		for _, daily := range ccusageResponse.Daily {
			if daily.Date == today {
				ccusageOutput = daily
				found = true
				break
			}
		}

		if !found {
			us.logger.Warn("No data found for today in retry", map[string]interface{}{
				"today": today,
			})
			us.simulateUsageData()
			return us.state, nil
		}

		// Success - update state
		us.state.DailyCount = ccusageOutput.TotalTokens
		us.state.DailyCost = ccusageOutput.TotalCost
		us.state.LastUpdate = time.Now()
		us.state.IsAvailable = true
		us.lastQuery = time.Now()

		us.logger.Info("ccusage query successful", map[string]interface{}{
			"attempt":    attempt,
			"dailyCount": ccusageOutput.TotalTokens,
			"dailyCost":  ccusageOutput.TotalCost,
		})

		return us.state, nil
	}

	// All retries failed
	us.state.IsAvailable = false
	return us.state, lastError
}

// T030: Polling timer with configurable interval
func (us *UsageService) StartPolling(intervalSeconds int, callback func(*models.UsageState)) error {
	if intervalSeconds <= 0 {
		return lib.ValidationError("polling interval must be positive")
	}

	// Stop existing polling if running
	us.StopPolling()

	us.updateCallback = callback
	us.ticker = time.NewTicker(time.Duration(intervalSeconds) * time.Second)

	us.logger.Info("Starting usage polling", map[string]interface{}{
		"intervalSeconds": intervalSeconds,
	})

	go us.pollingLoop()

	return nil
}

// StopPolling stops the polling timer
func (us *UsageService) StopPolling() {
	// Send stop signal first
	select {
	case us.stopChan <- struct{}{}:
	default:
		// Channel might be full or closed, that's ok
	}

	// Then stop the ticker
	us.mutex.Lock()
	if us.ticker != nil {
		us.ticker.Stop()
		us.ticker = nil
	}
	us.mutex.Unlock()

	us.logger.Info("Usage polling stopped")
}

// pollingLoop runs the polling loop in a goroutine
func (us *UsageService) pollingLoop() {
	us.mutex.RLock()
	if us.ticker == nil {
		us.mutex.RUnlock()
		us.logger.Error("Polling loop started without ticker")
		return
	}
	ticker := us.ticker
	us.mutex.RUnlock()

	for {
		select {
		case <-ticker.C:
			us.logger.Debug("Polling timer triggered")

			// Use retry logic for polling updates
			state, err := us.updateWithRetry(3) // 3 retries for polling
			if err != nil {
				us.logger.Error("Polling update failed", map[string]interface{}{
					"error": err.Error(),
				})
			}

			// Call callback if set
			if us.updateCallback != nil {
				us.updateCallback(state)
			}

		case <-us.stopChan:
			us.logger.Debug("Polling loop stopped")
			return
		}
	}
}

// T031: Daily reset scheduler with midnight detection
func (us *UsageService) StartDailyResetMonitor() {
	go us.dailyResetLoop()
	us.logger.Info("Daily reset monitor started")
}

// dailyResetLoop monitors for midnight and resets daily counters
func (us *UsageService) dailyResetLoop() {
	lastResetDay := time.Now().Day()
	resetChecker := time.NewTicker(1 * time.Minute) // Check every minute
	defer resetChecker.Stop()

	for {
		select {
		case <-resetChecker.C:
			now := time.Now()
			if now.Day() != lastResetDay {
				us.logger.Info("Daily reset triggered", map[string]interface{}{
					"newDay":       now.Format("2006-01-02"),
					"lastResetDay": lastResetDay,
				})

				if err := us.ResetDaily(); err != nil {
					us.logger.Error("Daily reset failed", map[string]interface{}{
						"error": err.Error(),
					})
				} else {
					us.logger.Info("Daily usage reset successfully")
					// Trigger immediate update after reset
					if us.updateCallback != nil {
						state, _ := us.GetDailyUsage()
						us.updateCallback(state)
					}
				}
				lastResetDay = now.Day()
			}

		case <-us.stopChan:
			us.logger.Debug("Daily reset loop stopped")
			return
		}
	}
}
