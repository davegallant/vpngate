package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/davegallant/vpngate/pkg/daemon"
)

func TestLogsNoLogYet(t *testing.T) {
	t.Setenv(daemon.DirEnvVar, t.TempDir())

	out := captureStdout(t, func() {
		assert.NoError(t, logsCmd.RunE(logsCmd, nil))
	})
	assert.Contains(t, out, "No daemon log yet")
}

func TestLogsPrintsWholeFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(daemon.DirEnvVar, dir)
	assert.NoError(t, os.MkdirAll(daemon.Dir(), 0o700))
	assert.NoError(t, os.WriteFile(daemon.LogPath(), []byte("line1\nline2\nline3\n"), 0o644))

	out := captureStdout(t, func() {
		assert.NoError(t, logsCmd.RunE(logsCmd, nil))
	})
	assert.Equal(t, "line1\nline2\nline3\n", out)
}

func TestLogsLinesFlag(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(daemon.DirEnvVar, dir)
	assert.NoError(t, os.MkdirAll(daemon.Dir(), 0o700))
	assert.NoError(t, os.WriteFile(daemon.LogPath(), []byte("line1\nline2\nline3\n"), 0o644))

	flagLogsLines = 2
	defer func() { flagLogsLines = 0 }()

	out := captureStdout(t, func() {
		assert.NoError(t, logsCmd.RunE(logsCmd, nil))
	})
	assert.Equal(t, "line2\nline3\n", out)
}

func TestLogsPermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission checks don't apply when running as root")
	}

	dir := t.TempDir()
	t.Setenv(daemon.DirEnvVar, dir)
	assert.NoError(t, os.MkdirAll(daemon.Dir(), 0o700))
	assert.NoError(t, os.WriteFile(daemon.LogPath(), []byte("secret\n"), 0o000))

	out := captureStdout(t, func() {
		assert.NoError(t, logsCmd.RunE(logsCmd, nil))
	})
	assert.Contains(t, out, "Insufficient permissions")
}

func TestLastLines(t *testing.T) {
	assert.Equal(t, "", lastLines([]byte(""), 5))
	assert.Equal(t, "a\n", lastLines([]byte("a"), 5))
	assert.Equal(t, "b\nc\n", lastLines([]byte("a\nb\nc\n"), 2))
	assert.Equal(t, "a\nb\nc\n", lastLines([]byte("a\nb\nc\n"), 10))
}
