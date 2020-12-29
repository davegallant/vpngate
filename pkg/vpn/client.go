package vpn

import (
	"time"

	"github.com/davegallant/vpngate/pkg/exec"
	"github.com/davegallant/vpngate/pkg/network"
	"github.com/rs/zerolog/log"
)

func Connect(config string) error {
	go func() {
		log.Info().Msg("Starting speed tests")
		for {
			network.TestSpeed()
			time.Sleep(time.Minute)
		}

	}()
	_, err := exec.Run("openvpn", ".", "--config", config)
	return err
}
