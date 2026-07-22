package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/davegallant/vpngate/pkg/daemon"
)

func init() {
	rootCmd.AddCommand(statusCmd)
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of a background vpn connection started with 'connect -d'",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := daemon.Load()
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("Not connected.")
				return nil
			}
			// daemon state is root-owned (openvpn itself requires root),
			// so a non-root invocation hits a permission error here
			// rather than "not exist" — report it plainly instead of a
			// raw "permission denied" error.
			if os.IsPermission(err) {
				fmt.Println("Not connected, or insufficient permissions to check (try with sudo).")
				return nil
			}
			return err
		}

		if !daemon.IsAlive(state.PID) {
			_ = daemon.Remove()
			fmt.Println("Not connected.")
			return nil
		}

		snap, err := daemon.SendStatus(state.ControlAddr, 5*time.Second)
		if err != nil {
			fmt.Printf("Status:  unknown (control socket unreachable: %v)\n", err)
			fmt.Printf("Server:  %s (%s) - %s\n", state.HostName, state.IPAddr, state.CountryLong)
			fmt.Printf("PID:     %d\n", state.PID)
			return nil
		}

		fmt.Printf("Status:  %s\n", snap.State)
		fmt.Printf("Server:  %s (%s) - %s\n", snap.HostName, snap.IPAddr, snap.CountryLong)
		if !snap.StartedAt.IsZero() {
			fmt.Printf("Uptime:  %s\n", time.Since(snap.StartedAt).Round(time.Second))
		}
		fmt.Printf("PID:     %d\n", snap.PID)
		return nil
	},
}
