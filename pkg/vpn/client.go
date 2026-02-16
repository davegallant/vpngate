package vpn

import (
	"runtime"

	"github.com/davegallant/vpngate/pkg/exec"
)

// Connect to a specified OpenVPN configuration
func Connect(configPath string) error {
	executable := "openvpn"
	if runtime.GOOS == "windows" {
		executable = "C:\\Program Files\\OpenVPN\\bin\\openvpn.exe"
	}

	return exec.Run(executable, ".", "--verb", "4", "--config", configPath, "--data-ciphers", "AES-128-CBC")
}