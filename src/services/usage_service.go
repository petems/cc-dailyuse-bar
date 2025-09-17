package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	lastQuery       time.Time
	state           *models.UsageState
	lastErr         error // wrapped error for cached failure state; nil after a successful refresh
	logger          *lib.Logger
	ticker          *time.Ticker
	pollStopChan    chan struct{}
	resetStopChan   chan struct{}
	updateCallback  func(*models.UsageState)
	ccusagePath     string
	cacheWindow     time.Duration
	mutex           sync.RWMutex // Protect shared state access
	cmdTimeout      time.Duration
	yellowThreshold float64
	redThreshold    float64
	fallbackLogOnce sync.Once
}

// NewUsageService creates a new UsageService instance
func NewUsageService(config *models.Config) *UsageService {
	return &UsageService{
		ccusagePath:     config.CCUsagePath,
		state:           models.NewUsageState(),
		cacheWindow:     time.Duration(config.CacheWindow) * time.Second,
		logger:          lib.NewLogger("usage-service"),
		pollStopChan:    make(chan struct{}),
		resetStopChan:   make(chan struct{}),
		cmdTimeout:      time.Duration(config.CmdTimeout) * time.Second,
		yellowThreshold: config.YellowThreshold,
		redThreshold:    config.RedThreshold,
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

// GetDailyUsage queries ccusage and returns current daily statistics. Returns
// cached data if the last query was within the cache window, including cached
// failure states so a sustained ccusage outage doesn't spawn one invocation
// per call. Cached failures are returned with the original wrapped error
// (parse error / command exit / missing binary / zero values) so callers and
// logs keep the real root cause for the duration of the throttle window.
func (us *UsageService) GetDailyUsage() (*models.UsageState, error) {
	us.mutex.RLock()
	if time.Since(us.lastQuery) < us.cacheWindow {
		// Copy the cached state and error while still holding the read lock
		// to avoid check-then-act races with concurrent writers.
		stateCopy := *us.state
		cachedErr := us.lastErr
		us.mutex.RUnlock()
		if !stateCopy.IsAvailable {
			return &stateCopy, cachedFailureError(cachedErr)
		}
		return &stateCopy, nil
	}
	us.mutex.RUnlock()

	us.mutex.Lock()
	defer us.mutex.Unlock()

	if time.Since(us.lastQuery) < us.cacheWindow {
		state := us.getStateCopyLocked()
		if !state.IsAvailable {
			return state, cachedFailureError(us.lastErr)
		}
		return state, nil
	}

	return us.performUpdateLocked(1)
}

// cachedFailureError returns the stored wrapped failure error, falling back
// to a generic unavailable error only when the cache predates a recorded
// cause (e.g. a manually-seeded failure state).
func cachedFailureError(cached error) error {
	if cached != nil {
		return cached
	}
	return lib.WrapError(errCCUsageUnavailable, lib.ErrCodeCCUsage, "serving cached ccusage failure state")
}

// newCCUsageUnavailableError wraps the bare sentinel with ErrCodeCCUsage so
// callers using errors.As(*lib.AppError) see the structured type. The chain
// preserves the sentinel, so errors.Is(err, errCCUsageUnavailable) still
// matches.
func newCCUsageUnavailableError() error {
	return lib.WrapError(errCCUsageUnavailable, lib.ErrCodeCCUsage, "ccusage is not available")
}

// UpdateUsage forces a fresh query to ccusage, bypassing cache
// Used for immediate updates when user requests refresh
// Returns error if ccusage command fails or data is invalid
func (us *UsageService) UpdateUsage() (*models.UsageState, error) {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	return us.performUpdateLocked(1)
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
	us.setStateMetricsLocked(0, 0, false)
	us.state.Status = models.Unknown
}

// recordFailureLocked transitions the cached state to Unknown and stores the
// wrapped cause so subsequent cached reads can surface the real error
// instead of a synthesized "unavailable" message.
func (us *UsageService) recordFailureLocked(err error) {
	us.setUnknownStateLocked()
	us.lastErr = err
}

// setNoDataForToday sets state for when ccusage works but has no data for today
func (us *UsageService) setNoDataForToday() {
	us.mutex.Lock()
	defer us.mutex.Unlock()
	us.setNoDataForTodayLocked()
}

func (us *UsageService) setNoDataForTodayLocked() {
	us.setStateMetricsLocked(0, 0, true)
	us.updateStatusLocked() // $0.00 cost should evaluate to Green
	us.lastErr = nil        // success state — clear any prior cached failure
}

func (us *UsageService) setStateMetricsLocked(tokens int, cost float64, available bool) {
	now := time.Now()
	us.state.DailyCount = tokens
	us.state.DailyCost = cost
	us.state.LastUpdate = now
	us.state.IsAvailable = available
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

	// ResolveCCUsagePath mirrors exec.CommandContext's PATH-only lookup for
	// bare names and adds a macOS Homebrew fallback so .app bundles launched
	// from /Applications (with a stripped PATH) can still find ccusage.
	// Both lookup paths (exec.LookPath and isExecutableFile) already verify
	// the result is an executable regular file, so we trust the resolver and
	// skip a redundant Stat — that redundant check used POSIX mode bits and
	// would incorrectly mark valid .exe files as unavailable on Windows.
	resolvedPath, fallback, err := ResolveCCUsagePath(us.ccusagePath)
	if err != nil {
		return false
	}

	if fallback {
		us.logFallbackOnce(resolvedPath)
	}
	return true
}

// logFallbackOnce surfaces a one-shot warning when ResolveCCUsagePath had to
// fall back to a Homebrew dir because PATH lookup failed. Helps users notice
// the .app-bundle PATH issue and pin ccusage_path to an absolute location.
func (us *UsageService) logFallbackOnce(resolvedPath string) {
	us.fallbackLogOnce.Do(func() {
		us.logger.Warn("ccusage not found on PATH; resolved via Homebrew fallback. Set ccusage_path to an absolute path in config to silence this.",
			map[string]interface{}{
				"configured": us.ccusagePath,
				"resolved":   resolvedPath,
			})
	})
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
	us.yellowThreshold = yellowThreshold
	us.redThreshold = redThreshold
	us.updateStatusLocked()
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
			lastErr = newCCUsageUnavailableError()
			us.logger.Warn("ccusage not available", map[string]interface{}{
				"attempt": attempt,
				"path":    us.ccusagePath,
			})

			if attempt < maxRetries {
				us.sleepForRetry(attempt)
				continue
			}

			us.recordFailureLocked(lastErr)
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
			us.recordFailureLocked(lastErr)
			return us.getStateCopyLocked(), lastErr
		}

		response, err := parseCCUsageResponse(output)
		if err != nil {
			us.logger.Warn("ccusage JSON parsing failed, marking as unknown", map[string]interface{}{
				"error":   err.Error(),
				"out_len": len(output),
				"output":  truncateOutput(output),
			})
			wrapped := lib.WrapError(err, lib.ErrCodeCCUsage, "failed to parse ccusage JSON output")
			us.recordFailureLocked(wrapped)
			return us.getStateCopyLocked(), wrapped
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
			wrapped := lib.WrapError(errors.New("ccusage returned zero values"), lib.ErrCodeCCUsage, "ccusage returned invalid zero values")
			us.recordFailureLocked(wrapped)
			return us.getStateCopyLocked(), wrapped
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
		lastErr = newCCUsageUnavailableError()
	}
	us.recordFailureLocked(lastErr)
	return us.getStateCopyLocked(), lastErr
}

func (us *UsageService) executeCCUsage() ([]byte, error) {
	// Use the resolved path so the macOS Homebrew fallback applies here too —
	// otherwise exec.CommandContext re-runs PATH lookup and fails for bare
	// names under launchd / LaunchServices' stripped PATH.
	resolvedPath, _, resolveErr := ResolveCCUsagePath(us.ccusagePath)
	if resolveErr != nil {
		return nil, lib.WrapError(resolveErr, lib.ErrCodeCCUsage,
			fmt.Sprintf("ccusage path %q could not be resolved", us.ccusagePath))
	}

	ctx, cancel := context.WithTimeout(context.Background(), us.cmdTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, resolvedPath, "daily", "--json")
	output, err := cmd.Output()
	if err != nil {
		// When the context deadline fires, Go kills the child with SIGKILL and
		// surfaces a generic "signal: killed". Translate it so users see what
		// actually happened and how to fix it.
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return output, fmt.Errorf("ccusage timed out after %s; increase cmd_timeout in config", us.cmdTimeout)
		}
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

func (us *UsageService) applyUsageDataLocked(output CCUsageOutput) {
	us.setStateMetricsLocked(output.TotalTokens, output.TotalCost, true)
	us.updateStatusLocked()
	us.lastErr = nil // success — clear any prior cached failure
}

func (us *UsageService) updateStatusLocked() {
	us.state.UpdateStatus(us.yellowThreshold, us.redThreshold)
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

// StartPolling starts a configurable-interval polling timer that invokes
// callback with the latest state on each tick (T030).
func (us *UsageService) StartPolling(intervalSeconds int, callback func(*models.UsageState)) error {
	if intervalSeconds <= 0 {
		return lib.ValidationError("polling interval must be positive")
	}

	us.StopPolling()

	ticker := time.NewTicker(time.Duration(intervalSeconds) * time.Second)

	// Create ticker and assign callback atomically with mutex protection
	us.mutex.Lock()
	us.updateCallback = callback
	us.ticker = ticker
	stopChan := us.pollStopChan
	us.mutex.Unlock()

	us.logger.Info("Starting usage polling", map[string]interface{}{
		"intervalSeconds": intervalSeconds,
	})

	go us.pollingLoop(ticker, stopChan)

	return nil
}

// StopPolling stops the polling timer
func (us *UsageService) StopPolling() {
	us.mutex.Lock()
	if us.ticker != nil {
		us.ticker.Stop()
		us.ticker = nil
	}

	pollStopChan := us.replaceStopChan(&us.pollStopChan)
	resetStopChan := us.replaceStopChan(&us.resetStopChan)
	us.mutex.Unlock()

	if pollStopChan != nil {
		close(pollStopChan)
	}
	if resetStopChan != nil {
		close(resetStopChan)
	}

	us.logger.Info("Usage polling stopped")
}

func (us *UsageService) replaceStopChan(chPtr *chan struct{}) chan struct{} {
	oldChan := *chPtr
	if oldChan != nil {
		*chPtr = make(chan struct{})
	}
	return oldChan
}

// pollingLoop runs the polling loop in a goroutine
func (us *UsageService) pollingLoop(ticker *time.Ticker, stopChan <-chan struct{}) {
	if ticker == nil {
		us.logger.Error("Polling loop started without ticker")
		return
	}

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

		case <-stopChan:
			us.logger.Debug("Polling loop stopped")
			return
		}
	}
}

// StartDailyResetMonitor starts the daily reset scheduler with midnight
// detection (T031).
func (us *UsageService) StartDailyResetMonitor() {
	us.mutex.RLock()
	stopChan := us.resetStopChan
	us.mutex.RUnlock()

	go us.dailyResetLoop(stopChan)
	us.logger.Info("Daily reset monitor started")
}

// dailyResetLoop monitors for midnight and resets daily counters
func (us *UsageService) dailyResetLoop(stopChan <-chan struct{}) {
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

		case <-stopChan:
			us.logger.Debug("Daily reset loop stopped")
			return
		}
	}
}
