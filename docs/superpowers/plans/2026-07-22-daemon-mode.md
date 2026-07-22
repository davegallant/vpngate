# Daemon Mode Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement `vpngate connect -d`, `vpngate status`, and `vpngate disconnect` per [issue #53](https://github.com/davegallant/vpngate/issues/53), per the design in `docs/superpowers/specs/2026-07-22-daemon-mode-design.md`.

**Architecture:** `connect -d` re-execs the `vpngate` binary itself, detached, as a supervisor process that owns the existing connect/reconnect loop and an openvpn child (started with `--management` enabled). The supervisor listens on its own loopback control socket; separate `status`/`disconnect` invocations talk to that socket (never to openvpn's management socket directly) so only the supervisor — which alone knows whether an exit means "reconnect" or "the user asked to stop" — decides what happens next. State (PID, control address, connected server) is persisted to `os.TempDir()/vpngate/state.json`.

**Tech Stack:** Go 1.26, cobra, testify/assert for tests, `golang.org/x/sys/windows` for the Windows process-liveness check (already an indirect dependency, promoted to direct).

## Global Constraints

- Go version: 1.26.1 (per `go.mod`) — do not use anything newer.
- Platforms: macOS, Linux, Windows (per README) — every new file that touches OS process APIs needs both a `!windows` and a `windows` build-tagged variant.
- No new third-party dependencies beyond `golang.org/x/sys` (already present as an indirect dependency; `go mod tidy` should promote it to direct once a non-test file imports it).
- State/config/log files live at `os.TempDir()/vpngate/{state.json,config.ovpn,daemon.log}`, overridable via the `VPNGATE_DAEMON_DIR` environment variable for tests — never `~/.vpngate` (see spec, "sudo $HOME ambiguity").
- `status`/`disconnect` never touch OpenVPN's management socket directly — only the supervisor's control socket (see spec, "disconnect vs. reconnect supervisor").
- Tests use `github.com/stretchr/testify/assert`, matching existing tests in `pkg/vpn/list_test.go` and `pkg/util/retry_test.go`.
- Existing `vpn.Connect` (foreground path) and its callers in `cmd/connect.go` must keep working unchanged when `-d` is not passed.

---

### Task 1: `pkg/daemon` — directories and state file

**Files:**
- Create: `pkg/daemon/dir.go`
- Create: `pkg/daemon/state.go`
- Test: `pkg/daemon/state_test.go`

**Interfaces:**
- Produces: `daemon.DirEnvVar string` (constant `"VPNGATE_DAEMON_DIR"`), `daemon.Dir() string`, `daemon.StatePath() string`, `daemon.ConfigPath() string`, `daemon.LogPath() string`, `daemon.State{PID int; ControlAddr string; HostName string; IPAddr string; CountryLong string; StartedAt time.Time}`, `daemon.Save(State) error`, `daemon.Load() (State, error)` (returns an error satisfying `os.IsNotExist` when no daemon is running), `daemon.Remove() error` (idempotent).

- [ ] **Step 1: Write the failing test**

```go
// pkg/daemon/state_test.go
package daemon

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSaveLoadRemove(t *testing.T) {
	t.Setenv(DirEnvVar, t.TempDir())

	_, err := Load()
	assert.Error(t, err)
	assert.True(t, os.IsNotExist(err))

	want := State{
		PID:         12345,
		ControlAddr: "127.0.0.1:9999",
		HostName:    "public-vpn-1",
		IPAddr:      "1.2.3.4",
		CountryLong: "Japan",
		StartedAt:   time.Now().Truncate(time.Second),
	}
	assert.NoError(t, Save(want))

	got, err := Load()
	assert.NoError(t, err)
	assert.Equal(t, want.PID, got.PID)
	assert.Equal(t, want.ControlAddr, got.ControlAddr)
	assert.Equal(t, want.HostName, got.HostName)
	assert.Equal(t, want.IPAddr, got.IPAddr)
	assert.Equal(t, want.CountryLong, got.CountryLong)
	assert.True(t, want.StartedAt.Equal(got.StartedAt))

	assert.NoError(t, Remove())
	_, err = Load()
	assert.True(t, os.IsNotExist(err))

	// Remove is idempotent.
	assert.NoError(t, Remove())
}

func TestDirUsesEnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv(DirEnvVar, tmp)
	assert.Contains(t, Dir(), tmp)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/daemon/... -run TestSaveLoadRemove -v`
Expected: FAIL — `package daemon: no Go files` (package doesn't exist yet)

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/daemon/dir.go
package daemon

import (
	"os"
	"path/filepath"
)

// DirEnvVar overrides the base directory Dir() resolves under. Tests (in
// this package and elsewhere) set it to isolate state files instead of
// touching the real os.TempDir().
const DirEnvVar = "VPNGATE_DAEMON_DIR"

// Dir returns the directory vpngate uses for daemon state, the persisted
// OpenVPN config, and the daemon log. It lives under os.TempDir() rather
// than the user's home directory because daemon mode is typically
// launched with sudo, and $HOME is unreliable under sudo (it may resolve
// to root's home or the invoking user's home depending on sudo's
// configuration) — os.TempDir() is stable regardless, and matches the
// daemon's "does not survive reboot" lifetime.
func Dir() string {
	base := os.TempDir()
	if v := os.Getenv(DirEnvVar); v != "" {
		base = v
	}
	return filepath.Join(base, "vpngate")
}

// StatePath returns the path to the daemon's state file.
func StatePath() string { return filepath.Join(Dir(), "state.json") }

// ConfigPath returns the path to the daemon's persisted OpenVPN config.
func ConfigPath() string { return filepath.Join(Dir(), "config.ovpn") }

// LogPath returns the path to the daemon's OpenVPN log file.
func LogPath() string { return filepath.Join(Dir(), "daemon.log") }
```

```go
// pkg/daemon/state.go
package daemon

import (
	"encoding/json"
	"os"
	"time"
)

// State is the on-disk record of a running daemon, written by the
// supervisor once OpenVPN reports a successful connection and read by
// `status`/`disconnect` in separate process invocations.
type State struct {
	PID         int       `json:"pid"`
	ControlAddr string    `json:"control_addr"`
	HostName    string    `json:"hostname"`
	IPAddr      string    `json:"ip_addr"`
	CountryLong string    `json:"country"`
	StartedAt   time.Time `json:"started_at"`
}

// Save writes state to StatePath(), creating Dir() if needed.
func Save(state State) error {
	if err := os.MkdirAll(Dir(), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(StatePath(), data, 0o600)
}

// Load reads and parses StatePath(). Callers should check
// os.IsNotExist(err) to distinguish "no daemon running" from a real
// error.
func Load() (State, error) {
	var state State
	data, err := os.ReadFile(StatePath())
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	return state, err
}

// Remove deletes StatePath(). It is not an error if the file is already
// gone.
func Remove() error {
	err := os.Remove(StatePath())
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/daemon/... -v`
Expected: PASS (both `TestSaveLoadRemove` and `TestDirUsesEnvOverride`)

- [ ] **Step 5: Commit**

```bash
git add pkg/daemon/dir.go pkg/daemon/state.go pkg/daemon/state_test.go
git commit -m "feat(daemon): add state file persistence"
```

---

### Task 2: `pkg/daemon` — process liveness and detach helpers (unix + windows)

**Files:**
- Create: `pkg/daemon/process_unix.go` (`//go:build !windows`)
- Create: `pkg/daemon/process_windows.go` (`//go:build windows`)
- Test: `pkg/daemon/process_unix_test.go` (`//go:build !windows`)
- Test: `pkg/daemon/process_windows_test.go` (`//go:build windows`)

**Interfaces:**
- Produces: `daemon.IsAlive(pid int) bool`, `daemon.DetachAttr() *syscall.SysProcAttr`.

- [ ] **Step 1: Write the failing test (unix)**

```go
// pkg/daemon/process_unix_test.go
//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAliveCurrentProcess(t *testing.T) {
	assert.True(t, IsAlive(os.Getpid()))
}

func TestIsAliveExitedProcess(t *testing.T) {
	cmd := exec.Command("true")
	assert.NoError(t, cmd.Run())
	assert.False(t, IsAlive(cmd.Process.Pid))
}

func TestDetachAttrNotNil(t *testing.T) {
	assert.NotNil(t, DetachAttr())
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/daemon/... -run 'TestIsAlive|TestDetachAttr' -v`
Expected: FAIL — `undefined: IsAlive`

- [ ] **Step 3: Write minimal implementation (unix)**

```go
// pkg/daemon/process_unix.go
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/daemon/... -v`
Expected: PASS

- [ ] **Step 5: Add the Windows variant (unverifiable on this machine — reviewed carefully, flagged in the PR as untested)**

```go
// pkg/daemon/process_windows.go
//go:build windows

package daemon

import (
	"syscall"

	"golang.org/x/sys/windows"
)

const detachedProcess = 0x00000008

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
	return code == uint32(windows.STILL_ACTIVE)
}

// DetachAttr returns the SysProcAttr that starts a child process detached
// from the parent's console and process group, so it survives the parent
// exiting.
func DetachAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | detachedProcess,
	}
}
```

```go
// pkg/daemon/process_windows_test.go
//go:build windows

package daemon

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsAliveCurrentProcess(t *testing.T) {
	assert.True(t, IsAlive(os.Getpid()))
}

func TestIsAliveExitedProcess(t *testing.T) {
	cmd := exec.Command("cmd.exe", "/c", "exit", "0")
	assert.NoError(t, cmd.Run())
	assert.False(t, IsAlive(cmd.Process.Pid))
}

func TestDetachAttrNotNil(t *testing.T) {
	assert.NotNil(t, DetachAttr())
}
```

- [ ] **Step 6: Promote `golang.org/x/sys` to a direct dependency**

Run: `go mod tidy`
Expected: `go.mod`'s `golang.org/x/sys` line loses its `// indirect` suffix (go mod tidy evaluates all build-tag variants, not just the current GOOS).

- [ ] **Step 7: Run the full test suite to confirm nothing else broke**

Run: `go build ./... && go test ./...`
Expected: PASS on this machine's platform; the Windows-tagged files won't compile here (expected — they're excluded by the build tag) but must be read carefully for correctness since they can't be tested locally.

- [ ] **Step 8: Commit**

```bash
git add pkg/daemon/process_unix.go pkg/daemon/process_unix_test.go pkg/daemon/process_windows.go pkg/daemon/process_windows_test.go go.mod go.sum
git commit -m "feat(daemon): add cross-platform process liveness and detach helpers"
```

---

### Task 3: `pkg/vpn` — `ConnectDetached`

**Files:**
- Modify: `pkg/vpn/client.go` (currently 17 lines, full content shown below)
- Test: `pkg/vpn/client_test.go`

**Interfaces:**
- Consumes: nothing new (stdlib + existing `pkg/exec`).
- Produces: `vpn.ConnectDetached(configPath, managementAddr string, logWriter io.Writer, sysProcAttr *syscall.SysProcAttr) (*exec.Cmd, error)` — starts openvpn with `--management <host> <port>` added to the existing flags, redirected to `logWriter`, detached via `sysProcAttr`, and returns immediately after `cmd.Start()` succeeds (caller calls `cmd.Wait()` independently). Existing `vpn.Connect(configPath string) error` is unchanged in behavior.

- [ ] **Step 1: Write the failing test**

```go
// pkg/vpn/client_test.go
package vpn

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectDetached(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ConnectDetached resolves an absolute openvpn.exe path on windows; not fakeable via PATH")
	}

	dir := t.TempDir()
	stub := filepath.Join(dir, "openvpn")
	argsFile := filepath.Join(dir, "args.txt")
	script := "#!/bin/sh\necho \"$@\" > \"" + argsFile + "\"\n"
	assert.NoError(t, os.WriteFile(stub, []byte(script), 0o755))

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	var log bytes.Buffer
	configPath := filepath.Join(dir, "config.ovpn")
	cmd, err := ConnectDetached(configPath, "127.0.0.1:12345", &log, nil)
	assert.NoError(t, err)
	assert.NoError(t, cmd.Wait())

	args, err := os.ReadFile(argsFile)
	assert.NoError(t, err)
	assert.Contains(t, string(args), "--management 127.0.0.1 12345")
	assert.Contains(t, string(args), "--config "+configPath)
}

func TestConnectDetachedMissingExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("ConnectDetached resolves an absolute openvpn.exe path on windows")
	}
	t.Setenv("PATH", t.TempDir())

	_, err := ConnectDetached("config.ovpn", "127.0.0.1:12345", &bytes.Buffer{}, nil)
	assert.Error(t, err)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/vpn/... -run TestConnectDetached -v`
Expected: FAIL — `undefined: ConnectDetached`

- [ ] **Step 3: Write minimal implementation**

Replace the entire contents of `pkg/vpn/client.go` with:

```go
package vpn

import (
	"fmt"
	"io"
	"net"
	osexec "os/exec"
	"runtime"
	"syscall"

	"github.com/davegallant/vpngate/pkg/exec"
)

// executablePath returns the platform-specific path to the openvpn
// binary.
func executablePath() string {
	if runtime.GOOS == "windows" {
		return `C:\Program Files\OpenVPN\bin\openvpn.exe`
	}
	return "openvpn"
}

// Connect to a specified OpenVPN configuration. Blocks until openvpn
// exits, streaming its output through pkg/exec's logger.
func Connect(configPath string) error {
	return exec.Run(executablePath(), ".", "--verb", "4", "--config", configPath, "--data-ciphers", "AES-128-CBC")
}

// ConnectDetached starts openvpn with a management interface enabled at
// managementAddr, detached via sysProcAttr so it outlives the calling
// process, writing its combined stdout/stderr to logWriter. It returns as
// soon as the process has started; callers wait on the returned *exec.Cmd
// independently (via cmd.Wait()) to learn when it exits.
func ConnectDetached(configPath, managementAddr string, logWriter io.Writer, sysProcAttr *syscall.SysProcAttr) (*osexec.Cmd, error) {
	executable := executablePath()
	if _, err := osexec.LookPath(executable); err != nil {
		return nil, err
	}

	host, port, err := net.SplitHostPort(managementAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid management address %q: %w", managementAddr, err)
	}

	cmd := osexec.Command(
		executable,
		"--verb", "4",
		"--config", configPath,
		"--data-ciphers", "AES-128-CBC",
		"--management", host, port,
	)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	cmd.SysProcAttr = sysProcAttr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return cmd, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/vpn/... -v`
Expected: PASS, including the pre-existing `TestGetListWithOptions`/`TestParseVpnList` (unaffected by this change).

- [ ] **Step 5: Commit**

```bash
git add pkg/vpn/client.go pkg/vpn/client_test.go
git commit -m "feat(vpn): add ConnectDetached for daemon-mode openvpn management"
```

---

### Task 4: `pkg/daemon` — OpenVPN management protocol client

**Files:**
- Create: `pkg/daemon/management.go`
- Test: `pkg/daemon/management_test.go`

**Interfaces:**
- Produces: `daemon.Management` (opaque struct), `daemon.DialManagement(addr string, timeout time.Duration) (*Management, error)`, `(*Management).State() (string, error)`, `(*Management).Disconnect() error`, `(*Management).Close() error`.

- [ ] **Step 1: Write the failing test**

```go
// pkg/daemon/management_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/daemon/... -run TestManagement -v`
Expected: FAIL — `undefined: DialManagement`

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/daemon/management.go
package daemon

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

// Management is a client for OpenVPN's plaintext management protocol. It
// is used only by the supervisor process, which is the sole client of a
// given OpenVPN instance's management socket — status/disconnect never
// dial it directly (see docs/superpowers/specs/2026-07-22-daemon-mode-design.md).
type Management struct {
	conn net.Conn
	r    *bufio.Reader
}

// DialManagement connects to addr and discards OpenVPN's greeting line.
func DialManagement(addr string, timeout time.Duration) (*Management, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}

	m := &Management{conn: conn, r: bufio.NewReader(conn)}
	if _, err := m.r.ReadString('\n'); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("reading management greeting: %w", err)
	}
	return m, nil
}

// Close closes the underlying connection.
func (m *Management) Close() error {
	return m.conn.Close()
}

// State queries OpenVPN's current connection state (e.g. "CONNECTED",
// "RECONNECTING", "EXITING").
func (m *Management) State() (string, error) {
	if _, err := m.conn.Write([]byte("state\n")); err != nil {
		return "", err
	}

	var lines []string
	for {
		line, err := m.r.ReadString('\n')
		if err != nil {
			return "", err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "END" {
			break
		}
		lines = append(lines, line)
	}
	return parseState(lines)
}

// parseState extracts the state field from OpenVPN's "state" command
// response. Each response line is comma-separated; the second field
// (index 1) is the connection state, per OpenVPN's management-notes.txt.
func parseState(lines []string) (string, error) {
	for _, line := range lines {
		fields := strings.Split(line, ",")
		if len(fields) >= 2 && fields[1] != "" {
			return fields[1], nil
		}
	}
	return "", fmt.Errorf("no state found in management response")
}

// Disconnect asks OpenVPN to shut down cleanly.
func (m *Management) Disconnect() error {
	_, err := m.conn.Write([]byte("signal SIGTERM\n"))
	return err
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/daemon/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/daemon/management.go pkg/daemon/management_test.go
git commit -m "feat(daemon): add OpenVPN management protocol client"
```

---

### Task 5: `pkg/daemon` — control protocol (supervisor ↔ status/disconnect)

**Files:**
- Create: `pkg/daemon/control.go`
- Test: `pkg/daemon/control_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks directly (decoupled from `Management` by design — the supervisor wires them together in Task 6).
- Produces: `daemon.Snapshot{State, HostName, IPAddr, CountryLong string; StartedAt time.Time; PID int}`, `daemon.NewControlServer(listener net.Listener, onStatus func() (Snapshot, error), onStop func()) *ControlServer`, `(*ControlServer).Addr() string`, `(*ControlServer).Serve()` (blocks until the listener is closed), `daemon.SendStatus(addr string, timeout time.Duration) (Snapshot, error)`, `daemon.SendStop(addr string, timeout time.Duration) error`.

- [ ] **Step 1: Write the failing test**

```go
// pkg/daemon/control_test.go
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
	defer ln.Close()

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
	defer ln.Close()

	server := NewControlServer(ln, func() (Snapshot, error) { return Snapshot{}, errors.New("boom") }, nil)
	go server.Serve()

	_, err = SendStatus(ln.Addr().String(), time.Second)
	assert.Error(t, err)
}

func TestControlStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer ln.Close()

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./pkg/daemon/... -run TestControl -v`
Expected: FAIL — `undefined: NewControlServer`

- [ ] **Step 3: Write minimal implementation**

```go
// pkg/daemon/control.go
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
	defer conn.Close()

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
	defer conn.Close()
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
	defer conn.Close()
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./pkg/daemon/... -v`
Expected: PASS (all of Tasks 1, 2, 4, 5's tests together)

- [ ] **Step 5: Commit**

```bash
git add pkg/daemon/control.go pkg/daemon/control_test.go
git commit -m "feat(daemon): add supervisor control protocol (STATUS/STOP)"
```

---

### Task 6: `cmd` — daemon supervisor loop

**Files:**
- Create: `cmd/daemon_supervisor.go`
- Test: `cmd/daemon_supervisor_test.go`

**Interfaces:**
- Consumes: `daemon.Dir/StatePath/ConfigPath/LogPath`, `daemon.State`, `daemon.Save/Remove`, `daemon.DetachAttr`, `daemon.Management`/`DialManagement`, `daemon.Snapshot`, `daemon.NewControlServer`, `vpn.Server`, `vpn.ConnectDetached`, package-level flags `flagProxy, flagSocks5Proxy, flagRefresh, flagNoCache, flagRandom, flagReconnect, flagDaemonHostname` (all already declared in `cmd/connect.go` — Task 7 adds the last three), and `filterServers` (from `cmd/servers.go`).
- Produces: `runSupervisor() error` (called from `cmd/connect.go`'s `RunE` in Task 7), plus the unexported `reserveLoopbackAddr() (string, error)` and `waitForManagement(addr string, timeout time.Duration) (*daemon.Management, error)` helpers, tested independently of any real openvpn process.

- [ ] **Step 1: Write the failing test**

These two helpers are the only pieces of the supervisor testable without a real `openvpn` binary — the rest (`runSupervisor`, `supervisor.run`, `connectOnce`) is exercised by manual testing per the spec's testing plan.

```go
// cmd/daemon_supervisor_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestReserveLoopbackAddr|TestWaitForManagement' -v`
Expected: FAIL — `undefined: reserveLoopbackAddr`

- [ ] **Step 3: Write minimal implementation**

```go
// cmd/daemon_supervisor.go
package cmd

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"net"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/daemon"
	"github.com/davegallant/vpngate/pkg/vpn"
)

// supervisor owns a single daemon's lifecycle: it launches openvpn,
// tracks the currently-active management connection, answers control
// requests from separate `status`/`disconnect` invocations, and — when
// --reconnect is set — restarts openvpn if it exits on its own.
type supervisor struct {
	vpnServers []vpn.Server
	random     bool
	reconnect  bool
	logFile    *os.File
	control    *daemon.ControlServer

	mu        sync.Mutex
	server    vpn.Server
	startedAt time.Time
	mgmt      *daemon.Management
	stopping  bool
}

// runSupervisor is the entry point used when connect is re-exec'd with
// --__daemon-run: it resolves the server to connect to, opens the
// control socket, and runs the connect/reconnect loop until told to
// stop.
func runSupervisor() error {
	vpnServers, err := vpn.GetListWithOptions(flagProxy, flagSocks5Proxy, vpn.ListOptions{Refresh: flagRefresh, NoCache: flagNoCache})
	if err != nil {
		return err
	}
	filtered := *filterServers(vpnServers)
	if len(filtered) == 0 {
		return fmt.Errorf("no vpn servers matched the provided filters")
	}

	var initial vpn.Server
	if flagRandom {
		initial = filtered[rand.Intn(len(filtered))]
	} else {
		found := false
		for _, s := range filtered {
			if s.HostName == flagDaemonHostname {
				initial = s
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("server %q was not found", flagDaemonHostname)
		}
	}

	if err := os.MkdirAll(daemon.Dir(), 0o755); err != nil {
		return err
	}
	logFile, err := os.OpenFile(daemon.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer logFile.Close()

	controlLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s := &supervisor{
		vpnServers: filtered,
		random:     flagRandom,
		reconnect:  flagReconnect,
		logFile:    logFile,
		server:     initial,
	}
	s.control = daemon.NewControlServer(controlLn, s.handleStatus, s.handleStop)
	go s.control.Serve()

	return s.run()
}

func (s *supervisor) run() error {
	defer func() {
		_ = daemon.Remove()
		_ = os.Remove(daemon.ConfigPath())
	}()

	for {
		s.mu.Lock()
		if s.stopping {
			s.mu.Unlock()
			return nil
		}
		if s.random {
			s.server = s.vpnServers[rand.Intn(len(s.vpnServers))]
		}
		server := s.server
		s.mu.Unlock()

		err := s.connectOnce(server)
		if err != nil {
			log.Error().Err(err).Msg("daemon connection attempt failed")
			if !s.reconnect {
				return err
			}
		}

		s.mu.Lock()
		stopping := s.stopping
		s.mu.Unlock()
		if stopping || !s.reconnect {
			return nil
		}
	}
}

// connectOnce starts openvpn for server, waits for it to report a
// successful connection, records daemon state, and blocks until it
// exits (either on its own or because handleStop signaled it).
func (s *supervisor) connectOnce(server vpn.Server) error {
	decoded, err := base64.StdEncoding.DecodeString(server.OpenVpnConfigData)
	if err != nil {
		return err
	}
	if err := os.WriteFile(daemon.ConfigPath(), decoded, 0o600); err != nil {
		return err
	}

	mgmtAddr, err := reserveLoopbackAddr()
	if err != nil {
		return err
	}

	cmd, err := vpn.ConnectDetached(daemon.ConfigPath(), mgmtAddr, s.logFile, daemon.DetachAttr())
	if err != nil {
		return fmt.Errorf("starting openvpn: %w", err)
	}

	mgmt, err := waitForManagement(mgmtAddr, 30*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	startedAt := time.Now()
	s.mu.Lock()
	if s.stopping {
		// disconnect() arrived while we were still connecting — undo
		// this attempt instead of publishing state for a connection
		// nobody asked for.
		s.mu.Unlock()
		_ = mgmt.Disconnect()
		_ = mgmt.Close()
		_ = cmd.Wait()
		return nil
	}
	s.server = server
	s.startedAt = startedAt
	s.mgmt = mgmt
	s.mu.Unlock()

	if err := daemon.Save(daemon.State{
		PID:         os.Getpid(),
		ControlAddr: s.control.Addr(),
		HostName:    server.HostName,
		IPAddr:      server.IPAddr,
		CountryLong: server.CountryLong,
		StartedAt:   startedAt,
	}); err != nil {
		_ = mgmt.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	log.Info().Msgf("Connected in background to %s (%s) in %s", server.HostName, server.IPAddr, server.CountryLong)

	waitErr := cmd.Wait()

	s.mu.Lock()
	if s.mgmt != nil {
		_ = s.mgmt.Close()
		s.mgmt = nil
	}
	s.mu.Unlock()

	return waitErr
}

// reserveLoopbackAddr picks a free loopback TCP port by opening then
// immediately closing a listener, so the caller can hand the address to
// a separate process (openvpn) rather than a listener it can't pass on.
func reserveLoopbackAddr() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		return "", err
	}
	return addr, nil
}

// waitForManagement polls addr until OpenVPN's management interface
// accepts a connection, or timeout elapses.
func waitForManagement(addr string, timeout time.Duration) (*daemon.Management, error) {
	deadline := time.Now().Add(timeout)
	for {
		mgmt, err := daemon.DialManagement(addr, time.Second)
		if err == nil {
			return mgmt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for openvpn management interface: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// handleStatus answers a STATUS control request with the supervisor's
// current view of the connection.
func (s *supervisor) handleStatus() (daemon.Snapshot, error) {
	s.mu.Lock()
	mgmt := s.mgmt
	server := s.server
	startedAt := s.startedAt
	s.mu.Unlock()

	state := "CONNECTING"
	if mgmt != nil {
		if st, err := mgmt.State(); err == nil {
			state = st
		}
	}

	return daemon.Snapshot{
		State:       state,
		HostName:    server.HostName,
		IPAddr:      server.IPAddr,
		CountryLong: server.CountryLong,
		StartedAt:   startedAt,
		PID:         os.Getpid(),
	}, nil
}

// handleStop answers a STOP control request: it marks the supervisor as
// stopping (so the reconnect loop in run() won't respawn openvpn) and,
// if openvpn is currently connected, asks it to exit cleanly. run()
// observes s.stopping either when connectOnce's cmd.Wait() returns or,
// if no openvpn is running yet, on its next loop iteration.
func (s *supervisor) handleStop() {
	s.mu.Lock()
	s.stopping = true
	mgmt := s.mgmt
	s.mu.Unlock()

	if mgmt != nil {
		_ = mgmt.Disconnect()
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -v`
Expected: PASS (new tests; existing `cmd` tests unaffected since `runSupervisor`/`flagDaemonHostname` etc. aren't wired into `RunE` yet — that's Task 7)

Note: this step will not compile until Task 7 adds the `flagDaemonHostname` (and other) package-level vars this file references — implement Task 6 and Task 7 together if running tests standalone, or stub the vars temporarily. Recommended: implement Task 7 immediately after this step, then run both tasks' tests together before committing either.

- [ ] **Step 5: Commit** (after Task 7's vars exist so the package compiles)

```bash
git add cmd/daemon_supervisor.go cmd/daemon_supervisor_test.go
git commit -m "feat(cmd): add daemon supervisor loop"
```

---

### Task 7: `cmd/connect.go` — `-d/--daemon` flag and re-exec

**Files:**
- Modify: `cmd/connect.go` (full replacement shown below)
- Modify: `cmd/connect_test.go` (append new test)

**Interfaces:**
- Consumes: `daemon.Load/Save/Remove/IsAlive/DetachAttr/LogPath`, `runSupervisor()` (Task 6).
- Produces: package-level vars `flagDaemon bool`, `flagDaemonRun bool`, `flagDaemonHostname string` (consumed by Task 6's `daemon_supervisor.go`), and the `--daemon`/`-d` flag on `connect`.

- [ ] **Step 1: Write the failing test**

```go
// append to cmd/connect_test.go
func TestForwardableConnectArgs(t *testing.T) {
	flagReconnect = true
	flagRandom = false
	flagProxy = "http://127.0.0.1:8080"
	flagSocks5Proxy = ""
	flagCountry = "Japan"
	flagMaxPing = 100
	flagMinScore = 0
	flagRefresh = true
	flagNoCache = false
	t.Cleanup(func() {
		flagReconnect = false
		flagProxy = ""
		flagCountry = ""
		flagMaxPing = 0
		flagRefresh = false
	})

	args := forwardableConnectArgs()
	assert.Equal(t, []string{
		"--reconnect",
		"--proxy", "http://127.0.0.1:8080",
		"--country", "Japan",
		"--max-ping", "100",
		"--refresh",
	}, args)
}

func TestTailLogMissingFile(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())
	assert.Equal(t, "", tailLog())
}
```

Add `"github.com/davegallant/vpngate/pkg/daemon"` to `cmd/connect_test.go`'s import block.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run 'TestForwardableConnectArgs|TestTailLog' -v`
Expected: FAIL — `undefined: forwardableConnectArgs`

- [ ] **Step 3: Write minimal implementation**

Replace the entire contents of `cmd/connect.go` with:

```go
package cmd

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	osexec "os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/daemon"
	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/spf13/cobra"
)

var (
	flagRandom         bool
	flagReconnect      bool
	flagProxy          string
	flagSocks5Proxy    string
	flagDaemon         bool
	flagDaemonRun      bool
	flagDaemonHostname string
)

func init() {
	connectCmd.Flags().BoolVarP(&flagRandom, "random", "r", false, "connect to a random server")
	connectCmd.Flags().BoolVarP(&flagReconnect, "reconnect", "t", false, "continually attempt to connect to the server")
	connectCmd.Flags().StringVarP(&flagProxy, "proxy", "p", "", "provide a http/https proxy server to make requests through (i.e. http://127.0.0.1:8080)")
	connectCmd.Flags().StringVarP(&flagSocks5Proxy, "socks5", "s", "", "provide a socks5 proxy server to make requests through (i.e. 127.0.0.1:1080)")
	connectCmd.Flags().StringVar(&flagCountry, "country", "", "filter by country name or country code (i.e. Japan or jp)")
	connectCmd.Flags().IntVar(&flagMaxPing, "max-ping", 0, "filter out servers with ping higher than this value")
	connectCmd.Flags().IntVar(&flagMinScore, "min-score", 0, "filter out servers with score lower than this value")
	connectCmd.Flags().BoolVar(&flagRefresh, "refresh", false, "refresh the vpn server list cache before connecting")
	connectCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "do not read from or write to the vpn server list cache")
	connectCmd.Flags().BoolVarP(&flagDaemon, "daemon", "d", false, "run the connection in the background; see 'vpngate status' and 'vpngate disconnect'")
	connectCmd.Flags().BoolVar(&flagDaemonRun, "__daemon-run", false, "internal: run as the background daemon supervisor")
	connectCmd.Flags().StringVar(&flagDaemonHostname, "__daemon-hostname", "", "internal: hostname resolved by the foreground process")
	_ = connectCmd.Flags().MarkHidden("__daemon-run")
	_ = connectCmd.Flags().MarkHidden("__daemon-hostname")
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a vpn server (survey selection appears if hostname is not provided)",
	Long:  `Connect to a vpn from a list of relay servers. Because openvpn creates a network interface, run the connect command with 'sudo' or a user with escalated privileges.`,
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagDaemonRun {
			return runSupervisor()
		}

		vpnServers, err := vpn.GetListWithOptions(flagProxy, flagSocks5Proxy, vpn.ListOptions{Refresh: flagRefresh, NoCache: flagNoCache})
		if err != nil {
			return err
		}

		vpnServers = filterServers(vpnServers)
		if len(*vpnServers) == 0 {
			return fmt.Errorf("no vpn servers matched the provided filters")
		}

		// Build rich server selection options and lookup map.
		serverSelection, serverMap := buildServerSelection(*vpnServers)

		selection := ""
		var serverSelected vpn.Server

		if !flagRandom {
			if len(args) > 0 {
				selection = args[0]
			} else {
				prompt := &survey.Select{
					Message: "Choose a server:",
					Options: serverSelection,
				}
				if err := survey.AskOne(prompt, &selection, survey.WithPageSize(10)); err != nil {
					return fmt.Errorf("unable to obtain hostname from survey: %w", err)
				}
			}

			// Lookup server from selection using map for O(1) lookup.
			if server, exists := serverMap[selection]; exists {
				serverSelected = server
			} else if server, exists := serverMap[extractHostname(selection)]; exists {
				serverSelected = server
			} else {
				return fmt.Errorf("server %q was not found", selection)
			}
		}

		if flagDaemon {
			return startDaemon(serverSelected)
		}

		for {
			if flagRandom {
				// Select a random server
				serverSelected = (*vpnServers)[rand.Intn(len(*vpnServers))]
			}

			decodedConfig, err := base64.StdEncoding.DecodeString(serverSelected.OpenVpnConfigData)
			if err != nil {
				return err
			}

			tmpfile, err := os.CreateTemp("", "vpngate-openvpn-config-")
			if err != nil {
				return err
			}

			if _, err := tmpfile.Write(decodedConfig); err != nil {
				_ = tmpfile.Close()
				_ = os.Remove(tmpfile.Name())
				return err
			}

			if err := tmpfile.Close(); err != nil {
				_ = os.Remove(tmpfile.Name())
				return err
			}

			log.Info().Msgf("Connecting to %s (%s) in %s", serverSelected.HostName, serverSelected.IPAddr, serverSelected.CountryLong)

			err = vpn.Connect(tmpfile.Name())

			// Always try to clean up temporary file
			_ = os.Remove(tmpfile.Name())

			if !flagReconnect {
				if err != nil {
					return fmt.Errorf("vpn connection failed: %w", err)
				}
				return nil
			}
		}
	},
}

// startDaemon re-execs the current binary detached from the terminal so
// it can run connect in the background, then waits for it to report a
// successful connection. serverSelected is the zero value when --random
// was passed — the daemon resolves its own server in that case, possibly
// reselecting on every reconnect attempt.
func startDaemon(serverSelected vpn.Server) error {
	if state, err := daemon.Load(); err == nil {
		if daemon.IsAlive(state.PID) {
			return fmt.Errorf("already connected to %s (PID %d); run 'vpngate disconnect' first", state.HostName, state.PID)
		}
		_ = daemon.Remove()
	} else if !os.IsNotExist(err) {
		return err
	}

	selfPath, err := os.Executable()
	if err != nil {
		return err
	}

	childArgs := []string{"connect", "--__daemon-run"}
	if !flagRandom {
		childArgs = append(childArgs, "--__daemon-hostname", serverSelected.HostName)
	}
	childArgs = append(childArgs, forwardableConnectArgs()...)

	child := osexec.Command(selfPath, childArgs...)
	child.SysProcAttr = daemon.DetachAttr()

	if err := child.Start(); err != nil {
		return fmt.Errorf("starting background daemon: %w", err)
	}
	if err := child.Process.Release(); err != nil {
		return err
	}

	return waitForDaemonReady(30 * time.Second)
}

// forwardableConnectArgs reproduces the subset of connect's own flags
// that the re-exec'd daemon supervisor needs to repeat the same server
// selection and connection behavior.
func forwardableConnectArgs() []string {
	var args []string
	if flagReconnect {
		args = append(args, "--reconnect")
	}
	if flagRandom {
		args = append(args, "--random")
	}
	if flagProxy != "" {
		args = append(args, "--proxy", flagProxy)
	}
	if flagSocks5Proxy != "" {
		args = append(args, "--socks5", flagSocks5Proxy)
	}
	if flagCountry != "" {
		args = append(args, "--country", flagCountry)
	}
	if flagMaxPing != 0 {
		args = append(args, "--max-ping", strconv.Itoa(flagMaxPing))
	}
	if flagMinScore != 0 {
		args = append(args, "--min-score", strconv.Itoa(flagMinScore))
	}
	if flagRefresh {
		args = append(args, "--refresh")
	}
	if flagNoCache {
		args = append(args, "--no-cache")
	}
	return args
}

// waitForDaemonReady polls for the daemon's state file to appear,
// signalling a successful first connection, surfacing the tail of the
// daemon log if it times out instead.
func waitForDaemonReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		state, err := daemon.Load()
		if err == nil {
			fmt.Printf("Connected in background to %s (PID %d)\n", state.HostName, state.PID)
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for background connection; see %s\n%s", daemon.LogPath(), tailLog())
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// tailLog returns the last few lines of the daemon log for error
// messages, or an empty string if it can't be read.
func tailLog() string {
	data, err := os.ReadFile(daemon.LogPath())
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) > 10 {
		lines = lines[len(lines)-10:]
	}
	return strings.Join(lines, "\n")
}

func buildServerSelection(servers []vpn.Server) ([]string, map[string]vpn.Server) {
	hostnameWidth := len("Hostname")
	countryWidth := len("Country")
	for _, server := range servers {
		if len(server.HostName) > hostnameWidth {
			hostnameWidth = len(server.HostName)
		}
		if len(server.CountryLong) > countryWidth {
			countryWidth = len(server.CountryLong)
		}
	}

	serverSelection := make([]string, len(servers))
	serverMap := make(map[string]vpn.Server, len(servers)*2)
	for i, server := range servers {
		label := formatServerSelection(server, hostnameWidth, countryWidth)
		serverSelection[i] = label
		serverMap[label] = server
		serverMap[server.HostName] = server
	}

	return serverSelection, serverMap
}

func formatServerSelection(server vpn.Server, hostnameWidth int, countryWidth int) string {
	return fmt.Sprintf(
		"%-*s  %-*s  %-15s  ping %s",
		hostnameWidth,
		server.HostName,
		countryWidth,
		server.CountryLong,
		server.IPAddr,
		server.Ping,
	)
}

// extractHostname extracts the hostname from a manually provided argument or legacy selection string.
func extractHostname(selection string) string {
	selection = strings.TrimSpace(selection)

	parts := strings.Split(selection, " | ")
	if len(parts) > 0 {
		selection = strings.TrimSpace(parts[0])
	}

	parts = strings.Split(selection, " (")
	if len(parts) > 0 {
		selection = strings.TrimSpace(parts[0])
	}

	parts = strings.Fields(selection)
	if len(parts) > 0 {
		return parts[0]
	}

	return selection
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... -v`
Expected: PASS across `pkg/daemon`, `pkg/vpn`, and `cmd` (this is the point where Task 6's `cmd/daemon_supervisor.go` finally compiles, since `flagDaemonHostname` etc. now exist)

- [ ] **Step 5: Manually smoke-test flag wiring (no real connection)**

Run: `go run . connect --help`
Expected: `-d, --daemon` is listed; `--__daemon-run` and `--__daemon-hostname` are NOT listed (hidden)

- [ ] **Step 6: Commit**

```bash
git add cmd/connect.go cmd/connect_test.go
git commit -m "feat(connect): add -d/--daemon flag with re-exec supervisor"
```

---

### Task 8: `cmd/status.go`

**Files:**
- Create: `cmd/status.go`
- Test: `cmd/status_test.go`

**Interfaces:**
- Consumes: `daemon.Load/IsAlive/Remove/SendStatus`, `daemon.State`, `daemon.Snapshot`, `daemon.DirEnvVar`, `daemon.NewControlServer` (for the test's fake server).
- Produces: `vpngate status` command (registered on `rootCmd`).

- [ ] **Step 1: Write the failing test**

```go
// cmd/status_test.go
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
	defer ln.Close()

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run TestStatus -v`
Expected: FAIL — `undefined: statusCmd`

- [ ] **Step 3: Write minimal implementation**

```go
// cmd/status.go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/davegallant/vpngate/pkg/daemon"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of a background vpn connection started with 'connect -d'",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := daemon.Load()
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not connected.")
				return nil
			}
			return err
		}

		if !daemon.IsAlive(state.PID) {
			_ = daemon.Remove()
			fmt.Println("Not connected.")
			return nil
		}

		snap, err := daemon.SendStatus(state.ControlAddr, 5*time.Second)
		if err != nil {
			fmt.Printf("Status:  unknown (control socket unreachable: %v)\n", err)
			fmt.Printf("Server:  %s (%s) - %s\n", state.HostName, state.IPAddr, state.CountryLong)
			fmt.Printf("PID:     %d\n", state.PID)
			return nil
		}

		fmt.Printf("Status:  %s\n", snap.State)
		fmt.Printf("Server:  %s (%s) - %s\n", snap.HostName, snap.IPAddr, snap.CountryLong)
		if !snap.StartedAt.IsZero() {
			fmt.Printf("Uptime:  %s\n", time.Since(snap.StartedAt).Round(time.Second))
		}
		fmt.Printf("PID:     %d\n", snap.PID)
		return nil
	},
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/... -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/status.go cmd/status_test.go
git commit -m "feat(cmd): add 'vpngate status'"
```

---

### Task 9: `cmd/disconnect.go`

**Files:**
- Create: `cmd/disconnect.go`
- Test: `cmd/disconnect_test.go`

**Interfaces:**
- Consumes: `daemon.Load/IsAlive/Remove/SendStop`, `daemon.State`, `daemon.DirEnvVar`, `daemon.NewControlServer` (for the test's fake server), and the `captureStdout(t *testing.T, fn func()) string` helper defined in `cmd/status_test.go` (Task 8) — Task 9 must run after Task 8.
- Produces: `vpngate disconnect` command (registered on `rootCmd`).

- [ ] **Step 1: Write the failing test**

```go
// cmd/disconnect_test.go
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

func TestDisconnectSendsStop(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	defer ln.Close()

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run TestDisconnect -v`
Expected: FAIL — `undefined: disconnectCmd`

- [ ] **Step 3: Write minimal implementation**

```go
// cmd/disconnect.go
package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/davegallant/vpngate/pkg/daemon"
)

func init() {
	rootCmd.AddCommand(disconnectCmd)
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect a background vpn connection started with 'connect -d'",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := daemon.Load()
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not connected.")
				return nil
			}
			return err
		}

		if !daemon.IsAlive(state.PID) {
			_ = daemon.Remove()
			fmt.Println("Not connected.")
			return nil
		}

		if err := daemon.SendStop(state.ControlAddr, 5*time.Second); err != nil {
			// Control socket unreachable (e.g. the supervisor crashed):
			// fall back to killing it directly and cleaning up ourselves.
			if proc, ferr := os.FindProcess(state.PID); ferr == nil {
				_ = proc.Kill()
			}
			_ = daemon.Remove()
		}

		fmt.Println("Disconnected.")
		return nil
	},
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./... -v`
Expected: PASS across the whole module

- [ ] **Step 5: Commit**

```bash
git add cmd/disconnect.go cmd/disconnect_test.go
git commit -m "feat(cmd): add 'vpngate disconnect'"
```

---

### Task 10: Docs and manual verification

**Files:**
- Modify: `README.md` (add a "Run in the background" subsection under Usage/Examples)
- Regenerate: `docs/cli/vpngate_connect.md`, create `docs/cli/vpngate_status.md`, `docs/cli/vpngate_disconnect.md` (all via `make docs`, not hand-edited)

**Interfaces:**
- Consumes: the finished `connect -d`/`status`/`disconnect` commands from Tasks 7–9.
- Produces: updated docs only — no new code.

- [ ] **Step 1: Regenerate CLI reference docs**

Run: `make docs`
Expected: `docs/cli/vpngate_connect.md` gains the `--daemon`/`-d` flag; new `docs/cli/vpngate_status.md` and `docs/cli/vpngate_disconnect.md` appear; `docs/cli/vpngate.md` lists the two new subcommands. `--__daemon-run`/`--__daemon-hostname` do NOT appear anywhere (hidden flags are excluded from cobra's generated docs).

- [ ] **Step 2: Add a README section**

Find the `### Examples` section in `README.md` (after the `> If on macOS...` PATH note) and add, before the first existing example:

```markdown
Run in the background, then check on it or disconnect later:

```shell
sudo vpngate connect -d --country Japan
vpngate status
vpngate disconnect
```
```

- [ ] **Step 3: Manual verification (cannot run in CI/sandbox — real openvpn + real network required)**

On macOS or Linux, with openvpn installed:

```bash
sudo go run . connect -d --country Japan
go run . status
go run . disconnect
go run . status
```

Expected: first command prints `Connected in background to <hostname> (PID <pid>)` and returns control of the terminal; `status` shows `CONNECTED` with a growing uptime; `disconnect` prints `Disconnected.`; the final `status` prints `Not connected.`. Also verify `curl ipinfo.io` reflects the VPN's exit IP while connected (same manual check the README already suggests for the foreground path).

Additionally verify `sudo go run . connect -d --country Japan --reconnect` survives an `openvpn` process being killed externally (`sudo pkill openvpn`) — `status` should briefly show a reconnect and then `CONNECTED` again with a new PID for the openvpn child (the daemon's own `Status: PID` continues to reflect the supervisor, which does not change).

The Windows detach/process-liveness path (`process_windows.go`) cannot be verified in this environment — call this out explicitly as untested in the PR description.

- [ ] **Step 4: Run full test + lint suite**

Run: `go build ./... && go test ./... && make lint`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md docs/cli
git commit -m "docs: document daemon mode (connect -d, status, disconnect)"
```

---

## Self-Review Notes

- **Spec coverage:** `connect -d` (Tasks 6–7), `status` (Task 8), `disconnect` (Task 9), control-socket architecture instead of direct management-socket access (Tasks 5–6), `os.TempDir()`-based state dir instead of `~/.vpngate` (Task 1), stale-PID cleanup (Tasks 8–9), reconnect-vs-disconnect race fix from the design review (Task 6, `connectOnce`'s stopping re-check) — all covered.
- **Placeholder scan:** none left; the one caveat note in Task 8 about `bytes.MinRead` was corrected to explicitly say not to include it.
- **Type consistency:** `daemon.State`, `daemon.Snapshot`, `vpn.ConnectDetached`'s signature, and `daemon.DetachAttr()`'s return type are used identically across Tasks 3, 6, 7, 8, and 9.

## Post-Implementation Correction (found during execution, not in the original plan)

Task 1's `Dir()` as originally planned (`os.TempDir()/vpngate`) turned out to have the *same* sudo-vs-non-sudo divergence problem the design review already fixed once for `~/.vpngate`, just one level down: on macOS, `$TMPDIR` is a per-user path assigned by launchd, and `sudo` does not preserve it by default, so a root `connect -d` and a non-root `status`/`disconnect` would resolve to two different directories for the same daemon (`status` reporting "Not connected" while actually connected). Fixed by adding a `defaultBaseDir()` split into `process_unix.go` (`/tmp`, a fixed literal, not `$TMPDIR`) and `process_windows.go` (`%ProgramData%`, falling back to `os.TempDir()` only if unset) — see `docs/superpowers/specs/2026-07-22-daemon-mode-design.md`'s "`Dir()` resolution (corrected during implementation)" note for the full explanation, and `pkg/daemon/process_unix_test.go`'s `TestDirIsFixedNotPerUserTemp` for the regression test that locks it in. This was caught by an advisor review after Task 10, not by the plan's own test suite — the tests as originally speced all pin `VPNGATE_DAEMON_DIR` to one directory for both "sides" of a comparison, which structurally cannot catch a divergence that only exists when the override is absent.
