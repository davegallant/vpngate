package vpn

import (
	"github.com/davegallant/vpngate/pkg/exec"
)

func Connect(config string) error {
	exec.Run("openvpn", ".", "--script-security", "2", "--config", config)
	return nil
}
