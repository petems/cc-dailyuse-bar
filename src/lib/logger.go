package lib

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel represents the severity of log messages
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger provides structured JSON logging with context
type Logger struct {
	component string
	level     LogLevel
	writer    io.Writer
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Context   map[string]interface{} `json:"context,omitempty"`
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Component string                 `json:"component"`
	Message   string                 `json:"message"`
}

// NewLogger creates a new logger for the specified component
func NewLogger(component string) *Logger {
	return &Logger{
		component: component,
		level:     INFO,
		writer:    getDefaultWriter(),
	}
}

var (
	defaultWriter    io.Writer = os.Stderr
	defaultWriterMux sync.RWMutex
)

func getDefaultWriter() io.Writer {
	defaultWriterMux.RLock()
	defer defaultWriterMux.RUnlock()
	return defaultWriter
}

func setDefaultWriter(writer io.Writer) {
	if writer == nil {
		writer = io.Discard
	}
	defaultWriterMux.Lock()
	defer defaultWriterMux.Unlock()
	defaultWriter = writer
}

// SetLevel sets the minimum log level
func (l *Logger) SetLevel(level LogLevel) {
	l.level = level
}

// SetOutput sets the destination writer for this logger instance
func (l *Logger) SetOutput(writer io.Writer) {
	if writer == nil {
		writer = io.Discard
	}
	l.writer = writer
}

// Debug logs a debug message with optional context
func (l *Logger) Debug(message string, context ...map[string]interface{}) {
	l.log(DEBUG, message, context...)
}

// Info logs an info message with optional context
func (l *Logger) Info(message string, context ...map[string]interface{}) {
	l.log(INFO, message, context...)
}

// Warn logs a warning message with optional context
func (l *Logger) Warn(message string, context ...map[string]interface{}) {
	l.log(WARN, message, context...)
}

// Error logs an error message with optional context
func (l *Logger) Error(message string, context ...map[string]interface{}) {
	l.log(ERROR, message, context...)
}

// Fatal logs a fatal message and exits the program
func (l *Logger) Fatal(message string, context ...map[string]interface{}) {
	l.log(FATAL, message, context...)
	os.Exit(1)
}

// log performs the actual logging with structured JSON output
func (l *Logger) log(level LogLevel, message string, context ...map[string]interface{}) {
	if level < l.level {
		return
	}

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Component: l.component,
		Message:   message,
	}

	// Merge all context maps
	if len(context) > 0 {
		entry.Context = make(map[string]interface{})
		for _, ctx := range context {
			for k, v := range ctx {
				entry.Context[k] = v
			}
		}
	}

	// Output as JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		// Fallback to plain text if JSON marshaling fails
		log.Printf("LOG_ERROR: %s [%s] %s: %s (JSON marshal error: %v)",
			entry.Timestamp, entry.Level, entry.Component, message, err)
		return
	}

	// Write to configured destination for structured logging
	writer := l.writer
	if writer == nil {
		writer = getDefaultWriter()
	}
	fmt.Fprintln(writer, string(jsonData))
}

// WithContext creates a convenience function for logging with common context
func (l *Logger) WithContext(context map[string]interface{}) func(LogLevel, string) {
	return func(level LogLevel, message string) {
		l.log(level, message, context)
	}
}

// Global logger instance for convenience
var globalLogger = NewLogger("cc-dailyuse-bar")

// SetGlobalLevel sets the global logger level
func SetGlobalLevel(level LogLevel) {
	globalLogger.SetLevel(level)
}

// SetGlobalOutput sets the output writer for global logging and future loggers
func SetGlobalOutput(writer io.Writer) {
	setDefaultWriter(writer)
	globalLogger.SetOutput(writer)
}

// Debug logs using the global logger
func Debug(message string, context ...map[string]interface{}) {
	globalLogger.Debug(message, context...)
}

// Info logs using the global logger
func Info(message string, context ...map[string]interface{}) {
	globalLogger.Info(message, context...)
}

// Warn logs using the global logger
func Warn(message string, context ...map[string]interface{}) {
	globalLogger.Warn(message, context...)
}

// Error logs using the global logger
func Error(message string, context ...map[string]interface{}) {
	globalLogger.Error(message, context...)
}

// Fatal logs using the global logger and exits
func Fatal(message string, context ...map[string]interface{}) {
	globalLogger.Fatal(message, context...)
}
