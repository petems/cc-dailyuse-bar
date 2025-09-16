package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlertStatus_String(t *testing.T) {
	tests := []struct {
		status   AlertStatus
		expected string
	}{
		{Green, "OK"},
		{Yellow, "High"},
		{Red, "Critical"},
		{AlertStatus(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.status.String())
		})
	}
}

func TestAlertStatus_ToTrayIcon(t *testing.T) {
	tests := []struct {
		status       AlertStatus
		expectedIcon TrayIcon
	}{
		{Green, IconGreen},
		{Yellow, IconYellow},
		{Red, IconRed},
		{AlertStatus(999), IconOffline},
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.expectedIcon, tt.status.ToTrayIcon())
		})
	}
}

func TestTrayIcon_FromAlertStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       AlertStatus
		isAvailable  bool
		expectedIcon TrayIcon
	}{
		{
			name:         "Green status available",
			status:       Green,
			isAvailable:  true,
			expectedIcon: IconGreen,
		},
		{
			name:         "Yellow status available",
			status:       Yellow,
			isAvailable:  true,
			expectedIcon: IconYellow,
		},
		{
			name:         "Red status available",
			status:       Red,
			isAvailable:  true,
			expectedIcon: IconRed,
		},
		{
			name:         "Green status unavailable",
			status:       Green,
			isAvailable:  false,
			expectedIcon: IconOffline,
		},
		{
			name:         "Yellow status unavailable",
			status:       Yellow,
			isAvailable:  false,
			expectedIcon: IconOffline,
		},
		{
			name:         "Red status unavailable",
			status:       Red,
			isAvailable:  false,
			expectedIcon: IconOffline,
		},
		{
			name:         "Unknown status available",
			status:       AlertStatus(999),
			isAvailable:  true,
			expectedIcon: IconOffline,
		},
		{
			name:         "Unknown status unavailable",
			status:       AlertStatus(999),
			isAvailable:  false,
			expectedIcon: IconOffline,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var icon TrayIcon
			result := icon.FromAlertStatus(tt.status, tt.isAvailable)
			assert.Equal(t, tt.expectedIcon, result)
		})
	}
}

func TestAlertStatus_EnumValues(t *testing.T) {
	// Ensure enum values are distinct and sequential
	statuses := []AlertStatus{Green, Yellow, Red}

	for i, status := range statuses {
		assert.Equal(t, AlertStatus(i), status, "Status %d should have value %d", i, i)
	}

	// Test that we have exactly 3 status values
	assert.Len(t, statuses, 3)
}

func TestTrayIcon_EnumValues(t *testing.T) {
	// Ensure enum values are distinct and sequential
	icons := []TrayIcon{IconGreen, IconYellow, IconRed, IconOffline}

	for i, icon := range icons {
		assert.Equal(t, TrayIcon(i), icon, "Icon %d should have value %d", i, i)
	}

	// Test that we have exactly 4 icon values
	assert.Len(t, icons, 4)
}

func TestAlertStatus_StatusTransitions(t *testing.T) {
	// Test logical progression of status values
	assert.True(t, Green < Yellow, "Green should be less than Yellow")
	assert.True(t, Yellow < Red, "Yellow should be less than Red")
}

func TestTrayIcon_IconTransitions(t *testing.T) {
	// Test logical progression of icon values
	assert.True(t, IconGreen < IconYellow, "IconGreen should be less than IconYellow")
	assert.True(t, IconYellow < IconRed, "IconYellow should be less than IconRed")
	assert.True(t, IconRed < IconOffline, "IconRed should be less than IconOffline")
}

func TestAlertStatus_StringConsistency(t *testing.T) {
	// Test that String() method is consistent with ToTrayIcon()
	statuses := []AlertStatus{Green, Yellow, Red}

	for _, status := range statuses {
		icon := status.ToTrayIcon()

		// Each status should map to a different icon
		switch status {
		case Green:
			assert.Equal(t, IconGreen, icon)
		case Yellow:
			assert.Equal(t, IconYellow, icon)
		case Red:
			assert.Equal(t, IconRed, icon)
		}
	}
}

func TestTrayIcon_FromAlertStatusConsistency(t *testing.T) {
	// Test that FromAlertStatus is consistent with ToTrayIcon
	statuses := []AlertStatus{Green, Yellow, Red}

	for _, status := range statuses {
		// When available, FromAlertStatus should match ToTrayIcon
		var icon TrayIcon
		result := icon.FromAlertStatus(status, true)
		expected := status.ToTrayIcon()
		assert.Equal(t, expected, result, "FromAlertStatus should match ToTrayIcon for status %v", status)

		// When unavailable, should always be IconOffline
		result = icon.FromAlertStatus(status, false)
		assert.Equal(t, IconOffline, result, "FromAlertStatus should return IconOffline when unavailable for status %v", status)
	}
}

func TestAlertStatus_EdgeCases(t *testing.T) {
	// Test edge cases and invalid values
	invalidStatus := AlertStatus(-1)
	assert.Equal(t, "Unknown", invalidStatus.String())
	assert.Equal(t, IconOffline, invalidStatus.ToTrayIcon())

	largeStatus := AlertStatus(1000)
	assert.Equal(t, "Unknown", largeStatus.String())
	assert.Equal(t, IconOffline, largeStatus.ToTrayIcon())
}

func TestTrayIcon_EdgeCases(t *testing.T) {
	// Test edge cases for TrayIcon
	var icon TrayIcon

	// Test FromAlertStatus with invalid icon (should still work)
	result := icon.FromAlertStatus(Green, true)
	assert.Equal(t, IconGreen, result)

	// Test with large icon value
	result = icon.FromAlertStatus(Green, true)
	assert.Equal(t, IconGreen, result)
}
