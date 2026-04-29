package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const rootNoArgsHelperEnv = "CC_DAILYUSE_BAR_TEST_ROOT_NOARGS_HELPER"

func TestRootExecute_NoArgsDoesNotStackOverflow(t *testing.T) {
	if os.Getenv(rootNoArgsHelperEnv) == "1" {
		// Avoid launching the tray app in tests; this should return an error quickly.
		runTrayApp = nil
		RootCmd.SetArgs(nil)
		if err := RootCmd.Execute(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRootExecute_NoArgsDoesNotStackOverflow$") //nolint:gosec // G702: re-execing the test binary itself is the standard Go subprocess pattern
	cmd.Env = append(os.Environ(), rootNoArgsHelperEnv+"=1")

	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		t.Fatalf("subprocess timed out; output: %s", string(out))
	}

	require.Error(t, err)
	output := string(out)
	assert.NotContains(t, output, "stack overflow")
	assert.NotContains(t, output, "goroutine stack exceeds")
	assert.Contains(t, output, "built without GUI support")
}
