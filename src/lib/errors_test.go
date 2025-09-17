package lib

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appError *AppError
		expected string
	}{
		{
			name: "error without cause",
			appError: &AppError{
				Code:    "TEST_ERROR",
				Message: "Test message",
			},
			expected: "[TEST_ERROR] Test message",
		},
		{
			name: "error with cause",
			appError: &AppError{
				Code:    "TEST_ERROR",
				Message: "Test message",
				Cause:   errors.New("underlying error"),
			},
			expected: "[TEST_ERROR] Test message: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.appError.Error())
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	appErr := &AppError{
		Code:    "TEST_ERROR",
		Message: "Test message",
		Cause:   underlyingErr,
	}

	assert.Equal(t, underlyingErr, appErr.Unwrap())
}

func TestNewError(t *testing.T) {
	err := NewError("TEST_ERROR", "Test message")

	assert.Equal(t, "TEST_ERROR", err.Code)
	assert.Equal(t, "Test message", err.Message)
	assert.NotEmpty(t, err.Component)
	assert.NotEmpty(t, err.Function)
	assert.NotEmpty(t, err.File)
	assert.Positive(t, err.Line)
}

func TestWrapError(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	wrappedErr := WrapError(underlyingErr, "TEST_ERROR", "Test message")

	assert.Equal(t, "TEST_ERROR", wrappedErr.Code)
	assert.Equal(t, "Test message", wrappedErr.Message)
	assert.Equal(t, underlyingErr, wrappedErr.Cause)
	assert.NotEmpty(t, wrappedErr.Component)
	assert.NotEmpty(t, wrappedErr.Function)
	assert.NotEmpty(t, wrappedErr.File)
	assert.Positive(t, wrappedErr.Line)
}

func TestWrapError_NilError(t *testing.T) {
	wrappedErr := WrapError(nil, "TEST_ERROR", "Test message")
	assert.Nil(t, wrappedErr)
}

func TestAppError_WithContext(t *testing.T) {
	err := &AppError{
		Code:    "TEST_ERROR",
		Message: "Test message",
	}

	// Add single context
	err = err.WithContext("key1", "value1")
	require.NotNil(t, err.Context)
	assert.Equal(t, "value1", err.Context["key1"])

	// Add another context
	err = err.WithContext("key2", "value2")
	assert.Equal(t, "value1", err.Context["key1"])
	assert.Equal(t, "value2", err.Context["key2"])
}

func TestAppError_WithContextMap(t *testing.T) {
	err := &AppError{
		Code:    "TEST_ERROR",
		Message: "Test message",
	}

	context := map[string]interface{}{
		"key1": "value1",
		"key2": "value2",
		"key3": 123,
	}

	err = err.WithContextMap(context)
	require.NotNil(t, err.Context)
	assert.Equal(t, "value1", err.Context["key1"])
	assert.Equal(t, "value2", err.Context["key2"])
	assert.Equal(t, 123, err.Context["key3"])
}

func TestAppError_WithContextMap_ExistingContext(t *testing.T) {
	err := &AppError{
		Code:    "TEST_ERROR",
		Message: "Test message",
		Context: map[string]interface{}{
			"existing": "value",
		},
	}

	context := map[string]interface{}{
		"new": "value",
	}

	err = err.WithContextMap(context)
	assert.Equal(t, "value", err.Context["existing"])
	assert.Equal(t, "value", err.Context["new"])
}

func TestExtractComponent(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		expected string
	}{
		{
			name:     "services component",
			file:     "/path/to/cc-dailyuse-bar/src/services/config_service.go",
			expected: "services",
		},
		{
			name:     "models component",
			file:     "/path/to/cc-dailyuse-bar/src/models/config.go",
			expected: "models",
		},
		{
			name:     "lib component",
			file:     "/path/to/cc-dailyuse-bar/src/lib/errors.go",
			expected: "lib",
		},
		{
			name:     "no src directory",
			file:     "/path/to/some/other/file.go",
			expected: "unknown",
		},
		{
			name:     "src at end",
			file:     "/path/to/src",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, extractComponent(tt.file))
		})
	}
}

func TestConvenienceErrorFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) *AppError
		code     string
	}{
		{"ConfigError", ConfigError, ErrCodeConfig},
		{"UsageError", UsageError, ErrCodeUsage},
		{"UIError", UIError, ErrCodeUI},
		{"CCUsageError", CCUsageError, ErrCodeCCUsage},
		{"ValidationError", ValidationError, ErrCodeValidation},
		{"SystemError", SystemError, ErrCodeSystem},
		{"TemplateError", TemplateError, ErrCodeTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.function("test message")
			assert.Equal(t, tt.code, err.Code)
			assert.Equal(t, "test message", err.Message)
		})
	}
}

func TestIsErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		code     string
		expected bool
	}{
		{
			name:     "AppError with matching code",
			err:      &AppError{Code: "TEST_ERROR"},
			code:     "TEST_ERROR",
			expected: true,
		},
		{
			name:     "AppError with different code",
			err:      &AppError{Code: "OTHER_ERROR"},
			code:     "TEST_ERROR",
			expected: false,
		},
		{
			name:     "non-AppError",
			err:      errors.New("regular error"),
			code:     "TEST_ERROR",
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			code:     "TEST_ERROR",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsErrorCode(tt.err, tt.code))
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "AppError",
			err:      &AppError{Code: "TEST_ERROR"},
			expected: "TEST_ERROR",
		},
		{
			name:     "non-AppError",
			err:      errors.New("regular error"),
			expected: "",
		},
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetErrorCode(tt.err))
		})
	}
}

func TestErrorConstants(t *testing.T) {
	expectedCodes := []string{
		ErrCodeConfig,
		ErrCodeUsage,
		ErrCodeUI,
		ErrCodeCCUsage,
		ErrCodeValidation,
		ErrCodeSystem,
		ErrCodeTemplate,
	}

	// Ensure all error codes are non-empty and unique
	seen := make(map[string]bool)
	for _, code := range expectedCodes {
		assert.NotEmpty(t, code, "Error code should not be empty")
		assert.False(t, seen[code], "Error code %s should be unique", code)
		seen[code] = true
	}

	assert.Len(t, seen, len(expectedCodes), "Should have expected number of error codes")
}
