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
