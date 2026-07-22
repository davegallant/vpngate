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

// defaultBaseDir is the fixed, machine-wide parent of Dir() on unix. It
// deliberately does not use os.TempDir()/$TMPDIR: on macOS, $TMPDIR is a
// per-user path assigned by launchd, and sudo does not preserve it by
// default, so a root supervisor (sudo connect -d) and a non-root
// status/disconnect invocation would otherwise resolve to two different
// directories for the same daemon. /tmp is a fixed absolute path shared
// by every user, root included.
func defaultBaseDir() string {
	return "/tmp"
}
