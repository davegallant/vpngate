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
