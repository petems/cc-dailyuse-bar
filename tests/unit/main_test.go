package unit

import (
	"io"
	"os"
	"testing"

	"cc-dailyuse-bar/src/lib"
)

func TestMain(m *testing.M) {
	lib.SetGlobalOutput(io.Discard)
	code := m.Run()
	lib.SetGlobalOutput(os.Stderr)
	os.Exit(code)
}
