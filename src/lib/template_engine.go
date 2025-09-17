// Package lib provides shared utilities like error handling and logging.
package lib

import (
	"bytes"
	"text/template"
)

// TemplateEngine provides template execution with validation and error handling.
type TemplateEngine struct {
	logger *Logger
}

// NewTemplateEngine creates a new template engine.
func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		logger: NewLogger("template"),
	}
}

// Execute executes a template string with the provided data.
func (te *TemplateEngine) Execute(templateStr string, data interface{}) (string, error) {
	if templateStr == "" {
		return "", TemplateError("template string cannot be empty")
	}

	// Parse the template
	tmpl, err := template.New("display").Parse(templateStr)
	if err != nil {
		te.logger.Error("Template parsing failed", map[string]interface{}{
			"template": templateStr,
			"error":    err.Error(),
		})
		return "", WrapError(err, ErrCodeTemplate, "failed to parse template")
	}

	// Execute the template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		te.logger.Error("Template execution failed", map[string]interface{}{
			"template": templateStr,
			"data":     data,
			"error":    err.Error(),
		})
		return "", WrapError(err, ErrCodeTemplate, "failed to execute template")
	}

	result := buf.String()

	te.logger.Debug("Template executed successfully", map[string]interface{}{
		"template": templateStr,
		"result":   result,
	})

	return result, nil
}

// Validate validates a template string without executing it.
func (te *TemplateEngine) Validate(templateStr string) error {
	if templateStr == "" {
		return TemplateError("template string cannot be empty")
	}

	_, err := template.New("validation").Parse(templateStr)
	if err != nil {
		te.logger.Warn("Template validation failed", map[string]interface{}{
			"template": templateStr,
			"error":    err.Error(),
		})
		return WrapError(err, ErrCodeTemplate, "template validation failed")
	}

	te.logger.Debug("Template validated successfully", map[string]interface{}{
		"template": templateStr,
	})

	return nil
}

// ExecuteWithDefault executes a template and returns a default value on error.
func (te *TemplateEngine) ExecuteWithDefault(templateStr string, data interface{}, defaultValue string) string {
	result, err := te.Execute(templateStr, data)
	if err != nil {
		te.logger.Warn("Template execution failed, using default", map[string]interface{}{
			"template": templateStr,
			"default":  defaultValue,
			"error":    err.Error(),
		})
		return defaultValue
	}
	return result
}

// Global template engine instance.
var globalTemplateEngine = NewTemplateEngine()

// ExecuteTemplate executes a template using the global engine.
func ExecuteTemplate(templateStr string, data interface{}) (string, error) {
	return globalTemplateEngine.Execute(templateStr, data)
}

// ValidateTemplate validates a template using the global engine.
func ValidateTemplate(templateStr string) error {
	return globalTemplateEngine.Validate(templateStr)
}

// ExecuteTemplateWithDefault executes a template with default using the global engine.
func ExecuteTemplateWithDefault(templateStr string, data interface{}, defaultValue string) string {
	return globalTemplateEngine.ExecuteWithDefault(templateStr, data, defaultValue)
}
