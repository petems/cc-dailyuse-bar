package services

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
)

// Helper function to create a usage service with default config
func newTestUsageService() *UsageService {
	config := models.ConfigDefaults()
	return NewUsageService(config)
}

func TestNewUsageService(t *testing.T) {
	config := models.ConfigDefaults()
	service := NewUsageService(config)

	assert.NotNil(t, service)
	assert.Equal(t, "ccusage", service.ccusagePath)
	assert.NotNil(t, service.state)
	assert.NotNil(t, service.logger)
	// Logger component is not exported, so we can't test it directly
	assert.Equal(t, 10*time.Second, service.cacheWindow)
	assert.Equal(t, 5*time.Second, service.cmdTimeout)
	assert.NotNil(t, service.pollStopChan)
}

func TestUsageService_IsAvailable(t *testing.T) {
	service := newTestUsageService()

	// Test with default path (should be available if ccusage is in PATH)
	available := service.IsAvailable()
	// This might be true or false depending on the test environment
	_ = available

	// Test with empty path
	service.ccusagePath = ""
	assert.False(t, service.IsAvailable())

	// Test with non-existent path
	service.ccusagePath = "/non/existent/path"
	assert.False(t, service.IsAvailable())

	// Test with directory instead of file
	service.ccusagePath = "/tmp"
	assert.False(t, service.IsAvailable())
}

func TestUsageService_SetCCUsagePath(t *testing.T) {
	service := newTestUsageService()

	// Test with empty path
	err := service.SetCCUsagePath("")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")

	// Test with non-existent path
	err = service.SetCCUsagePath("/non/existent/path")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not executable")

	// Test with directory
	err = service.SetCCUsagePath("/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not executable")

	// Test with valid path (if available)
	if _, err := exec.LookPath("ccusage"); err == nil {
		err = service.SetCCUsagePath("ccusage")
		assert.NoError(t, err)
		assert.Equal(t, "ccusage", service.ccusagePath)
	}
}

func TestUsageService_ResetDaily(t *testing.T) {
	service := newTestUsageService()

	// Set some data
	service.state.DailyCount = 100
	service.state.DailyCost = 25.0
	service.state.Status = models.Red
	service.state.IsAvailable = true
	service.lastQuery = time.Now()

	// Reset
	err := service.ResetDaily()
	require.NoError(t, err)

	// Verify reset
	assert.Equal(t, 0, service.state.DailyCount)
	assert.Equal(t, 0.0, service.state.DailyCost)
	assert.Equal(t, models.Green, service.state.Status)
	assert.True(t, service.state.IsAvailable)  // Should be preserved
	assert.True(t, service.lastQuery.IsZero()) // Should be cleared
}

func TestUsageService_SetThresholds(t *testing.T) {
	service := newTestUsageService()

	// Set some cost
	service.state.DailyCost = 15.0

	// Set thresholds
	service.SetThresholds(10.0, 20.0)

	// Should be yellow (between thresholds)
	assert.Equal(t, models.Yellow, service.state.Status)

	// Change cost and update thresholds
	service.state.DailyCost = 25.0
	service.SetThresholds(10.0, 20.0)

	// Should be red (above red threshold)
	assert.Equal(t, models.Red, service.state.Status)
}

func TestUsageService_SetUnknownState(t *testing.T) {
	service := newTestUsageService()

	// Call setUnknownState
	service.setUnknownState()

	// Verify unknown state
	assert.Equal(t, 0, service.state.DailyCount)
	assert.Equal(t, 0.0, service.state.DailyCost)
	assert.False(t, service.state.IsAvailable)
	assert.Equal(t, models.Unknown, service.state.Status)
	assert.False(t, service.state.LastUpdate.IsZero())
}

func TestUsageService_SetNoDataForToday(t *testing.T) {
	service := newTestUsageService()

	// Call setNoDataForToday
	service.setNoDataForToday()

	// Verify no data for today state
	assert.Equal(t, 0, service.state.DailyCount)
	assert.Equal(t, 0.0, service.state.DailyCost)
	assert.True(t, service.state.IsAvailable)  // ccusage works, just no data today
	assert.Equal(t, models.Green, service.state.Status) // $0.00 = Green
	assert.False(t, service.state.LastUpdate.IsZero())
}

func TestUsageService_UpdateUsage_NotAvailable(t *testing.T) {
	service := newTestUsageService()
	service.ccusagePath = "/non/existent/path"

	state, err := service.UpdateUsage()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
	assert.False(t, state.IsAvailable)
}

func TestUsageService_GetDailyUsage_Cache(t *testing.T) {
	service := newTestUsageService()

	// Set up some state
	service.state.DailyCount = 50
	service.state.DailyCost = 10.0
	service.state.IsAvailable = true
	service.lastQuery = time.Now()

	// Should return cached data
	state, err := service.GetDailyUsage()
	require.NoError(t, err)

	assert.Equal(t, 50, state.DailyCount)
	assert.Equal(t, 10.0, state.DailyCost)
	assert.True(t, state.IsAvailable)
}

func TestUsageService_GetDailyUsage_CacheExpired(t *testing.T) {
	service := newTestUsageService()

	// Set up some state with old timestamp
	service.state.DailyCount = 50
	service.state.DailyCost = 10.0
	service.state.IsAvailable = true
	service.lastQuery = time.Now().Add(-20 * time.Second) // Older than cache window

	// Should call UpdateUsage (which will fail if ccusage not available)
	state, err := service.GetDailyUsage()

	// If ccusage is not available, should get error
	if !service.IsAvailable() {
		assert.Error(t, err)
		assert.False(t, state.IsAvailable)
	}
}

func TestUsageService_StartPolling_InvalidInterval(t *testing.T) {
	service := newTestUsageService()

	// Test with zero interval
	err := service.StartPolling(0, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")

	// Test with negative interval
	err = service.StartPolling(-10, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be positive")
}

func TestUsageService_StartPolling_ValidInterval(t *testing.T) {
	service := newTestUsageService()

	// Ensure clean state
	service.StopPolling()

	callback := func(state *models.UsageState) {
		// Callback executed
	}

	// Start polling with short interval
	err := service.StartPolling(1, callback)
	require.NoError(t, err)

	// Verify ticker is set
	assert.NotNil(t, service.ticker)
	assert.NotNil(t, service.updateCallback)

	// Wait a bit for callback to be called
	time.Sleep(2 * time.Second)

	// Stop polling
	service.StopPolling()

	// Verify ticker is cleared
	assert.Nil(t, service.ticker)
}

func TestUsageService_StopPolling(t *testing.T) {
	service := newTestUsageService()

	// Ensure clean state
	service.StopPolling()

	// Start polling
	err := service.StartPolling(1, nil)
	require.NoError(t, err)

	// Stop polling
	service.StopPolling()

	// Verify ticker is cleared
	assert.Nil(t, service.ticker)
}

func TestUsageService_StartDailyResetMonitor(t *testing.T) {
	service := newTestUsageService()

	// Ensure clean state
	service.StopPolling()

	// Set some data
	service.state.DailyCount = 100
	service.state.DailyCost = 25.0

	// Start daily reset monitor
	service.StartDailyResetMonitor()

	// Wait a bit
	time.Sleep(100 * time.Millisecond)

	// Stop the service gracefully
	service.StopPolling()
}

func TestUsageService_UpdateWithRetry_NotAvailable(t *testing.T) {
	service := newTestUsageService()

	// Ensure clean state
	service.StopPolling()

	service.ccusagePath = "/non/existent/path"

	state, err := service.updateWithRetry(3)

	assert.Error(t, err)
	assert.False(t, state.IsAvailable)
}

func TestUsageService_UpdateWithRetry_CommandFailure(t *testing.T) {
	service := newTestUsageService()

	// Create a temporary script that always fails
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "failing-ccusage")

	scriptContent := `#!/bin/bash
exit 1`

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	service.ccusagePath = scriptPath

	state, err := service.updateWithRetry(2)

	assert.Error(t, err)
	assert.False(t, state.IsAvailable)
}

func TestUsageService_UpdateWithRetry_InvalidJSON(t *testing.T) {
	service := newTestUsageService()

	// Create a temporary script that returns invalid JSON
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "invalid-json-ccusage")

	scriptContent := `#!/bin/bash
echo "invalid json"`

	err := os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	service.ccusagePath = scriptPath

	state, err := service.updateWithRetry(1)

	// Should fall back to simulation
	require.NoError(t, err)
	assert.True(t, state.IsAvailable)
	assert.GreaterOrEqual(t, state.DailyCount, 0)
}

func TestUsageService_UpdateWithRetry_ValidJSON(t *testing.T) {
	service := newTestUsageService()

	// Create a temporary script that returns valid JSON
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "valid-json-ccusage")

	today := time.Now().Format("2006-01-02")
	response := CCUsageResponse{
		Daily: []CCUsageOutput{
			{
				Date:        today,
				TotalTokens: 100,
				TotalCost:   5.0,
			},
		},
		Totals: struct {
			TotalTokens int     `json:"totalTokens"`
			TotalCost   float64 `json:"totalCost"`
		}{
			TotalTokens: 100,
			TotalCost:   5.0,
		},
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	scriptContent := `#!/bin/bash
echo '` + string(jsonData) + `'`

	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	service.ccusagePath = scriptPath

	state, err := service.updateWithRetry(1)

	require.NoError(t, err)
	assert.True(t, state.IsAvailable)
	assert.Equal(t, 100, state.DailyCount)
	assert.Equal(t, 5.0, state.DailyCost)
}

func TestUsageService_UpdateWithRetry_NoDataForToday(t *testing.T) {
	service := newTestUsageService()

	// Create a temporary script that returns JSON without today's data
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "no-today-ccusage")

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	response := CCUsageResponse{
		Daily: []CCUsageOutput{
			{
				Date:        yesterday,
				TotalTokens: 50,
				TotalCost:   2.5,
			},
		},
		Totals: struct {
			TotalTokens int     `json:"totalTokens"`
			TotalCost   float64 `json:"totalCost"`
		}{
			TotalTokens: 50,
			TotalCost:   2.5,
		},
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	scriptContent := `#!/bin/bash
echo '` + string(jsonData) + `'`

	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	service.ccusagePath = scriptPath

	state, err := service.updateWithRetry(1)

	// Should fall back to simulation
	require.NoError(t, err)
	assert.True(t, state.IsAvailable)
	assert.GreaterOrEqual(t, state.DailyCount, 0)
}

func TestUsageService_UpdateWithRetry_ZeroValues(t *testing.T) {
	service := newTestUsageService()

	// Create a temporary script that returns zero values
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "zero-values-ccusage")

	today := time.Now().Format("2006-01-02")
	response := CCUsageResponse{
		Daily: []CCUsageOutput{
			{
				Date:        today,
				TotalTokens: 0,
				TotalCost:   0.0,
			},
		},
		Totals: struct {
			TotalTokens int     `json:"totalTokens"`
			TotalCost   float64 `json:"totalCost"`
		}{
			TotalTokens: 0,
			TotalCost:   0.0,
		},
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	scriptContent := `#!/bin/bash
echo '` + string(jsonData) + `'`

	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	service.ccusagePath = scriptPath

	state, err := service.updateWithRetry(1)

	// Should fall back to simulation
	require.NoError(t, err)
	assert.True(t, state.IsAvailable)
	assert.GreaterOrEqual(t, state.DailyCount, 0)
}

func TestUsageService_ConcurrentAccess(t *testing.T) {
	service := newTestUsageService()

	// Prime cached state so GetDailyUsage returns immediately without ccusage
	service.state.IsAvailable = true
	service.state.DailyCount = 100
	service.state.DailyCost = 5.0
	service.state.Status = models.Green
	service.state.LastUpdate = time.Now()
	service.lastQuery = time.Now()
	service.cacheWindow = time.Hour

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			state, err := service.GetDailyUsage()
			require.NoError(t, err)
			require.NotNil(t, state)
			assert.Equal(t, 100, state.DailyCount)
			assert.Equal(t, 5.0, state.DailyCost)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUsageService_EdgeCases(t *testing.T) {
	service := newTestUsageService()

	// Test with very large cache window
	service.cacheWindow = 24 * time.Hour

	// Set some data
	service.state.DailyCount = 100
	service.state.DailyCost = 10.0
	service.state.IsAvailable = true
	service.lastQuery = time.Now()

	// Should return cached data even after some time
	state, err := service.GetDailyUsage()
	require.NoError(t, err)
	assert.Equal(t, 100, state.DailyCount)

	// Test with zero cache window
	service.cacheWindow = 0
	service.lastQuery = time.Now().Add(-time.Second)

	// Should call UpdateUsage
	state, err = service.GetDailyUsage()
	// Result depends on ccusage availability
	_ = state
	_ = err
}

func TestUsageService_RealWorldScenarios(t *testing.T) {
	service := newTestUsageService()

	// Test scenario: ccusage not available
	service.ccusagePath = "/non/existent/path"
	state, err := service.GetDailyUsage()
	assert.Error(t, err)
	assert.False(t, state.IsAvailable)

	// Test scenario: valid ccusage with data
	if _, err := exec.LookPath("ccusage"); err == nil {
		service.ccusagePath = "ccusage"
		state, err := service.GetDailyUsage()
		// This might succeed or fail depending on environment
		_ = state
		_ = err
	}

	// Test scenario: reset and update
	service.state.DailyCount = 100
	service.state.DailyCost = 10.0
	service.state.Status = models.Red

	err = service.ResetDaily()
	require.NoError(t, err)

	assert.Equal(t, 0, service.state.DailyCount)
	assert.Equal(t, 0.0, service.state.DailyCost)
	assert.Equal(t, models.Green, service.state.Status)
}

func TestUsageService_NoDataForToday(t *testing.T) {
	service := newTestUsageService()

	// Create a mock ccusage script that returns data, but not for today
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "no-today-ccusage")

	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	response := CCUsageResponse{
		Daily: []CCUsageOutput{
			{
				Date:        yesterday,
				TotalTokens: 100,
				TotalCost:   5.0,
			},
		},
		Totals: struct {
			TotalTokens int     `json:"totalTokens"`
			TotalCost   float64 `json:"totalCost"`
		}{
			TotalTokens: 100,
			TotalCost:   5.0,
		},
	}

	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	scriptContent := `#!/bin/bash
echo '` + string(jsonData) + `'`

	err = os.WriteFile(scriptPath, []byte(scriptContent), 0755)
	require.NoError(t, err)

	service.ccusagePath = scriptPath

	// Act
	state, err := service.UpdateUsage()

	// Assert - Should show $0.00 for no data today, not Unknown
	assert.Error(t, err) // Should return error indicating no data for today
	assert.Contains(t, err.Error(), "no data for today")
	assert.Equal(t, 0, state.DailyCount)
	assert.Equal(t, 0.0, state.DailyCost)
	assert.True(t, state.IsAvailable) // ccusage works, just no data for today
	assert.NotEqual(t, models.Unknown, state.Status) // Should not be Unknown
}

func TestUsageService_DataUnavailable(t *testing.T) {
	service := newTestUsageService()

	// Test scenario: ccusage binary doesn't exist
	service.ccusagePath = "/non/existent/ccusage/binary"

	// Act
	state, err := service.UpdateUsage()

	// Assert - Should show Unknown status when ccusage is unavailable
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not available")
	assert.Equal(t, 0, state.DailyCount)
	assert.Equal(t, 0.0, state.DailyCost)
	assert.False(t, state.IsAvailable) // ccusage itself is unavailable
	assert.Equal(t, models.Unknown, state.Status) // Should be Unknown
}
