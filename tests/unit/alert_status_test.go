package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"cc-dailyuse-bar/src/models"
)

func TestAlertStatus_String(t *testing.T) {
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

func TestAlertStatus_ToTrayIcon(t *testing.T) {
	testCases := []struct {
		status       models.AlertStatus
		expectedIcon models.TrayIcon
	}{
		{models.Green, models.IconGreen},
		{models.Yellow, models.IconYellow},
		{models.Red, models.IconRed},
	}

	for _, tc := range testCases {
		t.Run(tc.status.String(), func(t *testing.T) {
			assert.Equal(t, tc.expectedIcon, tc.status.ToTrayIcon())
		})
	}
}

func TestUsageState_UpdateStatus(t *testing.T) {
	state := models.NewUsageState()

	testCases := []struct {
		name            string
		cost            float64
		yellowThreshold float64
		redThreshold    float64
		expectedStatus  models.AlertStatus
	}{
		{"below yellow", 2.0, 5.0, 10.0, models.Green},
		{"at yellow boundary", 5.0, 5.0, 10.0, models.Yellow},
		{"above yellow", 7.0, 5.0, 10.0, models.Yellow},
		{"at red boundary", 10.0, 5.0, 10.0, models.Red},
		{"above red", 15.0, 5.0, 10.0, models.Red},
		{"zero cost", 0.0, 5.0, 10.0, models.Green},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			state.DailyCost = tc.cost
			state.UpdateStatus(tc.yellowThreshold, tc.redThreshold)
			assert.Equal(t, tc.expectedStatus, state.Status)
		})
	}
}

func TestUsageState_UpdateStatus_EdgeCases(t *testing.T) {
	state := models.NewUsageState()

	// Test with zero thresholds
	state.DailyCost = 1.0
	state.UpdateStatus(0.0, 0.0)
	assert.Equal(t, models.Red, state.Status) // Any cost above 0 should be red

	// Test with very small thresholds
	state.DailyCost = 0.01
	state.UpdateStatus(0.005, 0.02)
	assert.Equal(t, models.Yellow, state.Status)

	// Test with large thresholds
	state.DailyCost = 100.0
	state.UpdateStatus(200.0, 500.0)
	assert.Equal(t, models.Green, state.Status)
}

func TestUsageState_Reset(t *testing.T) {
	state := models.NewUsageState()

	// Set up some data
	state.DailyCount = 42
	state.DailyCost = 15.75
	state.Status = models.Red
	state.IsAvailable = true
	originalIsAvailable := state.IsAvailable

	// Reset
	state.Reset()

	// Verify reset values
	assert.Equal(t, 0, state.DailyCount)
	assert.Equal(t, 0.0, state.DailyCost)
	assert.Equal(t, models.Green, state.Status)
	assert.Equal(t, originalIsAvailable, state.IsAvailable) // Should remain unchanged
	assert.False(t, state.LastReset.IsZero())               // Should have reset time
}

func TestAlertStatus_EnumValues(t *testing.T) {
	// Ensure enum values are distinct
	statuses := []models.AlertStatus{models.Green, models.Yellow, models.Red}

	statusMap := make(map[models.AlertStatus]bool)
	for _, status := range statuses {
		assert.False(t, statusMap[status], "Duplicate status value: %d", int(status))
		statusMap[status] = true
	}

	// Should have 3 distinct values
	assert.Len(t, statusMap, 3)
}

func TestTrayIcon_EnumValues(t *testing.T) {
	// Ensure tray icon values are distinct
	icons := []models.TrayIcon{
		models.IconGreen,
		models.IconYellow,
		models.IconRed,
		models.IconOffline,
	}

	iconMap := make(map[models.TrayIcon]bool)
	for _, icon := range icons {
		assert.False(t, iconMap[icon], "Duplicate icon value: %d", int(icon))
		iconMap[icon] = true
	}

	// Should have 4 distinct values
	assert.Len(t, iconMap, 4)
}

func TestUsageState_StatusTransitions(t *testing.T) {
	state := models.NewUsageState()
	yellowThreshold := 5.0
	redThreshold := 10.0

	// Start in green
	state.DailyCost = 0.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, models.Green, state.Status)

	// Move to yellow
	state.DailyCost = 7.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, models.Yellow, state.Status)

	// Move to red
	state.DailyCost = 12.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, models.Red, state.Status)

	// Move back to yellow
	state.DailyCost = 8.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, models.Yellow, state.Status)

	// Move back to green
	state.DailyCost = 2.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, models.Green, state.Status)
}
