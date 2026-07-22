package daemon

import (
	"os"
	"path/filepath"
)

// DirEnvVar overrides the base directory Dir() resolves under. Tests (in
// this package and elsewhere) set it to isolate state files instead of
// touching the real default location.
const DirEnvVar = "VPNGATE_DAEMON_DIR"

// Dir returns the directory vpngate uses for daemon state, the persisted
// OpenVPN config, and the daemon log. It deliberately avoids both $HOME
// and os.TempDir()/$TMPDIR: daemon mode is typically launched with sudo
// (root), while status/disconnect are typically run as the invoking
// user, and both $HOME and $TMPDIR can differ between the two — on
// macOS in particular, $TMPDIR is a per-user path assigned by launchd
// (e.g. /var/folders/.../T/) that sudo does not preserve by default, so
// os.TempDir() alone resolves to two different directories for the same
// daemon depending on who's asking. defaultBaseDir() (unix/windows split)
// picks a single fixed, machine-wide location instead, so every
// invocation agrees regardless of privilege.
func Dir() string {
	base := defaultBaseDir()
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
