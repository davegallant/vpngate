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
