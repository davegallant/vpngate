package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

const version = "0.4.0"

var rootCmd = &cobra.Command{
	Use:     "vpngate",
	Short:   "vpngate is a client for vpngate.net",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			_ = cmd.Help()
			os.Exit(0)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
