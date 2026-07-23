package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/davegallant/vpngate/pkg/daemon"
)

var (
	flagLogsFollow bool
	flagLogsLines  int
)

func init() {
	logsCmd.Flags().BoolVarP(&flagLogsFollow, "follow", "f", false, "follow the log as it's written")
	logsCmd.Flags().IntVarP(&flagLogsLines, "lines", "n", 0, "show only the last N lines (0 shows the whole file)")
	rootCmd.AddCommand(logsCmd)
}

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "View the log for a background vpn connection started with 'connect -d'",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLogs(cmd.OutOrStdout(), daemon.LogPath(), flagLogsLines, flagLogsFollow)
	},
}

func runLogs(w io.Writer, path string, lines int, follow bool) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintln(w, "No daemon log yet; start one with 'connect -d'.")
			return nil
		}
		// daemon.log is root-owned (openvpn itself requires root), so a
		// non-root invocation hits a permission error here rather than
		// "not exist" — report it plainly instead of a raw "permission
		// denied" error, matching status/disconnect.
		if os.IsPermission(err) {
			fmt.Fprintln(w, "Insufficient permissions to read the daemon log (try with sudo).")
			return nil
		}
		return err
	}

	if lines > 0 {
		data = []byte(lastLines(data, lines))
	}
	if len(data) > 0 {
		if _, err := w.Write(data); err != nil {
			return err
		}
		if !strings.HasSuffix(string(data), "\n") {
			fmt.Fprintln(w)
		}
	}

	if !follow {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return err
	}
	return followFile(w, f)
}

// followFile polls f for new content and writes it to w as it arrives,
// like `tail -f`. It blocks until the read loop hits an error other than
// EOF (e.g. the caller is interrupted).
func followFile(w io.Writer, f *os.File) error {
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadString('\n')
		if len(line) > 0 {
			if _, werr := io.WriteString(w, line); werr != nil {
				return werr
			}
		}
		if err != nil {
			if err != io.EOF {
				return err
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// lastLines returns the last n lines of data, joined with trailing
// newlines preserved.
func lastLines(data []byte, n int) string {
	trimmed := strings.TrimRight(string(data), "\n")
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n") + "\n"
}
