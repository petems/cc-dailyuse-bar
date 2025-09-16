package models

import (
	"fmt"
	"time"
)

// TemplateData represents data available to display format templates
type TemplateData struct {
	Cost   string `json:"cost"`
	Status string `json:"status"`
	Date   string `json:"date"`
	Time   string `json:"time"`
	Count  int    `json:"count"`
}

// NewTemplateData creates TemplateData from a UsageState
func NewTemplateData(usage *UsageState) *TemplateData {
	now := time.Now()

	return &TemplateData{
		Count:  usage.DailyCount,
		Cost:   fmt.Sprintf("$%.2f", usage.DailyCost),
		Status: usage.Status.String(),
		Date:   now.Format("2006-01-02"),
		Time:   now.Format("15:04"),
	}
}

// NewTemplateDataWithCustomValues creates TemplateData with specific values
// Used for testing and custom scenarios
func NewTemplateDataWithCustomValues(count int, cost float64, status AlertStatus) *TemplateData {
	now := time.Now()

	return &TemplateData{
		Count:  count,
		Cost:   fmt.Sprintf("$%.2f", cost),
		Status: status.String(),
		Date:   now.Format("2006-01-02"),
		Time:   now.Format("15:04"),
	}
}
