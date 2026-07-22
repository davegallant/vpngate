package cmd

import (
	"io"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/davegallant/vpngate/pkg/daemon"
)

// captureStdout redirects os.Stdout for the duration of fn and returns
// what was written to it.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	assert.NoError(t, err)
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	assert.NoError(t, w.Close())
	out, err := io.ReadAll(r)
	assert.NoError(t, err)
	return string(out)
}

func TestStatusNotConnected(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	out := captureStdout(t, func() {
		assert.NoError(t, statusCmd.RunE(statusCmd, nil))
	})
	assert.Contains(t, out, "Not connected.")
}

func TestStatusStalePID(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	assert.NoError(t, daemon.Save(daemon.State{PID: 999999, HostName: "public-vpn-1"}))

	out := captureStdout(t, func() {
		assert.NoError(t, statusCmd.RunE(statusCmd, nil))
	})
	assert.Contains(t, out, "Not connected.")

	_, err := daemon.Load()
	assert.True(t, os.IsNotExist(err))
}

func TestStatusConnected(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	snap := daemon.Snapshot{State: "CONNECTED", HostName: "public-vpn-1", IPAddr: "1.2.3.4", CountryLong: "Japan", StartedAt: time.Now().Add(-time.Minute), PID: os.Getpid()}
	server := daemon.NewControlServer(ln, func() (daemon.Snapshot, error) { return snap, nil }, nil)
	go server.Serve()

	assert.NoError(t, daemon.Save(daemon.State{
		PID:         os.Getpid(),
		ControlAddr: ln.Addr().String(),
		HostName:    "public-vpn-1",
		IPAddr:      "1.2.3.4",
		CountryLong: "Japan",
	}))

	out := captureStdout(t, func() {
		assert.NoError(t, statusCmd.RunE(statusCmd, nil))
	})
	assert.Contains(t, out, "CONNECTED")
	assert.Contains(t, out, "public-vpn-1")
	assert.Contains(t, out, "Uptime:")
}
