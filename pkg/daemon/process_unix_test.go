//go:build !windows

package daemon

import (
	"os"
	"os/exec"
	"strings"
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

// TestDirIsFixedNotPerUserTemp locks in the fix for a real bug: Dir()
// must NOT depend on os.TempDir()/$TMPDIR, because on macOS $TMPDIR is a
// per-user path assigned by launchd that sudo does not preserve — a root
// `connect -d` and a non-root `status`/`disconnect` would otherwise
// resolve to two different directories for the same daemon. Unsetting
// TMPDIR here simulates that divergence; Dir() must be unaffected. It
// also must not be under /tmp: /tmp is world-writable, which lets an
// unprivileged user pre-create Dir()'s path before the root supervisor
// does (see process_unix.go's defaultBaseDir doc comment).
func TestDirIsFixedNotPerUserTemp(t *testing.T) {
	t.Setenv("TMPDIR", "/some/other/per-user/temp/dir")
	assert.True(t, strings.HasPrefix(Dir(), "/var/run/vpngate"), "Dir() = %q, want prefix /var/run/vpngate", Dir())
}
