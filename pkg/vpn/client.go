package vpn

import (
	"os"
	"runtime"

	"github.com/davegallant/vpngate/pkg/exec"
	"github.com/juju/errors"
)

// Connect to a specified OpenVPN configuration
func Connect(configPath string) error {
	tmpLogFile, err := os.CreateTemp("", "vpngate-openvpn-log-")
	if err != nil {
		return errors.Annotate(err, "Unable to create a temporary log file")
	}
	defer os.Remove(tmpLogFile.Name())

	executable := "openvpn"
	if runtime.GOOS == "windows" {
		executable = "C:\\Program Files\\OpenVPN\\bin\\openvpn.exe"
	}

	err = exec.Run(executable, ".", "--verb", "4", "--config", configPath, "--data-ciphers", "AES-128-CBC")
	return err
}
