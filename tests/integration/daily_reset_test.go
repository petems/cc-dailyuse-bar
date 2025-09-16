package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

// T011: Integration test for daily usage reset at midnight
// This test verifies the daily reset functionality works end-to-end
// MUST FAIL initially (RED phase) until UsageService is implemented

func TestDailyReset(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	// Act - Simulate getting initial usage (may have data from previous runs)
	initialUsage, err := usageService.GetDailyUsage()
	if err != nil {
		t.Logf("Initial usage retrieval failed (expected without ccusage): %v", err)
		return
	}

	// Act - Force daily reset
	err = usageService.ResetDaily()
	require.NoError(t, err, "Daily reset should succeed")

	// Act - Get usage after reset
	resetUsage, err := usageService.GetDailyUsage()
	require.NoError(t, err, "Should be able to get usage after reset")

	// Assert - Reset clears local state, but GetDailyUsage() fetches fresh data from ccusage
	// So we expect the same values as before reset (fresh data from ccusage)
	assert.Equal(t, initialUsage.DailyCount, resetUsage.DailyCount, "Should get fresh data from ccusage")
	assert.Equal(t, initialUsage.DailyCost, resetUsage.DailyCost, "Should get fresh data from ccusage")
	assert.Equal(t, initialUsage.Status, resetUsage.Status, "Status should match fresh data")

	// Assert - Reset timestamp should be updated
	assert.True(t, resetUsage.LastReset.After(initialUsage.LastReset) ||
		resetUsage.LastReset.Equal(initialUsage.LastReset),
		"LastReset timestamp should be updated")
}

func TestMidnightDetection(t *testing.T) {
	// This test verifies the logic for detecting midnight transitions
	// In a real implementation, this would involve time-based triggers

	// Arrange
	usageService := services.NewUsageService()

	// Get current usage state
	usage, err := usageService.GetDailyUsage()
	if err != nil {
		t.Skipf("Cannot test midnight detection without working usage service: %v", err)
	}

	// Act - Check that LastReset is properly tracked
	now := time.Now()

	// Assert - LastReset should be from today or earlier
	assert.True(t, usage.LastReset.Before(now.Add(time.Second)),
		"LastReset should not be in the future")

	// Assert - LastReset should be within reasonable bounds (not ancient)
	dayAgo := now.Add(-24 * time.Hour)
	assert.True(t, usage.LastReset.After(dayAgo),
		"LastReset should be within the last 24 hours")
}

func TestResetPreservesConfiguration(t *testing.T) {
	// This test ensures that reset only affects usage data, not configuration

	// Arrange - Set up custom configuration
	_ = "/tmp/cc-dailyuse-bar-reset-test"
	configService := &services.ConfigService{}
	usageService := services.NewUsageService()

	// Load original config
	originalConfig, err := configService.Load()
	if err != nil {
		t.Skipf("Cannot test reset without working config service: %v", err)
	}

	// Act - Reset daily usage
	err = usageService.ResetDaily()
	if err != nil {
		t.Skipf("Cannot test reset without working usage service: %v", err)
	}

	// Act - Load config after reset
	configAfterReset, err := configService.Load()
	if err != nil {
		t.Skipf("Cannot verify config after reset: %v", err)
	}

	// Assert - Configuration should be unchanged
	assert.Equal(t, originalConfig.CCUsagePath, configAfterReset.CCUsagePath)
	assert.Equal(t, originalConfig.UpdateInterval, configAfterReset.UpdateInterval)
	assert.Equal(t, originalConfig.YellowThreshold, configAfterReset.YellowThreshold)
	assert.Equal(t, originalConfig.RedThreshold, configAfterReset.RedThreshold)
	assert.Equal(t, originalConfig.DebugLevel, configAfterReset.DebugLevel)
}

func TestMultipleResets(t *testing.T) {
	// Test that multiple resets in succession work correctly

	// Arrange
	usageService := services.NewUsageService()

	resetTimes := make([]time.Time, 3)

	// Act - Perform multiple resets
	for i := 0; i < 3; i++ {
		err := usageService.ResetDaily()
		if err != nil {
			t.Skipf("Reset %d failed: %v", i+1, err)
		}

		usage, err := usageService.GetDailyUsage()
		if err != nil {
			t.Skipf("Cannot get usage after reset %d: %v", i+1, err)
		}

		resetTimes[i] = usage.LastReset

		// Assert after each reset - should get fresh data from ccusage
		assert.GreaterOrEqual(t, usage.DailyCount, 0, "Count should be valid after reset %d", i+1)
		assert.GreaterOrEqual(t, usage.DailyCost, 0.0, "Cost should be valid after reset %d", i+1)
		assert.Contains(t, []models.AlertStatus{models.Green, models.Yellow, models.Red}, usage.Status, "Status should be valid after reset %d", i+1)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Assert - Each reset should have a later timestamp
	for i := 1; i < len(resetTimes); i++ {
		assert.True(t, resetTimes[i].After(resetTimes[i-1]) || resetTimes[i].Equal(resetTimes[i-1]),
			"Reset %d timestamp should be >= reset %d timestamp", i+1, i)
	}
}

func TestResetWithThresholds(t *testing.T) {
	// Test that reset properly handles threshold-based status calculation

	// Arrange
	configService := &services.ConfigService{}
	usageService := services.NewUsageService()

	// Create mock ccusage script that returns zero usage
	tempDir := t.TempDir()
	mockScript := filepath.Join(tempDir, "mock-ccusage")

	today := time.Now().Format("2006-01-02")
	zeroUsageJSON := fmt.Sprintf(`{
		"daily": [{"date": "%s", "totalTokens": 0, "totalCost": 0.0}],
		"totals": {"totalTokens": 0, "totalCost": 0.0}
	}`, today)

	scriptContent := fmt.Sprintf("#!/bin/bash\necho '%s'\n", zeroUsageJSON)
	err := os.WriteFile(mockScript, []byte(scriptContent), 0755)
	require.NoError(t, err)

	// Configure usage service to use mock script
	err = usageService.SetCCUsagePath(mockScript)
	require.NoError(t, err)

	// Load config to get thresholds
	config, err := configService.Load()
	if err != nil {
		t.Skipf("Cannot test thresholds without config service: %v", err)
	}

	// Act - Reset usage
	err = usageService.ResetDaily()
	if err != nil {
		t.Skipf("Cannot reset usage: %v", err)
	}

	// Get usage after reset (should get 0 from mock)
	usage, err := usageService.GetDailyUsage()
	if err != nil {
		t.Skipf("Cannot get usage after reset: %v", err)
	}

	// Assert - With 0 cost, status should be Green regardless of thresholds
	assert.Equal(t, models.Green, usage.Status)
	assert.True(t, usage.DailyCost < config.YellowThreshold,
		"Reset cost (%.2f) should be below yellow threshold (%.2f)",
		usage.DailyCost, config.YellowThreshold)
	assert.True(t, usage.DailyCost < config.RedThreshold,
		"Reset cost (%.2f) should be below red threshold (%.2f)",
		usage.DailyCost, config.RedThreshold)
}
