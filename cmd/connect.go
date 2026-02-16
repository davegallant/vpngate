package cmd

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/spf13/cobra"
)

var (
	flagRandom      bool
	flagReconnect   bool
	flagProxy       string
	flagSocks5Proxy string
)

func init() {
	connectCmd.Flags().BoolVarP(&flagRandom, "random", "r", false, "connect to a random server")
	connectCmd.Flags().BoolVarP(&flagReconnect, "reconnect", "t", false, "continually attempt to connect to the server")
	connectCmd.Flags().StringVarP(&flagProxy, "proxy", "p", "", "provide a http/https proxy server to make requests through (i.e. http://127.0.0.1:8080)")
	connectCmd.Flags().StringVarP(&flagSocks5Proxy, "socks5", "s", "", "provide a socks5 proxy server to make requests through (i.e. 127.0.0.1:1080)")
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a vpn server (survey selection appears if hostname is not provided)",
	Long:  `Connect to a vpn from a list of relay servers`,
	Args:  cobra.RangeArgs(0, 1),
	Run: func(cmd *cobra.Command, args []string) {
		vpnServers, err := vpn.GetList(flagProxy, flagSocks5Proxy)
		if err != nil {
			log.Fatal().Msg(err.Error())
		}

		// Build server selection options and hostname lookup map
		serverSelection := make([]string, len(*vpnServers))
		serverMap := make(map[string]vpn.Server, len(*vpnServers))
		for i, s := range *vpnServers {
			serverSelection[i] = fmt.Sprintf("%s (%s)", s.HostName, s.CountryLong)
			serverMap[s.HostName] = s
		}

		selection := ""
		var serverSelected vpn.Server

		if !flagRandom {
			if len(args) > 0 {
				selection = args[0]
			} else {
				prompt := &survey.Select{
					Message: "Choose a server:",
					Options: serverSelection,
				}
				err := survey.AskOne(prompt, &selection, survey.WithPageSize(10))
				if err != nil {
					log.Fatal().Msg("Unable to obtain hostname from survey")
				}
			}

			// Lookup server from selection using map for O(1) lookup
			hostname := extractHostname(selection)
			if server, exists := serverMap[hostname]; exists {
				serverSelected = server
			} else {
				log.Fatal().Msgf("Server '%s' was not found", selection)
			}
		}

		for {
			if flagRandom {
				// Select a random server
				serverSelected = (*vpnServers)[rand.Intn(len(*vpnServers))]
			}

			decodedConfig, err := base64.StdEncoding.DecodeString(serverSelected.OpenVpnConfigData)
			if err != nil {
				log.Fatal().Msg(err.Error())
			}

			tmpfile, err := os.CreateTemp("", "vpngate-openvpn-config-")
			if err != nil {
				log.Fatal().Msg(err.Error())
			}

			if _, err := tmpfile.Write(decodedConfig); err != nil {
				log.Fatal().Msg(err.Error())
			}

			if err := tmpfile.Close(); err != nil {
				log.Fatal().Msg(err.Error())
			}

			log.Info().Msgf("Connecting to %s (%s) in %s", serverSelected.HostName, serverSelected.IPAddr, serverSelected.CountryLong)

			err = vpn.Connect(tmpfile.Name())

			if err != nil && !flagReconnect {
				// VPN connection failed and reconnect is disabled
				_ = os.Remove(tmpfile.Name())
				log.Fatal().Msg("VPN connection failed")
			}

			// Always try to clean up temporary file
			_ = os.Remove(tmpfile.Name())
		}
	},
}

// extractHostname extracts the hostname from the selection string (format: "hostname (country)")
func extractHostname(selection string) string {
	parts := strings.Split(selection, " (")
	if len(parts) > 0 {
		return parts[0]
	}
	return selection
}