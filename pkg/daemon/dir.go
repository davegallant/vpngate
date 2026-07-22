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
