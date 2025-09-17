// Package lib provides shared utilities like error handling and logging.
package lib

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// AppError represents an application error with context.
type AppError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context,omitempty"`
	Cause     error                  `json:"cause,omitempty"`
	Component string                 `json:"component"`
	Function  string                 `json:"function"`
	File      string                 `json:"file"`
	Line      int                    `json:"line"`
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error for Go 1.13+ error handling.
func (e *AppError) Unwrap() error {
	return e.Cause
}

// NewError creates a new AppError with caller information.
func NewError(code, message string) *AppError {
	pc, file, line, _ := runtime.Caller(1)
	function := runtime.FuncForPC(pc).Name()

	// Extract component from file path
	component := extractComponent(file)

	return &AppError{
		Code:      code,
		Message:   message,
		Component: component,
		Function:  function,
		File:      file,
		Line:      line,
	}
}

// WrapError wraps an existing error with additional context.
func WrapError(err error, code, message string) *AppError {
	if err == nil {
		return nil
	}

	pc, file, line, _ := runtime.Caller(1)
	function := runtime.FuncForPC(pc).Name()
	component := extractComponent(file)

	return &AppError{
		Code:      code,
		Message:   message,
		Cause:     err,
		Component: component,
		Function:  function,
		File:      file,
		Line:      line,
	}
}

// WithContext adds context information to an error.
func (e *AppError) WithContext(key string, value interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithContextMap adds multiple context fields to an error.
func (e *AppError) WithContextMap(context map[string]interface{}) *AppError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	for k, v := range context {
		e.Context[k] = v
	}
	return e
}

// extractComponent extracts component name from file path.
func extractComponent(file string) string {
	// Extract component from path like /path/to/cc-dailyuse-bar/src/services/config_service.go
	parts := strings.Split(file, "/")
	for i, part := range parts {
		if part == "src" && i+1 < len(parts) {
			return parts[i+1] // Return "services", "models", etc.
		}
	}
	return "unknown"
}

// Common error codes.
const (
	ErrCodeConfig     = "CONFIG_ERROR"
	ErrCodeUsage      = "USAGE_ERROR"
	ErrCodeUI         = "UI_ERROR"
	ErrCodeCCUsage    = "CCUSAGE_ERROR"
	ErrCodeValidation = "VALIDATION_ERROR"
	ErrCodeSystem     = "SYSTEM_ERROR"
	ErrCodeTemplate   = "TEMPLATE_ERROR"
)

// Convenience functions for common error types

// ConfigError creates a configuration-related error.
func ConfigError(message string) *AppError {
	return NewError(ErrCodeConfig, message)
}

// UsageError creates a usage-related error.
func UsageError(message string) *AppError {
	return NewError(ErrCodeUsage, message)
}

// UIError creates a UI-related error.
func UIError(message string) *AppError {
	return NewError(ErrCodeUI, message)
}

// CCUsageError creates a ccusage-related error.
func CCUsageError(message string) *AppError {
	return NewError(ErrCodeCCUsage, message)
}

// ValidationError creates a validation-related error.
func ValidationError(message string) *AppError {
	return NewError(ErrCodeValidation, message)
}

// SystemError creates a system-related error.
func SystemError(message string) *AppError {
	return NewError(ErrCodeSystem, message)
}

// TemplateError creates a template-related error.
func TemplateError(message string) *AppError {
	return NewError(ErrCodeTemplate, message)
}

// IsErrorCode checks if an error has a specific error code.
func IsErrorCode(err error, code string) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code == code
	}
	return false
}

// GetErrorCode returns the error code from an AppError, or empty string for other errors.
func GetErrorCode(err error) string {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.Code
	}
	return ""
}
