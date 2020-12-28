package network

import (
	"github.com/rs/zerolog/log"
	"github.com/showwin/speedtest-go/speedtest"
)

func TestSpeed() {
	user, _ := speedtest.FetchUserInfo()

	serverList, _ := speedtest.FetchServerList(user)
	targets, _ := serverList.FindServer([]int{})

	for _, s := range targets {
		s.PingTest()
		s.DownloadTest(true)
		s.UploadTest(true)

		log.Info().Msgf("Latency: %s, Download: %f, Upload: %f", s.Latency, s.DLSpeed, s.ULSpeed)
	}

}
