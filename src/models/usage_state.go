package models

import "time"

// UsageState represents the current usage tracking state
type UsageState struct {
	LastUpdate  time.Time   `json:"last_update"`
	LastReset   time.Time   `json:"last_reset"`
	DailyCount  int         `json:"daily_count"`
	DailyCost   float64     `json:"daily_cost"`
	Status      AlertStatus `json:"status"`
	IsAvailable bool        `json:"is_available"`
}

// NewUsageState creates a new UsageState with default values
func NewUsageState() *UsageState {
	now := time.Now()
	return &UsageState{
		DailyCount:  0,
		DailyCost:   0.0,
		Status:      Green,
		LastUpdate:  now,
		LastReset:   now,
		IsAvailable: false,
	}
}

// UpdateStatus calculates and updates the alert status based on cost thresholds
func (u *UsageState) UpdateStatus(yellowThreshold, redThreshold float64) {
	if u.DailyCost >= redThreshold {
		u.Status = Red
	} else if u.DailyCost >= yellowThreshold {
		u.Status = Yellow
	} else {
		u.Status = Green
	}
}

// Reset resets the daily counters while preserving other state
func (u *UsageState) Reset() {
	u.DailyCount = 0
	u.DailyCost = 0.0
	u.Status = Green
	u.LastReset = time.Now()
}
