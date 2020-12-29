package network

import (
	"github.com/juju/errors"
	"github.com/rs/zerolog/log"
	"github.com/showwin/speedtest-go/speedtest"
)

// TestSpeed tests the speed of an active network connection
func TestSpeed() error {
	user, err := speedtest.FetchUserInfo()

	if err != nil {
		return errors.Annotate(err, "Unable to fetch user info")
	}

	serverList, err := speedtest.FetchServerList(user)

	if err != nil {
		return errors.Annotate(err, "Unable to fetch server list")
	}

	targets, _ := serverList.FindServer([]int{})

	if err != nil {
		return errors.Annotate(err, "Unable to find server")
	}

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(true)
		s.UploadTest(true)

		log.Info().Msgf("Latency: %s, Download: %f, Upload: %f", s.Latency, s.DLSpeed, s.ULSpeed)
	}

	return nil
}
