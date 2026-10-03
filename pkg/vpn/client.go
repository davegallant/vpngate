package vpn

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	osexec "os/exec"
	"runtime"
	"strings"
	"syscall"

	"github.com/davegallant/vpngate/pkg/exec"
)

// dataCiphers is the data-channel cipher list offered to the server.
// VPNGate hosts a mix of modern and legacy servers, so GCM comes first
// with CBC kept as a fallback for servers that only speak the older
// modes.
const dataCiphers = "AES-256-GCM:AES-128-GCM:AES-256-CBC:AES-128-CBC"

// scriptDirectives are OpenVPN config directives that can execute
// arbitrary programs or load plugins. VPNGate configs come from
// volunteer-run servers, so these are stripped before the config is
// handed to openvpn rather than trusted.
var scriptDirectives = map[string]bool{
	"up":                    true,
	"down":                  true,
	"route-up":              true,
	"route-pre-down":        true,
	"ipchange":              true,
	"tls-verify":            true,
	"auth-user-pass-verify": true,
	"client-connect":        true,
	"client-disconnect":     true,
	"learn-address":         true,
	"plugin":                true,
	"script-security":       true,
	"management":            true,
}

// SanitizeConfig comments out any script/plugin directives in an OpenVPN
// config, returning the sanitized config and the names of the stripped
// directives (for logging). Comment lines and blank lines pass through
// untouched; matching is on the first whitespace-separated token,
// case-insensitively, so e.g. `up` is stripped but `update-resolv-conf`
// would not be. The original newline structure is preserved byte for
// byte otherwise.
func SanitizeConfig(config []byte) ([]byte, []string) {
	var out bytes.Buffer
	var stripped []string
	lines := bytes.Split(config, []byte("\n"))
	for i, line := range lines {
		if i > 0 {
			out.WriteByte('\n')
		}
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 || trimmed[0] == '#' || trimmed[0] == ';' {
			out.Write(line)
			continue
		}
		directive := trimmed
		if i := bytes.IndexAny(trimmed, " \t"); i >= 0 {
			directive = trimmed[:i]
		}
		name := strings.ToLower(string(directive))
		if scriptDirectives[name] {
			stripped = append(stripped, name)
			out.WriteString("# vpngate disabled script directive: ")
			out.Write(line)
			continue
		}
		out.Write(line)
	}
	return out.Bytes(), stripped
}

// executablePath returns the platform-specific path to the openvpn
// binary.
func executablePath() string {
	if runtime.GOOS == "windows" {
		hardcoded := `C:\Program Files\OpenVPN\bin\openvpn.exe`
		if _, err := os.Stat(hardcoded); err == nil {
			return hardcoded
		}
		// Custom install location — fall through to PATH lookup.
	}
	return "openvpn"
}

// Connect to a specified OpenVPN configuration. Blocks until openvpn
// exits, streaming its output through pkg/exec's logger.
func Connect(configPath string) error {
	return exec.Run(executablePath(), ".", "--verb", "4", "--config", configPath, "--data-ciphers", dataCiphers)
}

// ConnectDetached starts openvpn with a management interface enabled at
// managementAddr, detached via sysProcAttr so it outlives the calling
// process, writing its combined stdout/stderr to logWriter. It returns as
// soon as the process has started; callers wait on the returned *exec.Cmd
// independently (via cmd.Wait()) to learn when it exits.
func ConnectDetached(configPath, managementAddr string, logWriter io.Writer, sysProcAttr *syscall.SysProcAttr) (*osexec.Cmd, error) {
	executable := executablePath()
	if _, err := osexec.LookPath(executable); err != nil {
		return nil, fmt.Errorf("%s is required, please install it", executable)
	}

	host, port, err := net.SplitHostPort(managementAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid management address %q: %w", managementAddr, err)
	}

	cmd := osexec.Command(
		executable,
		"--verb", "4",
		"--config", configPath,
		"--data-ciphers", dataCiphers,
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
