package services

import (
	"os"
	"testing"

	"cc-dailyuse-bar/tests/testhelpers"
)

func TestMain(m *testing.M) {
	os.Exit(testhelpers.RunSilenced(m))
}
