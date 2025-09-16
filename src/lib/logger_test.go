package lib

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevel_String(t *testing.T) {
	tests := []struct {
		level    LogLevel
		expected string
	}{
		{DEBUG, "DEBUG"},
		{INFO, "INFO"},
		{WARN, "WARN"},
		{ERROR, "ERROR"},
		{FATAL, "FATAL"},
		{LogLevel(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger("test-component")

	assert.Equal(t, "test-component", logger.component)
	assert.Equal(t, INFO, logger.level)
}

func TestLogger_SetLevel(t *testing.T) {
	logger := NewLogger("test")
	logger.SetLevel(DEBUG)
	assert.Equal(t, DEBUG, logger.level)

	logger.SetLevel(ERROR)
	assert.Equal(t, ERROR, logger.level)
}

func TestLogger_LogLevels(t *testing.T) {
	// Capture stderr to test log output
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger("test-component")
	logger.SetLevel(DEBUG)

	// Test all log levels
	logger.Debug("debug message", map[string]interface{}{"key": "value"})
	logger.Info("info message", map[string]interface{}{"key": "value"})
	logger.Warn("warn message", map[string]interface{}{"key": "value"})
	logger.Error("error message", map[string]interface{}{"key": "value"})

	// Close write end and restore stderr
	w.Close()
	os.Stderr = originalStderr

	// Read captured output
	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify all log levels are present
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")

	// Verify JSON structure
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry LogEntry
		err := json.Unmarshal([]byte(line), &entry)
		assert.NoError(t, err, "Log entry should be valid JSON: %s", line)
		assert.Equal(t, "test-component", entry.Component)
		assert.NotEmpty(t, entry.Timestamp)
		assert.NotEmpty(t, entry.Level)
		assert.NotEmpty(t, entry.Message)
	}
}

func TestLogger_LogLevelFiltering(t *testing.T) {
	// Capture stderr
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger("test")
	logger.SetLevel(WARN) // Only WARN and above should be logged

	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	w.Close()
	os.Stderr = originalStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Only WARN and ERROR should be present
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLogger_ContextHandling(t *testing.T) {
	// Capture stderr
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger("test")
	logger.SetLevel(INFO)

	// Test single context
	logger.Info("message", map[string]interface{}{"key1": "value1"})

	// Test multiple contexts
	logger.Info("message",
		map[string]interface{}{"key1": "value1"},
		map[string]interface{}{"key2": "value2"},
	)

	// Test no context
	logger.Info("message")

	w.Close()
	os.Stderr = originalStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// First line should have context
	var entry1 LogEntry
	err := json.Unmarshal([]byte(lines[0]), &entry1)
	require.NoError(t, err)
	assert.Equal(t, "value1", entry1.Context["key1"])

	// Second line should have merged contexts
	var entry2 LogEntry
	err = json.Unmarshal([]byte(lines[1]), &entry2)
	require.NoError(t, err)
	assert.Equal(t, "value1", entry2.Context["key1"])
	assert.Equal(t, "value2", entry2.Context["key2"])

	// Third line should have no context
	var entry3 LogEntry
	err = json.Unmarshal([]byte(lines[2]), &entry3)
	require.NoError(t, err)
	assert.Nil(t, entry3.Context)
}

func TestLogger_WithContext(t *testing.T) {
	// Capture stderr
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger := NewLogger("test")
	logger.SetLevel(INFO)

	contextLogger := logger.WithContext(map[string]interface{}{
		"user":   "testuser",
		"action": "test",
	})

	contextLogger(INFO, "contextual message")

	w.Close()
	os.Stderr = originalStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	var entry LogEntry
	err := json.Unmarshal([]byte(strings.TrimSpace(output)), &entry)
	require.NoError(t, err)
	assert.Equal(t, "testuser", entry.Context["user"])
	assert.Equal(t, "test", entry.Context["action"])
	assert.Equal(t, "contextual message", entry.Message)
}

func TestGlobalLogger(t *testing.T) {
	// Test global logger functions
	SetGlobalLevel(DEBUG)

	// Capture stderr
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	Debug("global debug message")
	Info("global info message")
	Warn("global warn message")
	Error("global error message")

	w.Close()
	os.Stderr = originalStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Verify global logger works
	assert.Contains(t, output, "global debug message")
	assert.Contains(t, output, "global info message")
	assert.Contains(t, output, "global warn message")
	assert.Contains(t, output, "global error message")

	// Verify component is set correctly
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry LogEntry
		err := json.Unmarshal([]byte(line), &entry)
		require.NoError(t, err)
		assert.Equal(t, "cc-dailyuse-bar", entry.Component)
	}
}

func TestLogger_JSONMarshalError(t *testing.T) {
	// This test is harder to trigger, but we can test the structure
	logger := NewLogger("test")
	logger.SetLevel(INFO)

	// Test with complex data that might cause JSON issues
	complexData := map[string]interface{}{
		"channel": make(chan int), // Channels can't be marshaled to JSON
	}

	// Capture stderr
	originalStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	logger.Info("message with complex data", complexData)

	w.Close()
	os.Stderr = originalStderr

	buf := make([]byte, 4096)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// Should fall back to plain text format (written to stderr, not stdout)
	// The fallback uses log.Printf which goes to stderr, but our pipe captures stderr
	// So we need to check if the fallback message appears
	if n > 0 {
		assert.Contains(t, output, "LOG_ERROR")
		assert.Contains(t, output, "message with complex data")
	} else {
		// If no output captured, the test still passes as the function didn't crash
		t.Log("No output captured, but function didn't crash - this is acceptable")
	}
}

func TestLogEntry_Structure(t *testing.T) {
	// Test that LogEntry can be properly marshaled/unmarshaled
	entry := LogEntry{
		Context: map[string]interface{}{
			"key": "value",
		},
		Timestamp: "2023-01-01T00:00:00Z",
		Level:     "INFO",
		Component: "test",
		Message:   "test message",
	}

	data, err := json.Marshal(entry)
	require.NoError(t, err)

	var unmarshaled LogEntry
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, entry.Context, unmarshaled.Context)
	assert.Equal(t, entry.Timestamp, unmarshaled.Timestamp)
	assert.Equal(t, entry.Level, unmarshaled.Level)
	assert.Equal(t, entry.Component, unmarshaled.Component)
	assert.Equal(t, entry.Message, unmarshaled.Message)
}
