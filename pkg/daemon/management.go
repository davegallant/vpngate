package daemon

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// Management is a client for OpenVPN's plaintext management protocol. It
// is used only by the supervisor process, which is the sole client of a
// given OpenVPN instance's management socket — status/disconnect never
// dial it directly (see docs/superpowers/specs/2026-07-22-daemon-mode-design.md).
type Management struct {
	conn net.Conn
	r    *bufio.Reader
}

// DialManagement connects to addr and discards OpenVPN's greeting line.
func DialManagement(addr string, timeout time.Duration) (*Management, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	m := &Management{conn: conn, r: bufio.NewReader(conn)}
	if _, err := m.r.ReadString('\n'); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("reading management greeting: %w", err)
	}
	return m, nil
}

// Close closes the underlying connection.
func (m *Management) Close() error {
	return m.conn.Close()
}

// State queries OpenVPN's current connection state (e.g. "CONNECTED",
// "RECONNECTING", "EXITING").
func (m *Management) State() (string, error) {
	if _, err := m.conn.Write([]byte("state\n")); err != nil {
		return "", err
	}

	var lines []string
	for {
		line, err := m.r.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "END" {
			break
		}
		lines = append(lines, line)
	}
	return parseState(lines)
}

// parseState extracts the state field from OpenVPN's "state" command
// response. Each response line is comma-separated; the second field
// (index 1) is the connection state, per OpenVPN's management-notes.txt.
func parseState(lines []string) (string, error) {
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) >= 2 && fields[1] != "" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("no state found in management response")
}

// Disconnect asks OpenVPN to shut down cleanly.
func (m *Management) Disconnect() error {
	_, err := m.conn.Write([]byte("signal SIGTERM\n"))
	return err
}
