package cmd

import (
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/davegallant/vpngate/pkg/vpn"
)

func init() {
	rootCmd.AddCommand(cacheCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cachePathCmd)
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage cached vpn server data",
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear cached vpn server data",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if err := vpn.ClearCache(); err != nil {
			log.Fatal().Msg(err.Error())
		}
		log.Info().Msg("Cleared vpngate cache")
	},
}

var cachePathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the cache directory path",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		cacheDir, err := vpn.CacheDir()
		if err != nil {
			log.Fatal().Msg(err.Error())
		}
		fmt.Println(cacheDir)
	},
}
