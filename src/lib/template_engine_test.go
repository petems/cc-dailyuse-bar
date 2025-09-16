package lib

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTemplateEngine(t *testing.T) {
	engine := NewTemplateEngine()

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.logger)
	assert.Equal(t, "template", engine.logger.component)
}

func TestTemplateEngine_Execute(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name        string
		template    string
		data        interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "simple template",
			template:    "Hello {{.Name}}",
			data:        map[string]string{"Name": "World"},
			expected:    "Hello World",
			expectError: false,
		},
		{
			name:        "template with multiple fields",
			template:    "{{.Count}}: ${{.Cost}} ({{.Status}})",
			data:        map[string]interface{}{"Count": 42, "Cost": 15.75, "Status": "High"},
			expected:    "42: $15.75 (High)",
			expectError: false,
		},
		{
			name:        "empty template",
			template:    "",
			data:        map[string]string{"Name": "World"},
			expected:    "",
			expectError: true,
		},
		{
			name:        "invalid template syntax",
			template:    "Hello {{.Name",
			data:        map[string]string{"Name": "World"},
			expected:    "",
			expectError: true,
		},
		{
			name:        "template with missing field",
			template:    "Hello {{.Name}} {{.Missing}}",
			data:        map[string]string{"Name": "World"},
			expected:    "Hello World <no value>",
			expectError: false,
		},
		{
			name:        "template with no variables",
			template:    "Static text",
			data:        map[string]string{"Name": "World"},
			expected:    "Static text",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Execute(tt.template, tt.data)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestTemplateEngine_Validate(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name        string
		template    string
		expectError bool
	}{
		{
			name:        "valid template",
			template:    "Hello {{.Name}}",
			expectError: false,
		},
		{
			name:        "valid complex template",
			template:    "{{.Count}}: ${{.Cost}} ({{.Status}})",
			expectError: false,
		},
		{
			name:        "empty template",
			template:    "",
			expectError: true,
		},
		{
			name:        "invalid template syntax",
			template:    "Hello {{.Name",
			expectError: true,
		},
		{
			name:        "template with no variables",
			template:    "Static text",
			expectError: false,
		},
		{
			name:        "template with invalid action",
			template:    "Hello {{invalid}}",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.Validate(tt.template)

			if tt.expectError {
				assert.Error(t, err)
				assert.True(t, IsErrorCode(err, ErrCodeTemplate))
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTemplateEngine_ExecuteWithDefault(t *testing.T) {
	engine := NewTemplateEngine()

	tests := []struct {
		name         string
		template     string
		data         interface{}
		defaultValue string
		expected     string
	}{
		{
			name:         "valid template",
			template:     "Hello {{.Name}}",
			data:         map[string]string{"Name": "World"},
			defaultValue: "Default",
			expected:     "Hello World",
		},
		{
			name:         "invalid template",
			template:     "Hello {{.Name",
			data:         map[string]string{"Name": "World"},
			defaultValue: "Default",
			expected:     "Default",
		},
		{
			name:         "empty template",
			template:     "",
			data:         map[string]string{"Name": "World"},
			defaultValue: "Default",
			expected:     "Default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := engine.ExecuteWithDefault(tt.template, tt.data, tt.defaultValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGlobalTemplateFunctions(t *testing.T) {
	tests := []struct {
		name        string
		template    string
		data        interface{}
		expected    string
		expectError bool
	}{
		{
			name:        "ExecuteTemplate - valid",
			template:    "Hello {{.Name}}",
			data:        map[string]string{"Name": "World"},
			expected:    "Hello World",
			expectError: false,
		},
		{
			name:        "ExecuteTemplate - invalid",
			template:    "Hello {{.Name",
			data:        map[string]string{"Name": "World"},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test ExecuteTemplate
			result, err := ExecuteTemplate(tt.template, tt.data)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}

			// Test ValidateTemplate
			err = ValidateTemplate(tt.template)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Test ExecuteTemplateWithDefault
			defaultResult := ExecuteTemplateWithDefault(tt.template, tt.data, "Default")
			if tt.expectError {
				assert.Equal(t, "Default", defaultResult)
			} else {
				assert.Equal(t, tt.expected, defaultResult)
			}
		})
	}
}

func TestTemplateEngine_ComplexData(t *testing.T) {
	engine := NewTemplateEngine()

	// Test with struct data
	type UsageData struct {
		Count  int
		Cost   float64
		Status string
	}

	data := UsageData{
		Count:  42,
		Cost:   15.75,
		Status: "High",
	}

	template := "Usage: {{.Count}} calls, ${{.Cost}}, Status: {{.Status}}"
	expected := "Usage: 42 calls, $15.75, Status: High"

	result, err := engine.Execute(template, data)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTemplateEngine_NestedData(t *testing.T) {
	engine := NewTemplateEngine()

	// Test with nested data
	data := map[string]interface{}{
		"User": map[string]interface{}{
			"Name": "John",
			"Age":  30,
		},
		"Count": 42,
	}

	template := "User {{.User.Name}} (age {{.User.Age}}) has {{.Count}} items"
	expected := "User John (age 30) has 42 items"

	result, err := engine.Execute(template, data)
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTemplateEngine_ErrorHandling(t *testing.T) {
	engine := NewTemplateEngine()

	// Test that errors are properly wrapped
	_, err := engine.Execute("{{.Name", map[string]string{"Name": "World"})
	require.Error(t, err)

	// Verify it's a template error
	assert.True(t, IsErrorCode(err, ErrCodeTemplate))
	assert.Contains(t, err.Error(), "failed to parse template")
}

func TestTemplateEngine_Logging(t *testing.T) {
	// This test verifies that logging occurs during template operations
	// We can't easily test the actual log output without complex setup,
	// but we can verify that the operations complete successfully
	engine := NewTemplateEngine()

	// Valid template should not cause errors
	err := engine.Validate("Hello {{.Name}}")
	assert.NoError(t, err)

	result, err := engine.Execute("Hello {{.Name}}", map[string]string{"Name": "World"})
	assert.NoError(t, err)
	assert.Equal(t, "Hello World", result)

	// Invalid template should cause errors but not panic
	err = engine.Validate("Hello {{.Name")
	assert.Error(t, err)

	result, err = engine.Execute("Hello {{.Name", map[string]string{"Name": "World"})
	assert.Error(t, err)
	assert.Empty(t, result)
}
