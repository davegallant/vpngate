package cmd

import (
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/davegallant/vpngate/pkg/daemon"
)

func TestDisconnectNotConnected(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	out := captureStdout(t, func() {
		assert.NoError(t, disconnectCmd.RunE(disconnectCmd, nil))
	})
	assert.Contains(t, out, "Not connected.")
}

func TestDisconnectStalePID(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	assert.NoError(t, daemon.Save(daemon.State{PID: 999999, HostName: "public-vpn-1"}))

	out := captureStdout(t, func() {
		assert.NoError(t, disconnectCmd.RunE(disconnectCmd, nil))
	})
	assert.Contains(t, out, "Not connected.")
}

// TestDisconnectPermissionDenied mirrors TestStatusPermissionDenied: a
// non-root `disconnect` against root-owned state gets EACCES, not
// ErrNotExist. Skips under root, where chmod 0o000 doesn't deny access.
func TestDisconnectPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks don't apply when running as root")
	}

	t.Setenv(daemon.DirEnvVar, t.TempDir())

	assert.NoError(t, daemon.Save(daemon.State{PID: os.Getpid(), HostName: "public-vpn-1"}))
	assert.NoError(t, os.Chmod(daemon.StatePath(), 0o000))

	out := captureStdout(t, func() {
		assert.NoError(t, disconnectCmd.RunE(disconnectCmd, nil))
	})
	assert.Contains(t, out, "insufficient permissions")
}

func TestDisconnectSendsStop(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	stopped := make(chan struct{})
	server := daemon.NewControlServer(ln, nil, func() { close(stopped) })
	go server.Serve()

	assert.NoError(t, daemon.Save(daemon.State{
		PID:         os.Getpid(),
		ControlAddr: ln.Addr().String(),
		HostName:    "public-vpn-1",
	}))

	out := captureStdout(t, func() {
		assert.NoError(t, disconnectCmd.RunE(disconnectCmd, nil))
	})
	assert.Contains(t, out, "Disconnected.")

	// onStop runs in its own goroutine on the server side (see
	// ControlServer.handle), so it may not have fired the instant
	// SendStop's "OK\n" reply is read — wait for it instead of checking
	// synchronously.
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("expected STOP to reach the control server")
	}
}
