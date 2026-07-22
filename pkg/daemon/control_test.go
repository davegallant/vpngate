package daemon

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestControlStatus(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	want := Snapshot{State: "CONNECTED", HostName: "public-vpn-1", IPAddr: "1.2.3.4", CountryLong: "Japan", PID: 42}
	server := NewControlServer(ln, func() (Snapshot, error) { return want, nil }, nil)
	go server.Serve()

	got, err := SendStatus(ln.Addr().String(), time.Second)
	assert.NoError(t, err)
	assert.Equal(t, want.State, got.State)
	assert.Equal(t, want.HostName, got.HostName)
	assert.Equal(t, want.PID, got.PID)
}

func TestControlStatusError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	server := NewControlServer(ln, func() (Snapshot, error) { return Snapshot{}, errors.New("boom") }, nil)
	go server.Serve()

	_, err = SendStatus(ln.Addr().String(), time.Second)
	assert.Error(t, err)
}

func TestControlStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer func() { _ = ln.Close() }()

	stopped := make(chan struct{})
	server := NewControlServer(ln, nil, func() { close(stopped) })
	go server.Serve()

	assert.NoError(t, SendStop(ln.Addr().String(), time.Second))

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("onStop was not called")
	}
}

func TestSendStatusUnreachable(t *testing.T) {
	_, err := SendStatus("127.0.0.1:1", 100*time.Millisecond)
	assert.Error(t, err)
}
