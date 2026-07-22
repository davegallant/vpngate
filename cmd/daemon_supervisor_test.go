package cmd

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReserveLoopbackAddr(t *testing.T) {
	addr, err := reserveLoopbackAddr()
	assert.NoError(t, err)
	assert.NotEmpty(t, addr)

	// The port must be free immediately afterward.
	ln, err := net.Listen("tcp", addr)
	assert.NoError(t, err)
	defer ln.Close()
}

func TestWaitForManagementSucceeds(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer ln.Close()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = conn.Write([]byte(">INFO:OpenVPN Management Interface Version 5\n"))
		// Keep the connection open for State()/Disconnect() calls the
		// caller might make; just block until the test ends.
		buf := make([]byte, 1)
		_, _ = conn.Read(buf)
	}()

	mgmt, err := waitForManagement(ln.Addr().String(), time.Second)
	assert.NoError(t, err)
	assert.NotNil(t, mgmt)
	defer mgmt.Close()
}

func TestWaitForManagementTimesOut(t *testing.T) {
	_, err := waitForManagement("127.0.0.1:1", 300*time.Millisecond)
	assert.Error(t, err)
}
