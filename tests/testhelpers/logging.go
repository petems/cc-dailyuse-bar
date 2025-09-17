package testhelpers

import (
	"io"
	"testing"

	"cc-dailyuse-bar/src/lib"
)

// RunSilenced executes the provided test suite with logging redirected to io.Discard.
// It restores the previous global logger output before returning.
func RunSilenced(m *testing.M) int {
	original := lib.GetGlobalOutput()
	lib.SetGlobalOutput(io.Discard)
	code := m.Run()
	lib.SetGlobalOutput(original)
	return code
}
