package cmd

import (
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect",
	Long:  `Connect to a vpn from a list of servers`,
	Run: func(cmd *cobra.Command, args []string) {

		vpnServers, err := vpn.GetList()

		if err != nil {
			log.Fatal()
		}

		for _, s := range *vpnServers {
			log.Info().Msgf("%s, %s", s.HostName, s.CountryLong)
		}

		vpn.Connect("fakeConfig")

	},
}
