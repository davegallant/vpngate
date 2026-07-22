//go:build windows

package daemon

import (
	"os"
	"syscall"

	"golang.org/x/sys/windows"
)

const (
	detachedProcess = 0x00000008
	// stillActive is the well-known Win32 GetExitCodeProcess() value for
	// a process that hasn't exited yet. Not exported by
	// golang.org/x/sys/windows, so it's defined here directly.
	stillActive = 259
)

// IsAlive reports whether pid identifies a running process.
func IsAlive(pid int) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	defer func() { _ = windows.CloseHandle(h) }()

	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return false
	}
	return code == uint32(stillActive)
}

// DetachAttr returns the SysProcAttr that starts a child process detached
// from the parent's console and process group, so it survives the parent
// exiting.
func DetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
}

// defaultBaseDir is the fixed, machine-wide parent of Dir() on Windows:
// %ProgramData% (falling back to os.TempDir() in the rare case it's
// unset) rather than %TEMP%, which is per-user and — for consistency
// with the unix side of this same fix — shouldn't be relied on to agree
// between an elevated "Run as Administrator" connect -d and a normal
// status/disconnect invocation.
func defaultBaseDir() string {
	if v := os.Getenv("ProgramData"); v != "" {
		return v
	}
	return os.TempDir()
}
