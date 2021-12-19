package cmd

import (
	"os"
	"strconv"

	"github.com/olekukonko/tablewriter"
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available vpn servers",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		vpnServers, err := vpn.GetList()
		if err != nil {
			log.Fatal().Msgf(err.Error())
			os.Exit(1)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"#", "HostName", "Country", "Ping", "Score"})

		for i, v := range *vpnServers {
			table.Append([]string{strconv.Itoa(i + 1), v.HostName, v.CountryLong, v.Ping, strconv.Itoa(v.Score)})
		}
		table.Render() // Send output
	},
}
