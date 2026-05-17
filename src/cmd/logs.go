package cmd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"cc-dailyuse-bar/src/lib"
)

var (
	logsTailLines int
	logsFollow    bool
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Print the latest entries from the application log",
	Long: `Print the tail of the cc-dailyuse-bar log file.

Useful when the menu bar shows "Unknown" or another unexpected state — the
structured logs explain why ccusage failed (path lookup, JSON parse, timeout,
etc.). On macOS the log lives at ~/Library/Logs/cc-dailyuse-bar/stderr.log
and is written both by the LaunchAgent (when installed via 'service install')
and by the .app bundle itself (since it now redirects its own stderr).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path := appLogPath()
		if path == "" {
			return lib.NewError(lib.ErrCodeSystem, "logs: application log file is only available on macOS")
		}

		out := cmd.OutOrStdout()

		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(out, "No log file at %s yet.\n", path)
			fmt.Fprintln(out, "The log is created the first time the app runs (either via the menu bar or 'cc-dailyuse-bar service install').")
			return nil
		} else if err != nil {
			return lib.WrapError(err, lib.ErrCodeSystem, fmt.Sprintf("logs: stat %q", path))
		}

		if err := printTail(out, path, logsTailLines); err != nil {
			return err
		}

		if !logsFollow {
			return nil
		}
		return followFile(cmd.Context(), out, path)
	},
}

func init() {
	logsCmd.Flags().IntVarP(&logsTailLines, "tail", "n", 50, "number of lines from the end of the log to print (0 = all)")
	logsCmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "keep reading new lines as they're appended (like tail -f)")
	RootCmd.AddCommand(logsCmd)
}

// printTail writes the last n lines of path to w. n <= 0 means "everything".
// Reads the whole file; fine for the log sizes this app produces.
func printTail(w io.Writer, path string, n int) error {
	// #nosec G304 -- path comes from appLogPath() (UserHomeDir + fixed app subpath), not user input
	data, err := os.ReadFile(path)
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, fmt.Sprintf("logs: read %q", path))
	}
	if len(data) == 0 {
		return nil
	}

	trimmed := bytes.TrimRight(data, "\n")
	if len(trimmed) == 0 {
		return nil
	}
	lines := bytes.Split(trimmed, []byte("\n"))

	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}

	for _, line := range lines {
		if _, err := fmt.Fprintf(w, "%s\n", line); err != nil {
			return err
		}
	}
	return nil
}

// followFile opens path at its end and prints new lines as they appear. Exits
// when ctx is cancelled (e.g. SIGINT via cobra's context). Doesn't handle
// rotation — the log file is append-only in this app.
func followFile(ctx context.Context, w io.Writer, path string) error {
	// #nosec G304 -- path comes from appLogPath() (UserHomeDir + fixed app subpath), not user input
	f, err := os.Open(path)
	if err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, fmt.Sprintf("logs: open %q for follow", path))
	}
	defer func() { _ = f.Close() }()

	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return lib.WrapError(err, lib.ErrCodeSystem, fmt.Sprintf("logs: seek to end of %q", path))
	}

	reader := bufio.NewReader(f)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		line, err := reader.ReadString('\n')
		if line != "" {
			if _, werr := fmt.Fprint(w, line); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(250 * time.Millisecond):
			}
			continue
		}
		if err != nil {
			return lib.WrapError(err, lib.ErrCodeSystem, fmt.Sprintf("logs: read %q", path))
		}
	}
}
