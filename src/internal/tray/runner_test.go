package tray

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/internal/testhelpers"
	"cc-dailyuse-bar/src/models"
	"cc-dailyuse-bar/src/services"
)

func TestMain(m *testing.M) {
	os.Exit(testhelpers.RunSilenced(m))
}

func newTestRunner() *Runner {
	config := models.ConfigDefaults()
	usageService := services.NewUsageService(config)
	return NewRunner(config, usageService)
}

func TestEmojiForStatus(t *testing.T) {
	runner := newTestRunner()

	tests := []struct {
		status   models.AlertStatus
		expected string
	}{
		{models.Green, "🟢"},
		{models.Yellow, "🟡"},
		{models.Red, "🔴"},
		{models.Unknown, "⚪️"},
		{models.AlertStatus(99), "⚪️"}, // default case
	}

	for _, tt := range tests {
		t.Run(tt.status.String(), func(t *testing.T) {
			assert.Equal(t, tt.expected, runner.emojiForStatus(tt.status))
		})
	}
}

func TestNewRunner_Fields(t *testing.T) {
	config := models.ConfigDefaults()
	usageService := services.NewUsageService(config)

	runner := NewRunner(config, usageService)

	require.NotNil(t, runner)
	assert.Equal(t, config, runner.config)
	assert.Equal(t, usageService, runner.usageService)
	assert.NotNil(t, runner.menuItems)
	assert.NotNil(t, runner.logger)
}
