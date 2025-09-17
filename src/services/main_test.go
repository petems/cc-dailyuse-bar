package services

import (
	"os"
	"testing"

	"cc-dailyuse-bar/src/internal/testhelpers"
)

func TestMain(m *testing.M) {
	os.Exit(testhelpers.RunSilenced(m))
}
