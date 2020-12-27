package cmd

import (
	"fmt"

	"github.com/AlecAivazis/survey/v2"
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

		serversAvailable := []string{}

		for _, s := range *vpnServers {
			serversAvailable = append(serversAvailable, fmt.Sprintf("%s (%s) (%dms)", s.HostName, s.CountryShort, s.Ping))
		}

		serverSelected := ""
		prompt := &survey.Select{
			Message: "Choose a server:",
			Options: serversAvailable,
		}
		survey.AskOne(prompt, &serverSelected, survey.WithPageSize(10))

		vpn.Connect("/home/dave/Downloads/openvpn/vpngate_public-vpn-227.opengw.net_tcp_443.ovpn")

	},
}
