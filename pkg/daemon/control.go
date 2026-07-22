package daemon

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"
)

// Snapshot is the supervisor's answer to a STATUS control request.
type Snapshot struct {
	State       string    `json:"state"`
	HostName    string    `json:"hostname"`
	IPAddr      string    `json:"ip_addr"`
	CountryLong string    `json:"country"`
	StartedAt   time.Time `json:"started_at"`
	PID         int       `json:"pid"`
}

// ControlServer is the supervisor side of the control protocol: a
// loopback TCP listener that answers STATUS and STOP requests from
// separate `vpngate status`/`vpngate disconnect` invocations. Neither of
// those commands touches OpenVPN's management socket directly — only the
// supervisor can tell "OpenVPN exited, reconnect" apart from "the user
// asked to disconnect".
type ControlServer struct {
	listener net.Listener
	onStatus func() (Snapshot, error)
	onStop   func()
}

// NewControlServer wraps an already-open listener. onStatus answers
// STATUS requests; onStop (may be nil) is invoked in its own goroutine,
// after the client has already been acknowledged, to tear the daemon
// down.
func NewControlServer(listener net.Listener, onStatus func() (Snapshot, error), onStop func()) *ControlServer {
	return &ControlServer{listener: listener, onStatus: onStatus, onStop: onStop}
}

// Addr returns the address clients should dial.
func (c *ControlServer) Addr() string {
	return c.listener.Addr().String()
}

// Serve accepts connections until the listener is closed.
func (c *ControlServer) Serve() {
	for {
		conn, err := c.listener.Accept()
		if err != nil {
			return
		}
		c.handle(conn)
	}
}

func (c *ControlServer) handle(conn net.Conn) {
	defer func() { _ = conn.Close() }()

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return
	}

	switch strings.TrimSpace(line) {
	case "STATUS":
		if c.onStatus == nil {
			_, _ = fmt.Fprintln(conn, "ERROR status unavailable")
			return
		}
		snap, err := c.onStatus()
		if err != nil {
			_, _ = fmt.Fprintf(conn, "ERROR %s\n", err)
			return
		}
		data, err := json.Marshal(snap)
		if err != nil {
			_, _ = fmt.Fprintf(conn, "ERROR %s\n", err)
			return
		}
		_, _ = conn.Write(append(data, '\n'))
	case "STOP":
		_, _ = conn.Write([]byte("OK\n"))
		if c.onStop != nil {
			go c.onStop()
		}
	default:
		_, _ = fmt.Fprintln(conn, "ERROR unknown command")
	}
}

// SendStatus dials addr and returns the daemon's current snapshot.
func SendStatus(addr string, timeout time.Duration) (Snapshot, error) {
	var snap Snapshot

	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return snap, err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write([]byte("STATUS\n")); err != nil {
		return snap, err
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return snap, err
	}
	line = strings.TrimSpace(line)
	if strings.HasPrefix(line, "ERROR") {
		return snap, fmt.Errorf("daemon: %s", line)
	}

	if err := json.Unmarshal([]byte(line), &snap); err != nil {
		return snap, fmt.Errorf("parsing status response: %w", err)
	}
	return snap, nil
}

// SendStop dials addr and asks the daemon to disconnect and exit. It
// returns once the daemon has acknowledged the request, not once
// teardown is complete.
func SendStop(addr string, timeout time.Duration) error {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return err
	}
	defer func() { _ = conn.Close() }()
	_ = conn.SetDeadline(time.Now().Add(timeout))

	if _, err := conn.Write([]byte("STOP\n")); err != nil {
		return err
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return err
	}
	line = strings.TrimSpace(line)
	if line != "OK" {
		return fmt.Errorf("daemon: unexpected response %q", line)
	}
	return nil
}
