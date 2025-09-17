// Package models contains domain models and configuration types.
package models

// AlertStatus represents the current alert level.
type AlertStatus int

const (
	Green   AlertStatus = iota // Usage below yellow threshold
	Yellow                     // Usage above yellow, below red threshold
	Red                        // Usage above red threshold
	Unknown                    // Usage data unavailable or invalid
)

// String returns human-readable alert status.
func (a AlertStatus) String() string {
	switch a {
	case Green:
		return "OK"
	case Yellow:
		return "High"
	case Red:
		return "Critical"
	case Unknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// ToTrayIcon converts an AlertStatus to the corresponding TrayIcon.
func (a AlertStatus) ToTrayIcon() TrayIcon {
	switch a {
	case Green:
		return IconGreen
	case Yellow:
		return IconYellow
	case Red:
		return IconRed
	case Unknown:
		return IconOffline
	default:
		return IconOffline
	}
}

// TrayIcon represents different icon states for the system tray.
type TrayIcon int

const (
	IconGreen   TrayIcon = iota // Normal usage level
	IconYellow                  // Warning usage level
	IconRed                     // Critical usage level
	IconOffline                 // ccusage unavailable
)

// FromAlertStatus converts an AlertStatus to the corresponding TrayIcon.
func (t TrayIcon) FromAlertStatus(status AlertStatus, isAvailable bool) TrayIcon {
	if !isAvailable {
		return IconOffline
	}

	switch status {
	case Green:
		return IconGreen
	case Yellow:
		return IconYellow
	case Red:
		return IconRed
	case Unknown:
		return IconOffline
	default:
		return IconOffline
	}
}
