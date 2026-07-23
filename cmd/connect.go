package cmd

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"os"
	osexec "os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/daemon"
	"github.com/davegallant/vpngate/pkg/vpn"
	"github.com/spf13/cobra"
)

var (
	flagRandom         bool
	flagReconnect      bool
	flagProxy          string
	flagSocks5Proxy    string
	flagDaemon         bool
	flagDaemonRun      bool
	flagDaemonHostname string
)

func init() {
	connectCmd.Flags().BoolVarP(&flagRandom, "random", "r", false, "connect to a random server")
	connectCmd.Flags().BoolVarP(&flagReconnect, "reconnect", "t", false, "continually attempt to connect to the server")
	connectCmd.Flags().StringVarP(&flagProxy, "proxy", "p", "", "provide a http/https proxy server to make requests through (i.e. http://127.0.0.1:8080)")
	connectCmd.Flags().StringVarP(&flagSocks5Proxy, "socks5", "s", "", "provide a socks5 proxy server to make requests through (i.e. 127.0.0.1:1080)")
	connectCmd.Flags().StringVar(&flagCountry, "country", "", "filter by country name or country code (i.e. Japan or jp)")
	connectCmd.Flags().IntVar(&flagMaxPing, "max-ping", 0, "filter out servers with ping higher than this value")
	connectCmd.Flags().IntVar(&flagMinScore, "min-score", 0, "filter out servers with score lower than this value")
	connectCmd.Flags().BoolVar(&flagRefresh, "refresh", false, "refresh the vpn server list cache before connecting")
	connectCmd.Flags().BoolVar(&flagNoCache, "no-cache", false, "do not read from or write to the vpn server list cache")
	connectCmd.Flags().BoolVarP(&flagDaemon, "daemon", "d", false, "run the connection in the background; see 'vpngate status' and 'vpngate disconnect'")
	connectCmd.Flags().BoolVar(&flagDaemonRun, "__daemon-run", false, "internal: run as the background daemon supervisor")
	connectCmd.Flags().StringVar(&flagDaemonHostname, "__daemon-hostname", "", "internal: hostname resolved by the foreground process")
	_ = connectCmd.Flags().MarkHidden("__daemon-run")
	_ = connectCmd.Flags().MarkHidden("__daemon-hostname")
	rootCmd.AddCommand(connectCmd)
}

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect to a vpn server (survey selection appears if hostname is not provided)",
	Long:  `Connect to a vpn from a list of relay servers. Because openvpn creates a network interface, run the connect command with 'sudo' or a user with escalated privileges.`,
	Args:  cobra.RangeArgs(0, 1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if flagDaemonRun {
			return runSupervisor()
		}

		vpnServers, err := vpn.GetListWithOptions(flagProxy, flagSocks5Proxy, vpn.ListOptions{Refresh: flagRefresh, NoCache: flagNoCache})
		if err != nil {
			return err
		}

		vpnServers = filterServers(vpnServers)
		if len(*vpnServers) == 0 {
			return fmt.Errorf("no vpn servers matched the provided filters")
		}

		// Build rich server selection options and lookup map.
		serverSelection, serverMap := buildServerSelection(*vpnServers)

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
				if err := survey.AskOne(prompt, &selection, survey.WithPageSize(10)); err != nil {
					return fmt.Errorf("unable to obtain hostname from survey: %w", err)
				}
			}

			// Lookup server from selection using map for O(1) lookup.
			if server, exists := serverMap[selection]; exists {
				serverSelected = server
			} else if server, exists := serverMap[extractHostname(selection)]; exists {
				serverSelected = server
			} else {
				return fmt.Errorf("server %q was not found", selection)
			}
		}

		if flagDaemon {
			return startDaemon(serverSelected)
		}

		for {
			if flagRandom {
				// Select a random server
				serverSelected = (*vpnServers)[rand.Intn(len(*vpnServers))]
			}

			decodedConfig, err := base64.StdEncoding.DecodeString(serverSelected.OpenVpnConfigData)
			if err != nil {
				return err
			}

			tmpfile, err := os.CreateTemp("", "vpngate-openvpn-config-")
			if err != nil {
				return err
			}

			if _, err := tmpfile.Write(decodedConfig); err != nil {
				_ = tmpfile.Close()
				_ = os.Remove(tmpfile.Name())
				return err
			}

			if err := tmpfile.Close(); err != nil {
				_ = os.Remove(tmpfile.Name())
				return err
			}

			log.Info().Msgf("Connecting to %s (%s) in %s", serverSelected.HostName, serverSelected.IPAddr, serverSelected.CountryLong)

			err = vpn.Connect(tmpfile.Name())

			// Always try to clean up temporary file
			_ = os.Remove(tmpfile.Name())

			if !flagReconnect {
				if err != nil {
					return fmt.Errorf("vpn connection failed: %w", err)
				}
				return nil
			}
		}
	},
}

// startDaemon re-execs the current binary detached from the terminal so
// it can run connect in the background, then waits for it to report a
// successful connection. serverSelected is the zero value when --random
// was passed — the daemon resolves its own server in that case, possibly
// reselecting on every reconnect attempt.
func startDaemon(serverSelected vpn.Server) error {
	if state, err := daemon.Load(); err == nil {
		if daemon.IsAlive(state.PID) {
			return fmt.Errorf("already connected to %s (PID %d); run 'vpngate disconnect' first", state.HostName, state.PID)
		}
		_ = daemon.Remove()
	} else if !os.IsNotExist(err) {
		return err
	}

	selfPath, err := os.Executable()
	if err != nil {
		return err
	}

	childArgs := []string{"connect", "--__daemon-run"}
	if !flagRandom {
		childArgs = append(childArgs, "--__daemon-hostname", serverSelected.HostName)
	}
	childArgs = append(childArgs, forwardableConnectArgs()...)

	child := osexec.Command(selfPath, childArgs...)
	child.SysProcAttr = daemon.DetachAttr()

	if err := child.Start(); err != nil {
		return fmt.Errorf("starting background daemon: %w", err)
	}
	if err := child.Process.Release(); err != nil {
		return err
	}

	return waitForDaemonReady(30 * time.Second)
}

// forwardableConnectArgs reproduces the subset of connect's own flags
// that the re-exec'd daemon supervisor needs to repeat the same server
// selection and connection behavior.
func forwardableConnectArgs() []string {
	var args []string
	if flagReconnect {
		args = append(args, "--reconnect")
	}
	if flagRandom {
		args = append(args, "--random")
	}
	if flagProxy != "" {
		args = append(args, "--proxy", flagProxy)
	}
	if flagSocks5Proxy != "" {
		args = append(args, "--socks5", flagSocks5Proxy)
	}
	if flagCountry != "" {
		args = append(args, "--country", flagCountry)
	}
	if flagMaxPing != 0 {
		args = append(args, "--max-ping", strconv.Itoa(flagMaxPing))
	}
	if flagMinScore != 0 {
		args = append(args, "--min-score", strconv.Itoa(flagMinScore))
	}
	if flagRefresh {
		args = append(args, "--refresh")
	}
	if flagNoCache {
		args = append(args, "--no-cache")
	}
	return args
}

// waitForDaemonReady polls for the daemon's state file to appear,
// signalling a successful first connection, surfacing the tail of the
// daemon log if it times out instead.
func waitForDaemonReady(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		state, err := daemon.Load()
		if err == nil {
			fmt.Printf("Connected in background to %s (PID %d)\n", state.HostName, state.PID)
			return nil
		}
		if !os.IsNotExist(err) {
			return err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for background connection; see %s\n%s", daemon.LogPath(), tailLog())
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// tailLog returns the last few lines of the daemon log for error
// messages, or an empty string if it can't be read.
func tailLog() string {
	data, err := os.ReadFile(daemon.LogPath())
	if err != nil {
		return ""
	}
	return strings.TrimRight(lastLines(data, 10), "\n")
}

func buildServerSelection(servers []vpn.Server) ([]string, map[string]vpn.Server) {
	hostnameWidth := len("Hostname")
	countryWidth := len("Country")
	for _, server := range servers {
		if len(server.HostName) > hostnameWidth {
			hostnameWidth = len(server.HostName)
		}
		if len(server.CountryLong) > countryWidth {
			countryWidth = len(server.CountryLong)
		}
	}

	serverSelection := make([]string, len(servers))
	serverMap := make(map[string]vpn.Server, len(servers)*2)
	for i, server := range servers {
		label := formatServerSelection(server, hostnameWidth, countryWidth)
		serverSelection[i] = label
		serverMap[label] = server
		serverMap[server.HostName] = server
	}

	return serverSelection, serverMap
}

func formatServerSelection(server vpn.Server, hostnameWidth int, countryWidth int) string {
	return fmt.Sprintf(
		"%-*s  %-*s  %-15s  ping %s",
		hostnameWidth,
		server.HostName,
		countryWidth,
		server.CountryLong,
		server.IPAddr,
		server.Ping,
	)
}

// extractHostname extracts the hostname from a manually provided argument or legacy selection string.
func extractHostname(selection string) string {
	selection = strings.TrimSpace(selection)

	parts := strings.Split(selection, " | ")
	if len(parts) > 0 {
		selection = strings.TrimSpace(parts[0])
	}

	parts = strings.Split(selection, " (")
	if len(parts) > 0 {
		selection = strings.TrimSpace(parts[0])
	}

	parts = strings.Fields(selection)
	if len(parts) > 0 {
		return parts[0]
	}

	return selection
}
