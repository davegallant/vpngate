package vpn

import (
	"github.com/davegallant/vpngate/pkg/exec"
)

func Connect(config string) error {
	exec.Run("sudo", ".", "openvpn", "--config", config)
	return nil
}
