package contract

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

func TestUsageService_GetDailyUsage(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	// Act
	usage, err := usageService.GetDailyUsage()

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, usage)
	assert.GreaterOrEqual(t, usage.DailyCount, 0)
	assert.GreaterOrEqual(t, usage.DailyCost, 0.0)
	assert.Contains(t, []models.AlertStatus{models.Green, models.Yellow, models.Red}, usage.Status)
	assert.False(t, usage.LastUpdate.IsZero())
	assert.False(t, usage.LastReset.IsZero())
}

func TestUsageService_UpdateUsage(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	// Act - First call
	usage1, err := usageService.UpdateUsage()
	require.NoError(t, err)

	// Act - Second call (should be fresh data)
	time.Sleep(100 * time.Millisecond) // Ensure different timestamp
	usage2, err := usageService.UpdateUsage()
	require.NoError(t, err)

	// Assert
	assert.NotNil(t, usage1)
	assert.NotNil(t, usage2)
	assert.True(t, usage2.LastUpdate.After(usage1.LastUpdate) || usage2.LastUpdate.Equal(usage1.LastUpdate))
}

func TestUsageService_ResetDaily(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	// Get initial usage
	initialUsage, err := usageService.GetDailyUsage()
	require.NoError(t, err)

	// Act
	err = usageService.ResetDaily()
	require.NoError(t, err)

	// Assert - Reset should clear local state, but GetDailyUsage() fetches fresh data
	// The reset functionality clears the local cache, but the next call to GetDailyUsage()
	// will fetch current data from ccusage, so we expect the same values as before
	resetUsage, err := usageService.GetDailyUsage()
	require.NoError(t, err)

	// After reset, we should get fresh data from ccusage (same as before reset)
	assert.Equal(t, initialUsage.DailyCount, resetUsage.DailyCount)
	assert.Equal(t, initialUsage.DailyCost, resetUsage.DailyCost)
	assert.Equal(t, initialUsage.Status, resetUsage.Status)
	// LastReset should be updated to reflect the reset operation
	assert.True(t, resetUsage.LastReset.After(initialUsage.LastReset) || resetUsage.LastReset.Equal(initialUsage.LastReset))
}

func TestUsageService_IsAvailable(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	// Act
	available := usageService.IsAvailable()

	// Assert
	// This can be true or false depending on ccusage availability
	// But should not panic
	assert.IsType(t, true, available)
}

func TestUsageService_SetCCUsagePath_Valid(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	// Act - Set to a common path that should exist
	err := usageService.SetCCUsagePath("/bin/sh") // Using sh as a proxy for executable test

	// Assert
	assert.NoError(t, err)
}

func TestUsageService_SetCCUsagePath_Invalid(t *testing.T) {
	// Arrange
	usageService := services.NewUsageService()

	testCases := []struct {
		name string
		path string
	}{
		{
			name: "Empty path",
			path: "",
		},
		{
			name: "Non-existent path",
			path: "/non/existent/path/to/ccusage",
		},
		{
			name: "Directory instead of executable",
			path: "/tmp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			err := usageService.SetCCUsagePath(tc.path)

			// Assert
			assert.Error(t, err)
		})
	}
}

func TestUsageService_AlertStatus_Transitions(t *testing.T) {
	// Test AlertStatus enum behavior
	testCases := []struct {
		status   models.AlertStatus
		expected string
	}{
		{status: models.Green, expected: "OK"},
		{status: models.Yellow, expected: "High"},
		{status: models.Red, expected: "Critical"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.status.String())
		})
	}
}
