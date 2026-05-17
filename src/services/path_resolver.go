package services

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"cc-dailyuse-bar/src/lib"
)

// homebrewFallbackDirs lists Homebrew install locations searched on macOS when
// a bare ccusage name can't be found via PATH. macOS LaunchServices launches
// .app bundles with a minimal PATH (/usr/bin:/bin:/usr/sbin:/sbin) that omits
// these locations, so a config like `ccusage_path: ccusage` works from a
// terminal but fails for the menu bar app. Var (not const) so tests can swap.
var homebrewFallbackDirs = []string{
	"/opt/homebrew/bin",
	"/usr/local/bin",
}

// ResolveCCUsagePath returns an absolute path that exec.Command can invoke
// for the configured ccusage location. It mirrors exec.CommandContext's PATH
// lookup, then on macOS falls back to common Homebrew dirs for bare names.
// The fallback flag is true when LookPath failed and a Homebrew dir matched.
func ResolveCCUsagePath(configured string) (resolved string, fallback bool, err error) {
	if configured == "" {
		return "", false, lib.ValidationError("ccusage path is empty")
	}

	if p, lookErr := exec.LookPath(configured); lookErr == nil {
		return p, false, nil
	} else if runtime.GOOS == "darwin" && !strings.ContainsRune(configured, '/') {
		for _, dir := range homebrewFallbackDirs {
			candidate := filepath.Join(dir, configured)
			if isExecutableFile(candidate) {
				return candidate, true, nil
			}
		}
		return "", false, lib.WrapError(lookErr, lib.ErrCodeCCUsage, "ccusage not found on PATH or Homebrew fallback dirs")
	} else {
		return "", false, lib.WrapError(lookErr, lib.ErrCodeCCUsage, "ccusage not found on PATH")
	}
}

func isExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode()&0o111 != 0
}
