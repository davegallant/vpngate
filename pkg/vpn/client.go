package vpn

import (
	"time"

	"github.com/davegallant/vpngate/pkg/exec"
	"github.com/davegallant/vpngate/pkg/network"
)

// Connect to a specified OpenVPN configuration
func Connect(configPath string) error {
	go func() {
		for {
			network.TestSpeed()
			time.Sleep(time.Minute)
		}

	}()
	_, err := exec.Run("openvpn", ".", "--verb", "4", "--config", configPath)
	return err
}
