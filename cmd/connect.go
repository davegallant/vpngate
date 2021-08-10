package cmd

import (
	"encoding/base64"
	"time"

	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/spf13/cobra"
)

var flagRandom bool
var flagReconnect bool

func init() {
	connectCmd.Flags().BoolVarP(&flagRandom, "random", "r", false, "connect to a random server")
	connectCmd.Flags().BoolVarP(&flagReconnect, "reconnect", "t", false, "continually attempt to connect to the server")
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a vpn server (survey selection appears if hostname is not provided)",
	Long:  `Connect to a vpn from a list of relay servers`,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {

		vpnServers, err := vpn.GetList()

		if err != nil {
			log.Fatal().Msgf(err.Error())
			os.Exit(1)
		}

		serverSelection := []string{}
		serverSelected := vpn.Server{}

		for _, s := range *vpnServers {
			serverSelection = append(serverSelection, fmt.Sprintf("%s (%s)", s.HostName, s.CountryLong))
		}

		selection := ""
		prompt := &survey.Select{
			Message: "Choose a server:",
			Options: serverSelection,
		}

		if flagRandom {
			// Select a random server
			rand.Seed(time.Now().UnixNano())
			serverSelected = (*vpnServers)[rand.Intn(len(*vpnServers))]
		} else {

			if len(args) > 0 {
				selection = args[0]
			} else {
				err := survey.AskOne(prompt, &selection, survey.WithPageSize(10))
				if err != nil {
					log.Error().Msg("Unable to obtain hostname from survey")
					os.Exit(1)
				}
			}

			// Server lookup from selection could be more optimized with a hash map
			for _, s := range *vpnServers {
				if strings.Contains(selection, s.HostName) {
					serverSelected = s
				}
			}

			if serverSelected.HostName == "" {
				log.Fatal().Msgf("Server '%s' was not found", selection)
				os.Exit(1)
			}
		}

		decodedConfig, err := base64.StdEncoding.DecodeString(serverSelected.OpenVpnConfigData)
		if err != nil {
			log.Fatal().Msgf(err.Error())
			os.Exit(1)
		}

		tmpfile, err := ioutil.TempFile("", "vpngate-openvpn-config-")
		if err != nil {
			log.Fatal().Msgf(err.Error())
			os.Exit(1)
		}

		defer os.Remove(tmpfile.Name())

		if _, err := tmpfile.Write(decodedConfig); err != nil {
			log.Fatal().Msgf(err.Error())
			os.Exit(1)
		}

		if err := tmpfile.Close(); err != nil {
			log.Fatal().Msgf(err.Error())
			os.Exit(1)
		}

		for {

			log.Info().Msgf("Connecting to %s (%s) in %s", serverSelected.HostName, serverSelected.IPAddr, serverSelected.CountryLong)

			err = vpn.Connect(tmpfile.Name())

			if err != nil && !flagReconnect {
				log.Fatal().Msgf(err.Error())
				os.Exit(1)
			}
		}

	},
}
