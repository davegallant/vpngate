package daemon

import (
	"bufio"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// startFakeManagementServer starts a loopback TCP server that mimics
// OpenVPN's management interface: it sends the greeting line on connect,
// then calls onCommand for each line the client sends.
func startFakeManagementServer(t *testing.T, onCommand func(cmd string, conn net.Conn)) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	t.Cleanup(func() { _ = ln.Close() })

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		_, _ = conn.Write([]byte(">INFO:OpenVPN Management Interface Version 5 -- type 'help' for more info\n"))

		reader := bufio.NewReader(conn)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			onCommand(strings.TrimRight(line, "\r\n"), conn)
		}
	}()

	return ln.Addr().String()
}

func TestManagementState(t *testing.T) {
	addr := startFakeManagementServer(t, func(cmd string, conn net.Conn) {
		if cmd == "state" {
			_, _ = conn.Write([]byte("1690000000,CONNECTED,SUCCESS,10.9.0.2,1.2.3.4,1194,,\r\nEND\r\n"))
		}
	})

	m, err := DialManagement(addr, time.Second)
	assert.NoError(t, err)
	defer m.Close()

	state, err := m.State()
	assert.NoError(t, err)
	assert.Equal(t, "CONNECTED", state)
}

func TestManagementDisconnect(t *testing.T) {
	received := make(chan string, 1)
	addr := startFakeManagementServer(t, func(cmd string, conn net.Conn) {
		received <- cmd
	})

	m, err := DialManagement(addr, time.Second)
	assert.NoError(t, err)
	defer m.Close()

	assert.NoError(t, m.Disconnect())

	select {
	case cmd := <-received:
		assert.Equal(t, "signal SIGTERM", cmd)
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for disconnect command")
	}
}

func TestParseStateNoStateLine(t *testing.T) {
	_, err := parseState([]string{"", "garbage-with-no-comma"})
	assert.Error(t, err)
}

func TestDialManagementConnectionRefused(t *testing.T) {
	_, err := DialManagement("127.0.0.1:1", 100*time.Millisecond)
	assert.Error(t, err)
}
