package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/davegallant/vpngate/pkg/vpn"
)

const (
	outputTable = "table"
	outputJSON  = "json"
	outputCSV   = "csv"
)

var (
	flagCountry  string
	flagMaxPing  int
	flagMinScore int
	flagSort     string
	flagOutput   string
	flagRefresh  bool
	flagNoCache  bool
)

func filterServers(servers *[]vpn.Server) *[]vpn.Server {
	filtered := make([]vpn.Server, 0, len(*servers))

	for _, server := range *servers {
		if flagMinScore > 0 && server.Score < flagMinScore {
			continue
		}

		if flagMaxPing > 0 {
			ping, err := strconv.Atoi(server.Ping)
			if err != nil || ping > flagMaxPing {
				continue
			}
		}

		filtered = append(filtered, server)
	}

	if country := strings.ToLower(flagCountry); country != "" {
		// Prefer exact matches on the country code or full name:
		// substring matching makes "--country us" match "russia".
		if exact := filterByCountryExact(filtered, country); len(exact) > 0 {
			return &exact
		}
		sub := filterByCountrySubstring(filtered, country)
		return &sub
	}

	return &filtered
}

func filterByCountryExact(servers []vpn.Server, country string) []vpn.Server {
	matched := make([]vpn.Server, 0)
	for _, server := range servers {
		if strings.ToLower(server.CountryShort) == country || strings.ToLower(server.CountryLong) == country {
			matched = append(matched, server)
		}
	}
	return matched
}

func filterByCountrySubstring(servers []vpn.Server, country string) []vpn.Server {
	matched := make([]vpn.Server, 0)
	for _, server := range servers {
		if strings.Contains(strings.ToLower(server.CountryLong), country) {
			matched = append(matched, server)
		}
	}
	return matched
}

func sortServers(servers *[]vpn.Server) {
	switch strings.ToLower(flagSort) {
	case "", "none":
		return
	case "score":
		sort.SliceStable(*servers, func(i, j int) bool {
			return (*servers)[i].Score > (*servers)[j].Score
		})
	case "ping":
		sort.SliceStable(*servers, func(i, j int) bool {
			return pingSortValue((*servers)[i].Ping) < pingSortValue((*servers)[j].Ping)
		})
	case "country":
		sort.SliceStable(*servers, func(i, j int) bool {
			return (*servers)[i].CountryLong < (*servers)[j].CountryLong
		})
	case "hostname":
		sort.SliceStable(*servers, func(i, j int) bool {
			return (*servers)[i].HostName < (*servers)[j].HostName
		})
	}
}

func pingSortValue(ping string) int {
	value, err := strconv.Atoi(ping)
	if err != nil {
		return int(^uint(0) >> 1)
	}
	return value
}

func writeServersJSON(servers *[]vpn.Server) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(servers)
}

func writeServersCSV(servers *[]vpn.Server) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	if err := writer.Write([]string{"HostName", "CountryLong", "CountryShort", "IP", "Ping", "Score"}); err != nil {
		return err
	}

	for _, server := range *servers {
		if err := writer.Write([]string{server.HostName, server.CountryLong, server.CountryShort, server.IPAddr, server.Ping, strconv.Itoa(server.Score)}); err != nil {
			return err
		}
	}

	return writer.Error()
}

func validateSortFlag() error {
	switch strings.ToLower(flagSort) {
	case "", "none", "score", "ping", "country", "hostname":
		return nil
	default:
		return fmt.Errorf("invalid sort %q: must be one of none, score, ping, country, hostname", flagSort)
	}
}

func validateOutputFlag() error {
	switch strings.ToLower(flagOutput) {
	case outputTable, outputJSON, outputCSV:
		return nil
	default:
		return fmt.Errorf("invalid output %q: must be one of table, json, csv", flagOutput)
	}
}
