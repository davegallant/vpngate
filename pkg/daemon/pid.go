package daemon

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PidPath returns the path of the file holding the supervisor's PID.
// The supervisor writes it at startup, before its first connection
// attempt, and removes it on exit. It lets a foreground `connect -d`
// tell "daemon died during startup" apart from "daemon still
// negotiating", and guards against starting a second daemon while the
// first hasn't written its state file yet (state is only written once
// the tunnel is up).
func PidPath() string { return filepath.Join(Dir(), "daemon.pid") }

// WritePid records pid in PidPath(), creating Dir() if needed.
func WritePid(pid int) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	return os.WriteFile(PidPath(), []byte(strconv.Itoa(pid)), 0o600)
}

// ReadPid returns the PID recorded by WritePid. Check os.IsNotExist(err)
// for "no daemon starting".
func ReadPid() (int, error) {
	data, err := os.ReadFile(PidPath())
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}

// RemovePid deletes PidPath(). It is not an error if the file is
// already gone.
func RemovePid() error {
	err := os.Remove(PidPath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
