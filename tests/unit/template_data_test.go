package unit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"cc-dailyuse-bar/src/models"
)

// T034: Unit tests for TemplateData formatting

func TestNewTemplateData(t *testing.T) {
	// Create a usage state
	state := models.NewUsageState()
	state.DailyCount = 42
	state.DailyCost = 15.75
	state.Status = models.Yellow

	// Create template data
	data := models.NewTemplateData(state)

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
	status := models.Red

	data := models.NewTemplateDataWithCustomValues(count, cost, status)

	assert.Equal(t, 25, data.Count)
	assert.Equal(t, "$7.50", data.Cost)
	assert.Equal(t, "Critical", data.Status)
	assert.NotEmpty(t, data.Date)
	assert.NotEmpty(t, data.Time)
}

func TestTemplateData_CostFormatting(t *testing.T) {
	testCases := []struct {
		name         string
		inputCost    float64
		expectedCost string
	}{
		{name: "zero cost", inputCost: 0.0, expectedCost: "$0.00"},
		{name: "small cost", inputCost: 0.05, expectedCost: "$0.05"},
		{name: "normal cost", inputCost: 5.25, expectedCost: "$5.25"},
		{name: "large cost", inputCost: 123.45, expectedCost: "$123.45"},
		{name: "rounded cost", inputCost: 10.0, expectedCost: "$10.00"},
		{name: "long decimal", inputCost: 3.14159, expectedCost: "$3.14"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := models.NewTemplateDataWithCustomValues(10, tc.inputCost, models.Green)
			assert.Equal(t, tc.expectedCost, data.Cost)
		})
	}
}

func TestTemplateData_StatusFormatting(t *testing.T) {
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
			data := models.NewTemplateDataWithCustomValues(10, 5.0, tc.status)
			assert.Equal(t, tc.expected, data.Status)
		})
	}
}

func TestTemplateData_DateTimeFormatting(t *testing.T) {
	// Create template data
	data := models.NewTemplateDataWithCustomValues(10, 5.0, models.Green)

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
	state := models.NewUsageState()
	// State starts with zero values

	data := models.NewTemplateData(state)

	assert.Equal(t, 0, data.Count)
	assert.Equal(t, "$0.00", data.Cost)
	assert.Equal(t, "OK", data.Status) // Should be Green/OK
}

func TestTemplateData_HighUsageScenarios(t *testing.T) {
	testCases := []struct {
		name   string
		count  int
		cost   float64
		status models.AlertStatus
	}{
		{"moderate usage", 50, 8.75, models.Yellow},
		{"high usage", 100, 18.50, models.Red},
		{"extreme usage", 500, 87.25, models.Red},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := models.NewTemplateDataWithCustomValues(tc.count, tc.cost, tc.status)

			assert.Equal(t, tc.count, data.Count)
			assert.Equal(t, tc.status.String(), data.Status)
			// Cost should be formatted correctly
			assert.Contains(t, data.Cost, "$")
			assert.NotContains(t, data.Cost, "$0.00") // Should not be zero
		})
	}
}

func TestTemplateData_FromUsageState(t *testing.T) {
	// Test that template data correctly reflects usage state
	state := models.NewUsageState()
	state.DailyCount = 73
	state.DailyCost = 12.34
	state.Status = models.Red

	data := models.NewTemplateData(state)

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
	status := models.Yellow

	data1 := models.NewTemplateDataWithCustomValues(count, cost, status)
	data2 := models.NewTemplateDataWithCustomValues(count, cost, status)

	// These should be identical
	assert.Equal(t, data1.Count, data2.Count)
	assert.Equal(t, data1.Cost, data2.Cost)
	assert.Equal(t, data1.Status, data2.Status)
	assert.Equal(t, data1.Date, data2.Date)
	// Time might differ by a few seconds, so just check it's not empty
	assert.NotEmpty(t, data1.Time)
	assert.NotEmpty(t, data2.Time)
}
