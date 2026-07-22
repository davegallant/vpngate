//go:build !windows

package daemon

import (
	"os"
	"syscall"
)

// IsAlive reports whether pid identifies a running process.
func IsAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return process.Signal(syscall.Signal(0)) == nil
}

// DetachAttr returns the SysProcAttr that starts a child process in its
// own session, detached from the parent's controlling terminal and
// process group, so it survives the parent exiting.
func DetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
