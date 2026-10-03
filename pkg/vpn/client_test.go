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

func TestSanitizeConfig(t *testing.T) {
	config := []byte("client\n" +
		"dev tun\n" +
		"up /etc/openvpn/update-resolv-conf\n" +
		"DOWN /etc/openvpn/update-resolv-conf\n" +
		"plugin /usr/lib/openvpn/plugins/openvpn-plugin-auth-pam.so login\n" +
		"script-security 2\n" +
		"# up is commented out here\n" +
		";down also commented\n" +
		"auth-user-pass\n" +
		"remote vpn.example.com 1194\n")

	sanitized, stripped := SanitizeConfig(config)

	assert.Equal(t, []string{"up", "down", "plugin", "script-security"}, stripped)
	out := string(sanitized)
	assert.Contains(t, out, "# vpngate disabled script directive: up /etc/openvpn/update-resolv-conf")
	assert.Contains(t, out, "# vpngate disabled script directive: DOWN /etc/openvpn/update-resolv-conf")
	// Comments mentioning directives are left alone.
	assert.Contains(t, out, "# up is commented out here")
	assert.Contains(t, out, ";down also commented")
	// Non-script directives pass through untouched.
	assert.Contains(t, out, "\nauth-user-pass\n")
	assert.Contains(t, out, "\nremote vpn.example.com 1194\n")
}

func TestSanitizeConfigClean(t *testing.T) {
	config := []byte("client\ndev tun\nremote vpn.example.com 1194\n")
	sanitized, stripped := SanitizeConfig(config)
	assert.Empty(t, stripped)
	assert.Equal(t, config, sanitized)
}
