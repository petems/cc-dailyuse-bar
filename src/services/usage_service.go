package services

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

	"cc-dailyuse-bar/src/lib"
	"cc-dailyuse-bar/src/models"
)

const maxLoggedOutputLength = 128

var errCCUsageUnavailable = errors.New("ccusage is not available")

// UsageService implements Claude Code usage tracking via ccusage integration
type UsageService struct {
	lastQuery      time.Time
	state          *models.UsageState
	logger         *lib.Logger
	ticker         *time.Ticker
	pollStopChan   chan struct{}
	resetStopChan  chan struct{}
	updateCallback func(*models.UsageState)
	ccusagePath    string
	cacheWindow    time.Duration
	mutex          sync.RWMutex // Protect shared state access
	cmdTimeout     time.Duration
}

// NewUsageService creates a new UsageService instance
func NewUsageService(config *models.Config) *UsageService {
	return &UsageService{
		ccusagePath:   config.CCUsagePath,
		state:         models.NewUsageState(),
		cacheWindow:   time.Duration(config.CacheWindow) * time.Second,
		logger:        lib.NewLogger("usage-service"),
		pollStopChan:  make(chan struct{}),
		resetStopChan: make(chan struct{}),
		cmdTimeout:    time.Duration(config.CmdTimeout) * time.Second,
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
	us.mutex.RLock()
	if time.Since(us.lastQuery) < us.cacheWindow && us.state.IsAvailable {
		// Copy the cached state while still holding the read lock to avoid
		// check-then-act races with concurrent writers.
		stateCopy := *us.state
		us.mutex.RUnlock()
		return &stateCopy, nil
	}
	us.mutex.RUnlock()

	us.mutex.Lock()
	defer us.mutex.Unlock()

	if time.Since(us.lastQuery) < us.cacheWindow && us.state.IsAvailable {
		return us.getStateCopyLocked(), nil
	}

	return us.performUpdateLocked(1)
}

// UpdateUsage forces a fresh query to ccusage, bypassing cache
// Used for immediate updates when user requests refresh
// Returns error if ccusage command fails or data is invalid
func (us *UsageService) UpdateUsage() (*models.UsageState, error) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	return us.performUpdateLocked(1)
}

// getStateCopy returns a thread-safe copy of the current state
func (us *UsageService) getStateCopy() *models.UsageState {
	us.mutex.RLock()
	defer us.mutex.RUnlock()
	return us.getStateCopyLocked()
}

func (us *UsageService) getStateCopyLocked() *models.UsageState {
	stateCopy := *us.state
	return &stateCopy
}

// setUnknownState marks the usage data as unavailable/unknown
func (us *UsageService) setUnknownState() {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.setUnknownStateLocked()
}

func (us *UsageService) setUnknownStateLocked() {
	now := time.Now()
	us.state.DailyCount = 0
	us.state.DailyCost = 0.0
	us.state.LastUpdate = now
	us.state.IsAvailable = false
	us.state.Status = models.Unknown
	us.lastQuery = now
}

// setNoDataForToday sets state for when ccusage works but has no data for today
func (us *UsageService) setNoDataForToday() {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.setNoDataForTodayLocked()
}

func (us *UsageService) setNoDataForTodayLocked() {
	now := time.Now()
	us.state.DailyCount = 0
	us.state.DailyCost = 0.0
	us.state.LastUpdate = now
	us.state.IsAvailable = true    // ccusage itself works
	us.state.Status = models.Green // $0.00 is Green status
	us.lastQuery = now
}

// ResetDaily resets counters for a new day
// Called automatically at midnight or manually by user
// Returns error only for system clock issues
func (us *UsageService) ResetDaily() error {
	us.mutex.Lock()
	defer us.mutex.Unlock()
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

	info, err := os.Stat(us.ccusagePath)
	if err != nil {
		if _, pathErr := exec.LookPath(us.ccusagePath); pathErr != nil {
			return false
		}
		return true
	}

	if info.IsDir() {
		return false
	}

	return info.Mode()&0o111 != 0
}

// SetCCUsagePath updates the path to ccusage binary
// Validates that the new path is executable
// Returns error if path is invalid or not executable
func (us *UsageService) SetCCUsagePath(path string) error {
	if path == "" {
		return lib.ValidationError("ccusage path cannot be empty")
	}

	oldPath := us.ccusagePath
	us.ccusagePath = path

	if !us.IsAvailable() {
		us.ccusagePath = oldPath
		return lib.ValidationError("ccusage path is not executable: " + path)
	}

	return nil
}

// SetThresholds updates the alert thresholds and recalculates status
func (us *UsageService) SetThresholds(yellowThreshold, redThreshold float64) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.state.UpdateStatus(yellowThreshold, redThreshold)
}

// T025: Connect to ccusage binary with retry logic
func (us *UsageService) updateWithRetry(maxRetries int) (*models.UsageState, error) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	return us.performUpdateLocked(maxRetries)
}

// performUpdateLocked assumes us.mutex is already held by the caller.
// It returns a copy of the current state after attempting to refresh usage data.
func (us *UsageService) performUpdateLocked(maxRetries int) (*models.UsageState, error) {
	if maxRetries < 1 {
		maxRetries = 1
	}

	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if maxRetries > 1 {
			us.logger.Debug("Attempting ccusage query", map[string]interface{}{
				"attempt":     attempt,
				"maxRetries":  maxRetries,
				"ccusagePath": us.ccusagePath,
			})
		}

		if !us.IsAvailable() {
			lastErr = errCCUsageUnavailable
			us.logger.Warn("ccusage not available", map[string]interface{}{
				"attempt": attempt,
				"path":    us.ccusagePath,
			})

			if attempt < maxRetries {
				us.sleepForRetry(attempt)
				continue
			}

			if lastErr == nil {
				lastErr = errCCUsageUnavailable
			}
			us.setUnknownStateLocked()
			return us.getStateCopyLocked(), lastErr
		}

		output, err := us.executeCCUsage()
		if err != nil {
			wrapped := lib.WrapError(err, lib.ErrCodeCCUsage, "ccusage command failed")
			if wrapped != nil {
				lastErr = wrapped
			} else {
				lastErr = err
			}

			extra := map[string]interface{}{}
			if maxRetries > 1 {
				extra["attempt"] = attempt
				extra["maxRetries"] = maxRetries
			}
			us.state.IsAvailable = false
			us.logCommandFailure(err, output, extra)

			if attempt < maxRetries {
				us.sleepForRetry(attempt)
				continue
			}

			if lastErr == nil {
				lastErr = err
			}
			us.setUnknownStateLocked()
			return us.getStateCopyLocked(), lastErr
		}

		response, err := parseCCUsageResponse(output)
		if err != nil {
			us.logger.Warn("ccusage JSON parsing failed, marking as unknown", map[string]interface{}{
				"error":   err.Error(),
				"out_len": len(output),
				"output":  truncateOutput(output),
			})
			us.setUnknownStateLocked()
			return us.getStateCopyLocked(), lib.WrapError(err, lib.ErrCodeCCUsage, "failed to parse ccusage JSON output")
		}

		today := time.Now().Format("2006-01-02")
		ccusageOutput, found := findTodayOutput(response, today)
		if !found {
			us.logger.Info("No data found for today, setting to $0.00", map[string]interface{}{
				"today":          today,
				"availableDates": availableDates(response.Daily),
			})
			us.setNoDataForTodayLocked()
			return us.getStateCopyLocked(), lib.WrapError(errors.New("no data for today"), lib.ErrCodeCCUsage, "ccusage has no data for today")
		}

		if ccusageOutput.TotalCost == 0 && ccusageOutput.TotalTokens == 0 {
			us.logger.Warn("ccusage returned zero values, marking as unknown", map[string]interface{}{
				"totalTokens": ccusageOutput.TotalTokens,
				"totalCost":   ccusageOutput.TotalCost,
				"date":        ccusageOutput.Date,
			})
			us.setUnknownStateLocked()
			return us.getStateCopyLocked(), lib.WrapError(errors.New("ccusage returned zero values"), lib.ErrCodeCCUsage, "ccusage returned invalid zero values")
		}

		us.applyUsageDataLocked(ccusageOutput)

		context := map[string]interface{}{
			"totalTokens": ccusageOutput.TotalTokens,
			"totalCost":   ccusageOutput.TotalCost,
			"date":        ccusageOutput.Date,
		}
		if maxRetries > 1 {
			context["attempt"] = attempt
		}
		us.logger.Info("Successfully parsed ccusage data", context)

		return us.getStateCopyLocked(), nil
	}

	if lastErr == nil {
		lastErr = errCCUsageUnavailable
	}
	us.setUnknownStateLocked()
	return us.getStateCopyLocked(), lastErr
}

func (us *UsageService) executeCCUsage() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), us.cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, us.ccusagePath, "daily", "--json")
	output, err := cmd.Output()
	if err != nil {
		return output, err
	}

	us.logger.Debug("ccusage command successful", map[string]interface{}{
		"out_len": len(output),
	})

	return output, nil
}

func parseCCUsageResponse(output []byte) (*CCUsageResponse, error) {
	var response CCUsageResponse
	if err := json.Unmarshal(output, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

func findTodayOutput(response *CCUsageResponse, today string) (CCUsageOutput, bool) {
	for _, daily := range response.Daily {
		if daily.Date == today {
			return daily, true
		}
	}
	return CCUsageOutput{}, false
}

func availableDates(daily []CCUsageOutput) []string {
	dates := make([]string, len(daily))
	for i, d := range daily {
		dates[i] = d.Date
	}
	return dates
}

func (us *UsageService) applyUsageData(output CCUsageOutput) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.applyUsageDataLocked(output)
}

func (us *UsageService) applyUsageDataLocked(output CCUsageOutput) {
	now := time.Now()
	us.state.DailyCount = output.TotalTokens
	us.state.DailyCost = output.TotalCost
	us.state.LastUpdate = now
	us.state.IsAvailable = true
	us.lastQuery = now
}

func (us *UsageService) logCommandFailure(err error, output []byte, extra map[string]interface{}) {
	context := map[string]interface{}{
		"error":   err.Error(),
		"out_len": len(output),
		"output":  truncateOutput(output),
		"path":    us.ccusagePath,
	}
	for k, v := range extra {
		context[k] = v
	}

	us.logger.Warn("ccusage command failed", context)
}

func truncateOutput(output []byte) string {
	if len(output) <= maxLoggedOutputLength {
		return string(output)
	}
	return string(output[:maxLoggedOutputLength]) + "..."
}

func (us *UsageService) sleepForRetry(attempt int) {
	time.Sleep(time.Duration(attempt) * time.Second)
}

// T030: Polling timer with configurable interval
func (us *UsageService) StartPolling(intervalSeconds int, callback func(*models.UsageState)) error {
	if intervalSeconds <= 0 {
		return lib.ValidationError("polling interval must be positive")
	}

	us.StopPolling()

	// Create ticker and assign callback atomically with mutex protection
	us.mutex.Lock()
	us.updateCallback = callback
	us.ticker = time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	us.mutex.Unlock()

	us.logger.Info("Starting usage polling", map[string]interface{}{
		"intervalSeconds": intervalSeconds,
	})

	go us.pollingLoop()

	return nil
}

// StopPolling stops the polling timer
func (us *UsageService) StopPolling() {
	select {
	case us.pollStopChan <- struct{}{}:
	default:
	}
	select {
	case us.resetStopChan <- struct{}{}:
	default:
	}

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

			state, err := us.updateWithRetry(3) // 3 retries for polling
			if err != nil {
				us.logger.Error("Polling update failed", map[string]interface{}{
					"error": err.Error(),
				})
			}

			us.mutex.RLock()
			callback := us.updateCallback
			us.mutex.RUnlock()
			if callback != nil {
				callback(state)
			}

		case <-us.pollStopChan:
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
	resetChecker := time.NewTicker(1 * time.Minute)
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
					us.mutex.RLock()
					callback := us.updateCallback
					us.mutex.RUnlock()
					if callback != nil {
						state, _ := us.GetDailyUsage()
						callback(state)
					}
				}
				lastResetDay = now.Day()
			}

		case <-us.resetStopChan:
			us.logger.Debug("Daily reset loop stopped")
			return
		}
	}
}
