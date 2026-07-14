package cmd

import (
	"os"
	"strconv"
	"strings"

	tw "github.com/olekukonko/tablewriter"

	"github.com/davegallant/vpngate/pkg/vpn"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().StringVarP(&flagProxy, "proxy", "p", "", "provide a http/https proxy server to make requests through (i.e. http://127.0.0.1:8080)")
	listCmd.Flags().StringVarP(&flagSocks5Proxy, "socks5", "s", "", "provide a socks5 proxy server to make requests through (i.e. 127.0.0.1:1080)")
	listCmd.Flags().StringVar(&flagCountry, "country", "", "filter by country name or country code (i.e. Japan or jp)")
	listCmd.Flags().IntVar(&flagMaxPing, "max-ping", 0, "filter out servers with ping higher than this value")
	listCmd.Flags().IntVar(&flagMinScore, "min-score", 0, "filter out servers with score lower than this value")
	listCmd.Flags().StringVar(&flagSort, "sort", "none", "sort by one of none, score, ping, country, hostname")
	listCmd.Flags().StringVarP(&flagOutput, "output", "o", outputTable, "output format: table, json, csv")
	listCmd.Flags().BoolVar(&flagRefresh, "refresh", false, "refresh the vpn server list cache before listing")
	listCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "do not read from or write to the vpn server list cache")
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available vpn servers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := validateSortFlag(); err != nil {
			return err
		}
		if err := validateOutputFlag(); err != nil {
			return err
		}

		vpnServers, err := vpn.GetListWithOptions(flagProxy, flagSocks5Proxy, vpn.ListOptions{Refresh: flagRefresh, NoCache: flagNoCache})
		if err != nil {
			return err
		}

		vpnServers = filterServers(vpnServers)
		sortServers(vpnServers)

		switch strings.ToLower(flagOutput) {
		case outputJSON:
			return writeServersJSON(vpnServers)
		case outputCSV:
			return writeServersCSV(vpnServers)
		}

		table := tw.NewWriter(os.Stdout)
		table.Header([]string{"#", "HostName", "Country", "Ping", "Score"})

		for i, v := range *vpnServers {
			if err := table.Append([]string{strconv.Itoa(i + 1), v.HostName, v.CountryLong, v.Ping, strconv.Itoa(v.Score)}); err != nil {
				return err
			}
		}
		return table.Render()
	},
}
