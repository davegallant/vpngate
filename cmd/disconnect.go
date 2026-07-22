package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/davegallant/vpngate/pkg/daemon"
)

func init() {
	rootCmd.AddCommand(disconnectCmd)
}

var disconnectCmd = &cobra.Command{
	Use:   "disconnect",
	Short: "Disconnect a background vpn connection started with 'connect -d'",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := daemon.Load()
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not connected.")
				return nil
			}
			return err
		}

		if !daemon.IsAlive(state.PID) {
			_ = daemon.Remove()
			fmt.Println("Not connected.")
			return nil
		}

		if err := daemon.SendStop(state.ControlAddr, 5*time.Second); err != nil {
			// Control socket unreachable (e.g. the supervisor crashed):
			// fall back to killing it directly and cleaning up ourselves.
			if proc, ferr := os.FindProcess(state.PID); ferr == nil {
				_ = proc.Kill()
			}
			_ = daemon.Remove()
		}

		fmt.Println("Disconnected.")
		return nil
	},
}
