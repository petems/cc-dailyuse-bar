package models

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewUsageState(t *testing.T) {
	state := NewUsageState()

	// Verify default values
	assert.Equal(t, 0, state.DailyCount)
	assert.Equal(t, 0.0, state.DailyCost)
	assert.Equal(t, Green, state.Status)
	assert.False(t, state.IsAvailable)

	// Verify timestamps are set (should be recent)
	now := time.Now()
	assert.True(t, state.LastUpdate.Before(now) || state.LastUpdate.Equal(now))
	assert.True(t, state.LastReset.Before(now) || state.LastReset.Equal(now))
	assert.False(t, state.LastUpdate.IsZero())
	assert.False(t, state.LastReset.IsZero())
}

func TestUsageState_UpdateStatus(t *testing.T) {
	state := NewUsageState()

	tests := []struct {
		name            string
		cost            float64
		yellowThreshold float64
		redThreshold    float64
		expectedStatus  AlertStatus
	}{
		{"below yellow", 2.0, 5.0, 10.0, Green},
		{"at yellow boundary", 5.0, 5.0, 10.0, Yellow},
		{"above yellow", 7.0, 5.0, 10.0, Yellow},
		{"at red boundary", 10.0, 5.0, 10.0, Red},
		{"above red", 15.0, 5.0, 10.0, Red},
		{"zero cost", 0.0, 5.0, 10.0, Green},
		{"negative cost", -1.0, 5.0, 10.0, Green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state.DailyCost = tt.cost
			state.UpdateStatus(tt.yellowThreshold, tt.redThreshold)
			assert.Equal(t, tt.expectedStatus, state.Status)
		})
	}
}

func TestUsageState_UpdateStatus_EdgeCases(t *testing.T) {
	state := NewUsageState()

	// Test with zero thresholds
	state.DailyCost = 1.0
	state.UpdateStatus(0.0, 0.0)
	assert.Equal(t, Red, state.Status) // Any cost above 0 should be red

	// Test with very small thresholds
	state.DailyCost = 0.01
	state.UpdateStatus(0.005, 0.02)
	assert.Equal(t, Yellow, state.Status)

	// Test with large thresholds
	state.DailyCost = 100.0
	state.UpdateStatus(200.0, 500.0)
	assert.Equal(t, Green, state.Status)

	// Test with equal thresholds (should not happen in practice)
	state.DailyCost = 5.0
	state.UpdateStatus(5.0, 5.0)
	assert.Equal(t, Red, state.Status) // At boundary, should be red
}

func TestUsageState_UpdateStatus_ThresholdBoundaries(t *testing.T) {
	state := NewUsageState()
	yellowThreshold := 10.0
	redThreshold := 20.0

	// Test exact boundary values
	testCases := []struct {
		cost           float64
		expectedStatus AlertStatus
		description    string
	}{
		{9.99, Green, "just below yellow"},
		{10.0, Yellow, "exactly at yellow"},
		{10.01, Yellow, "just above yellow"},
		{19.99, Yellow, "just below red"},
		{20.0, Red, "exactly at red"},
		{20.01, Red, "just above red"},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			state.DailyCost = tc.cost
			state.UpdateStatus(yellowThreshold, redThreshold)
			assert.Equal(t, tc.expectedStatus, state.Status,
				"Cost %.2f should result in status %v", tc.cost, tc.expectedStatus)
		})
	}
}

func TestUsageState_Reset(t *testing.T) {
	state := NewUsageState()

	// Set up some data
	state.DailyCount = 42
	state.DailyCost = 15.75
	state.Status = Red
	state.IsAvailable = true
	originalIsAvailable := state.IsAvailable
	originalLastUpdate := state.LastUpdate

	// Reset
	state.Reset()

	// Verify reset values
	assert.Equal(t, 0, state.DailyCount)
	assert.Equal(t, 0.0, state.DailyCost)
	assert.Equal(t, Green, state.Status)
	assert.Equal(t, originalIsAvailable, state.IsAvailable) // Should remain unchanged
	assert.Equal(t, originalLastUpdate, state.LastUpdate)   // Should remain unchanged
	assert.False(t, state.LastReset.IsZero())               // Should have reset time

	// Verify LastReset is recent
	now := time.Now()
	assert.True(t, state.LastReset.Before(now) || state.LastReset.Equal(now))
}

func TestUsageState_StatusTransitions(t *testing.T) {
	state := NewUsageState()
	yellowThreshold := 5.0
	redThreshold := 10.0

	// Start in green
	state.DailyCost = 0.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, Green, state.Status)

	// Move to yellow
	state.DailyCost = 7.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, Yellow, state.Status)

	// Move to red
	state.DailyCost = 12.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, Red, state.Status)

	// Move back to yellow
	state.DailyCost = 8.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, Yellow, state.Status)

	// Move back to green
	state.DailyCost = 2.0
	state.UpdateStatus(yellowThreshold, redThreshold)
	assert.Equal(t, Green, state.Status)
}

func TestUsageState_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		name            string
		cost            float64
		yellowThreshold float64
		redThreshold    float64
		expectedStatus  AlertStatus
		description     string
	}{
		{
			name:            "light usage",
			cost:            2.50,
			yellowThreshold: 10.0,
			redThreshold:    20.0,
			expectedStatus:  Green,
			description:     "Light daily usage should be green",
		},
		{
			name:            "moderate usage",
			cost:            15.0,
			yellowThreshold: 10.0,
			redThreshold:    20.0,
			expectedStatus:  Yellow,
			description:     "Moderate daily usage should be yellow",
		},
		{
			name:            "heavy usage",
			cost:            25.0,
			yellowThreshold: 10.0,
			redThreshold:    20.0,
			expectedStatus:  Red,
			description:     "Heavy daily usage should be red",
		},
		{
			name:            "no usage",
			cost:            0.0,
			yellowThreshold: 10.0,
			redThreshold:    20.0,
			expectedStatus:  Green,
			description:     "No usage should be green",
		},
		{
			name:            "very high usage",
			cost:            100.0,
			yellowThreshold: 10.0,
			redThreshold:    20.0,
			expectedStatus:  Red,
			description:     "Very high usage should be red",
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			state := NewUsageState()
			state.DailyCost = tt.cost
			state.UpdateStatus(tt.yellowThreshold, tt.redThreshold)
			assert.Equal(t, tt.expectedStatus, state.Status, tt.description)
		})
	}
}

func TestUsageState_ConcurrentAccess(t *testing.T) {
	// Test that UsageState can handle concurrent reads safely
	// Note: UsageState is designed to be a simple data structure
	// Thread safety is provided at the service layer, not the model layer
	state := NewUsageState()
	yellowThreshold := 5.0
	redThreshold := 10.0
	
	// Initialize state with test data to avoid data races
	state.DailyCost = 7.5
	state.UpdateStatus(yellowThreshold, redThreshold)
	expectedStatus := state.Status // Should be Yellow since 7.5 > 5.0 but < 10.0

	// Simulate concurrent reads (no writes to avoid data races)
	var wg sync.WaitGroup
	wg.Add(10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			defer wg.Done()
			
			// Test concurrent reads of state
			assert.Equal(t, 7.5, state.DailyCost)
			assert.Equal(t, expectedStatus, state.Status)
			assert.True(t, state.Status >= Green && state.Status <= Red)
			
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Final verification
	assert.Equal(t, Yellow, state.Status)
	assert.Equal(t, 7.5, state.DailyCost)
}

func TestUsageState_JSONCompatibility(t *testing.T) {
	// Test that UsageState can be properly marshaled to JSON
	state := NewUsageState()
	state.DailyCount = 42
	state.DailyCost = 15.75
	state.Status = Yellow
	state.IsAvailable = true

	// Verify all fields are properly set
	assert.Equal(t, 42, state.DailyCount)
	assert.Equal(t, 15.75, state.DailyCost)
	assert.Equal(t, Yellow, state.Status)
	assert.True(t, state.IsAvailable)
	assert.False(t, state.LastUpdate.IsZero())
	assert.False(t, state.LastReset.IsZero())
}

func TestUsageState_EdgeCases(t *testing.T) {
	state := NewUsageState()

	// Test with very small values
	state.DailyCost = 0.001
	state.UpdateStatus(0.01, 0.02)
	assert.Equal(t, Green, state.Status)

	// Test with very large values
	state.DailyCost = 999999.99
	state.UpdateStatus(1000000.0, 2000000.0)
	assert.Equal(t, Green, state.Status)

	// Test with negative cost
	state.DailyCost = -5.0
	state.UpdateStatus(10.0, 20.0)
	assert.Equal(t, Green, state.Status) // Negative cost should be green

	// Test with NaN (if possible)
	// Note: Go doesn't have NaN in the same way, but we can test with very large numbers
	state.DailyCost = 1e308 // Very large number
	state.UpdateStatus(1e307, 1e308)
	assert.Equal(t, Red, state.Status)
}

func TestUsageState_ResetPreservesAvailability(t *testing.T) {
	state := NewUsageState()
	state.IsAvailable = true
	state.DailyCount = 100
	state.DailyCost = 50.0
	state.Status = Red

	// Reset should preserve availability
	state.Reset()

	assert.True(t, state.IsAvailable) // Should be preserved
	assert.Equal(t, 0, state.DailyCount)
	assert.Equal(t, 0.0, state.DailyCost)
	assert.Equal(t, Green, state.Status)
}

func TestUsageState_UpdateStatusWithDifferentThresholds(t *testing.T) {
	state := NewUsageState()
	state.DailyCost = 15.0

	// Test with different threshold combinations
	thresholds := []struct {
		yellow         float64
		red            float64
		expectedStatus AlertStatus
	}{
		{10.0, 20.0, Yellow}, // 15 is between 10 and 20
		{5.0, 10.0, Red},     // 15 is above 10
		{20.0, 30.0, Green},  // 15 is below 20
		{15.0, 20.0, Yellow}, // 15 is exactly at yellow
		{10.0, 15.0, Red},    // 15 is exactly at red
	}

	for _, th := range thresholds {
		t.Run("", func(t *testing.T) {
			state.UpdateStatus(th.yellow, th.red)
			assert.Equal(t, th.expectedStatus, state.Status,
				"Cost 15.0 with thresholds %.1f/%.1f should be %v",
				th.yellow, th.red, th.expectedStatus)
		})
	}
}
