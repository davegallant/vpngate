package vpn

import (
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/davegallant/vpngate/pkg/network"
	"github.com/juju/errors"
	"github.com/rs/zerolog/log"
)

func Connect(config string) error {
	path := "openvpn"
	_, err := exec.LookPath(path)
	if err != nil {
		log.Error().Msgf("%s required, please install it and ensure that it is within your PATH", path)
		os.Exit(1)
	}
	cmd := exec.Command("openvpn", "--script-security", "2", "--config", config)
	log.Debug().Msgf("Executing " + strings.Join(cmd.Args, " "))
	err = cmd.Start()
	if err != nil {
		return errors.Annotatef(err, "%s %s", path, cmd.Args)
	}

	for {
		network.TestSpeed()
		time.Sleep(time.Minute)
	}

}
