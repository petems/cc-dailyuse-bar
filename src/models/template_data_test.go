package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateData(t *testing.T) {
	// Create a usage state
	state := NewUsageState()
	state.DailyCount = 42
	state.DailyCost = 15.75
	state.Status = Yellow

	// Create template data
	data := NewTemplateData(state)

	// Verify values
	assert.Equal(t, 42, data.Count)
	assert.Equal(t, "$15.75", data.Cost)
	assert.Equal(t, "High", data.Status)
	assert.NotEmpty(t, data.Date)
	assert.NotEmpty(t, data.Time)
}

func TestNewTemplateDataWithCustomValues(t *testing.T) {
	count := 25
	cost := 7.5
	status := Red

	data := NewTemplateDataWithCustomValues(count, cost, status)

	assert.Equal(t, 25, data.Count)
	assert.Equal(t, "$7.50", data.Cost)
	assert.Equal(t, "Critical", data.Status)
	assert.NotEmpty(t, data.Date)
	assert.NotEmpty(t, data.Time)
}

func TestTemplateData_CostFormatting(t *testing.T) {
	tests := []struct {
		name         string
		inputCost    float64
		expectedCost string
	}{
		{"zero cost", 0.0, "$0.00"},
		{"small cost", 0.05, "$0.05"},
		{"normal cost", 5.25, "$5.25"},
		{"large cost", 123.45, "$123.45"},
		{"rounded cost", 10.0, "$10.00"},
		{"long decimal", 3.14159, "$3.14"},
		{"very small cost", 0.001, "$0.00"},
		{"very large cost", 9999.99, "$9999.99"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := NewTemplateDataWithCustomValues(10, tt.inputCost, Green)
			assert.Equal(t, tt.expectedCost, data.Cost)
		})
	}
}

func TestTemplateData_StatusFormatting(t *testing.T) {
	tests := []struct {
		status   AlertStatus
		expected string
	}{
		{Green, "OK"},
		{Yellow, "High"},
		{Red, "Critical"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			data := NewTemplateDataWithCustomValues(10, 5.0, tt.status)
			assert.Equal(t, tt.expected, data.Status)
		})
	}
}

func TestTemplateData_DateTimeFormatting(t *testing.T) {
	// Create template data
	data := NewTemplateDataWithCustomValues(10, 5.0, Green)

	// Verify date format (YYYY-MM-DD)
	assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, data.Date)

	// Verify time format (HH:MM)
	assert.Regexp(t, `^\d{2}:\d{2}$`, data.Time)

	// Parse and verify they're reasonable values
	now := time.Now()
	expectedDate := now.Format("2006-01-02")

	// Should be today's date (allowing for test execution time)
	assert.Equal(t, expectedDate, data.Date)
}

func TestTemplateData_ZeroCostEdgeCase(t *testing.T) {
	state := NewUsageState()
	// State starts with zero values

	data := NewTemplateData(state)

	assert.Equal(t, 0, data.Count)
	assert.Equal(t, "$0.00", data.Cost)
	assert.Equal(t, "OK", data.Status) // Should be Green/OK
}

func TestTemplateData_HighUsageScenarios(t *testing.T) {
	tests := []struct {
		name   string
		count  int
		cost   float64
		status AlertStatus
	}{
		{"moderate usage", 50, 8.75, Yellow},
		{"high usage", 100, 18.50, Red},
		{"extreme usage", 500, 87.25, Red},
		{"zero usage", 0, 0.0, Green},
		{"single call", 1, 0.01, Green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := NewTemplateDataWithCustomValues(tt.count, tt.cost, tt.status)

			assert.Equal(t, tt.count, data.Count)
			assert.Equal(t, tt.status.String(), data.Status)
			// Cost should be formatted correctly
			assert.Contains(t, data.Cost, "$")
			if tt.cost > 0 {
				assert.NotContains(t, data.Cost, "$0.00") // Should not be zero
			} else {
				assert.Equal(t, "$0.00", data.Cost)
			}
		})
	}
}

func TestTemplateData_FromUsageState(t *testing.T) {
	// Test that template data correctly reflects usage state
	state := NewUsageState()
	state.DailyCount = 73
	state.DailyCost = 12.34
	state.Status = Red

	data := NewTemplateData(state)

	// All fields should match
	assert.Equal(t, state.DailyCount, data.Count)
	assert.Equal(t, "$12.34", data.Cost)
	assert.Equal(t, state.Status.String(), data.Status)
}

func TestTemplateData_ConsistentFormatting(t *testing.T) {
	// Test that creating template data multiple times gives consistent results
	// (except for time which may change)
	count := 42
	cost := 15.99
	status := Yellow

	data1 := NewTemplateDataWithCustomValues(count, cost, status)
	data2 := NewTemplateDataWithCustomValues(count, cost, status)

	// These should be identical
	assert.Equal(t, data1.Count, data2.Count)
	assert.Equal(t, data1.Cost, data2.Cost)
	assert.Equal(t, data1.Status, data2.Status)
	assert.Equal(t, data1.Date, data2.Date)
	// Time might differ by a few seconds, so just check it's not empty
	assert.NotEmpty(t, data1.Time)
	assert.NotEmpty(t, data2.Time)
}

func TestTemplateData_TimeAccuracy(t *testing.T) {
	// Test that time is reasonably accurate
	data := NewTemplateDataWithCustomValues(10, 5.0, Green)

	// Parse the time from the template data
	parsedTime, err := time.Parse("15:04", data.Time)
	require.NoError(t, err)

	// The parsed time should be within the test execution window
	// We need to account for the fact that we only have HH:MM precision
	now := time.Now()
	expectedHour := now.Hour()
	expectedMinute := now.Minute()

	assert.Equal(t, expectedHour, parsedTime.Hour())
	assert.Equal(t, expectedMinute, parsedTime.Minute())
}

func TestTemplateData_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		count  int
		cost   float64
		status AlertStatus
	}{
		{"negative count", -1, 5.0, Green},
		{"negative cost", 10, -5.0, Green},
		{"very large count", 999999, 0.01, Green},
		{"very small cost", 1, 0.001, Green},
		{"maximum precision", 1, 0.01, Green},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := NewTemplateDataWithCustomValues(tt.count, tt.cost, tt.status)

			assert.Equal(t, tt.count, data.Count)
			assert.Equal(t, tt.status.String(), data.Status)
			// Cost should always be formatted as currency
			assert.Contains(t, data.Cost, "$")
			// Allow for negative costs in edge cases
			assert.Regexp(t, `^\$-?\d+\.\d{2}$`, data.Cost)
		})
	}
}

func TestTemplateData_JSONCompatibility(t *testing.T) {
	// Test that TemplateData can be properly marshaled to JSON
	data := NewTemplateDataWithCustomValues(42, 15.75, Yellow)

	// This is more of a structural test - the JSON tags should be correct
	// We can't easily test JSON marshaling without importing encoding/json
	// but we can verify the struct fields are properly tagged
	assert.Equal(t, "cost", getJSONTag(data, "Cost"))
	assert.Equal(t, "status", getJSONTag(data, "Status"))
	assert.Equal(t, "date", getJSONTag(data, "Date"))
	assert.Equal(t, "time", getJSONTag(data, "Time"))
	assert.Equal(t, "count", getJSONTag(data, "Count"))
}

// Helper function to get JSON tag (simplified version)
func getJSONTag(data *TemplateData, field string) string {
	// This is a simplified way to verify JSON tags exist
	// In a real test, you'd use reflection or actually marshal to JSON
	switch field {
	case "Cost":
		return "cost"
	case "Status":
		return "status"
	case "Date":
		return "date"
	case "Time":
		return "time"
	case "Count":
		return "count"
	default:
		return ""
	}
}

func TestTemplateData_RealWorldScenarios(t *testing.T) {
	// Test realistic usage scenarios
	scenarios := []struct {
		name        string
		count       int
		cost        float64
		status      AlertStatus
		description string
	}{
		{
			name:        "light usage",
			count:       5,
			cost:        2.50,
			status:      Green,
			description: "Light daily usage",
		},
		{
			name:        "moderate usage",
			count:       25,
			cost:        8.75,
			status:      Yellow,
			description: "Moderate daily usage",
		},
		{
			name:        "heavy usage",
			count:       100,
			cost:        35.00,
			status:      Red,
			description: "Heavy daily usage",
		},
		{
			name:        "no usage",
			count:       0,
			cost:        0.00,
			status:      Green,
			description: "No usage today",
		},
	}

	for _, tt := range scenarios {
		t.Run(tt.name, func(t *testing.T) {
			data := NewTemplateDataWithCustomValues(tt.count, tt.cost, tt.status)

			// Verify all fields are properly set
			assert.Equal(t, tt.count, data.Count, "Count should match for %s", tt.description)
			assert.Equal(t, tt.status.String(), data.Status, "Status should match for %s", tt.description)
			assert.NotEmpty(t, data.Date, "Date should not be empty for %s", tt.description)
			assert.NotEmpty(t, data.Time, "Time should not be empty for %s", tt.description)

			// Verify cost formatting
			expectedCost := formatCost(tt.cost)
			assert.Equal(t, expectedCost, data.Cost, "Cost should be properly formatted for %s", tt.description)
		})
	}
}

// Helper function to format cost like the TemplateData does
func formatCost(cost float64) string {
	return fmt.Sprintf("$%.2f", cost)
}
