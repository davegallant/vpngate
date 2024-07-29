package vpn

import (
	"os"
	"runtime"

	"github.com/davegallant/vpngate/pkg/exec"
	"github.com/juju/errors"
	"github.com/nxadm/tail"
	"github.com/rs/zerolog/log"
)

// Connect to a specified OpenVPN configuration
func Connect(configPath string) error {
	tmpLogFile, err := os.CreateTemp("", "vpngate-openvpn-log-")
	if err != nil {
		return errors.Annotate(err, "Unable to create a temporary log file")
	}
	defer os.Remove(tmpLogFile.Name())

	go func() {
		// Tail the temporary openvpn log file
		t, err := tail.TailFile(tmpLogFile.Name(), tail.Config{Follow: true})
		if err != nil {
			log.Error().Msgf("%s", err)
		}
		for line := range t.Lines {
			log.Debug().Msg(line.Text)
		}
	}()

	executable := "openvpn"
	if runtime.GOOS == "windows" {
		executable = "C:\\Program Files\\OpenVPN\\bin\\openvpn.exe"
	}

	_, err = exec.Run(executable, ".", "--verb", "4", "--log", tmpLogFile.Name(), "--config", configPath, "--data-ciphers", "AES-128-CBC")
	return err
}
