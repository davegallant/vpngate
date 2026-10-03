package cmd

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/davegallant/vpngate/pkg/daemon"
	"github.com/davegallant/vpngate/pkg/util"
	"github.com/davegallant/vpngate/pkg/vpn"
)

// supervisor owns a single daemon's lifecycle: it launches openvpn,
// tracks the currently-active management connection, answers control
// requests from separate `status`/`disconnect` invocations, and — when
// --reconnect is set — restarts openvpn if it exits on its own.
type supervisor struct {
	vpnServers []vpn.Server
	random     bool
	reconnect  bool
	logFile    *os.File
	control    *daemon.ControlServer
	// wake interrupts the backoff sleep between reconnect attempts so
	// handleStop is prompt even when no openvpn process is running.
	wake chan struct{}

	mu        sync.Mutex
	server    vpn.Server
	startedAt time.Time
	mgmt      *daemon.Management
	stopping  bool
}

// runSupervisor is the entry point used when connect is re-exec'd with
// --__daemon-run: it resolves the server to connect to, opens the
// control socket, and runs the connect/reconnect loop until told to
// stop.
func runSupervisor() error {
	vpnServers, err := vpn.GetListWithOptions(flagProxy, flagSocks5Proxy, vpn.ListOptions{Refresh: flagRefresh, NoCache: flagNoCache})
	if err != nil {
		return err
	}
	filtered := *filterServers(vpnServers)
	if len(filtered) == 0 {
		return fmt.Errorf("no vpn servers matched the provided filters")
	}

	var initial vpn.Server
	if flagRandom {
		initial = filtered[rand.IntN(len(filtered))]
	} else {
		found := false
		for _, s := range filtered {
			if s.HostName == flagDaemonHostname {
				initial = s
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("server %q was not found", flagDaemonHostname)
		}
	}

	if err := os.MkdirAll(daemon.Dir(), 0o700); err != nil {
		return err
	}
	logFile, err := os.OpenFile(daemon.LogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() { _ = logFile.Close() }()

	// The supervisor is a detached, re-exec'd child: nothing connects its
	// stdout/stderr back to the terminal the user is watching, so the
	// default console logger would silently discard everything (e.g. an
	// "openvpn is required" failure before openvpn ever starts, which
	// never reaches daemon.log otherwise). Redirect it to daemon.log,
	// the one place the foreground process points users at on failure.
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: logFile, NoColor: true})

	controlLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	s := &supervisor{
		vpnServers: filtered,
		random:     flagRandom,
		reconnect:  flagReconnect,
		logFile:    logFile,
		server:     initial,
		wake:       make(chan struct{}, 1),
	}
	s.control = daemon.NewControlServer(controlLn, s.handleStatus, s.handleStop)
	go s.control.Serve()

	// Record the PID before the first attempt so the foreground process
	// can tell "daemon died during startup" apart from "daemon still
	// negotiating", and refuse to start a second daemon meanwhile.
	if err := daemon.WritePid(os.Getpid()); err != nil {
		return err
	}

	return s.run()
}

func (s *supervisor) run() error {
	defer func() {
		_ = daemon.Remove()
		_ = daemon.RemovePid()
		_ = os.Remove(daemon.ConfigPath())
	}()

	backoff := util.NewBackoff(2*time.Second, time.Minute)

	for {
		s.mu.Lock()
		if s.stopping {
			s.mu.Unlock()
			return nil
		}
		if s.random {
			s.server = s.vpnServers[rand.IntN(len(s.vpnServers))]
		}
		server := s.server
		s.mu.Unlock()

		started := time.Now()
		err := s.connectOnce(server)
		if err != nil {
			log.Error().Err(err).Msg("daemon connection attempt failed")
			if !s.reconnect {
				return err
			}
		}

		s.mu.Lock()
		stopping := s.stopping
		s.mu.Unlock()
		if stopping || !s.reconnect {
			return nil
		}

		// Back off before retrying so a dead server doesn't spin the
		// loop. A connection that lived a while resets the sequence.
		if time.Since(started) > 30*time.Second {
			backoff.Reset()
		}
		select {
		case <-time.After(backoff.Next()):
		case <-s.wake:
			// disconnect() arrived during backoff — loop around so
			// the stopping check above exits promptly.
		}
	}
}

// connectOnce starts openvpn for server, waits for it to report a
// successful connection, records daemon state, and blocks until it
// exits (either on its own or because handleStop signaled it).
func (s *supervisor) connectOnce(server vpn.Server) error {
	decoded, err := base64.StdEncoding.DecodeString(server.OpenVpnConfigData)
	if err != nil {
		return err
	}

	// VPNGate configs come from volunteer-run servers: strip any
	// script/plugin directives before handing the file to openvpn.
	config, stripped := vpn.SanitizeConfig(decoded)
	if len(stripped) > 0 {
		log.Warn().Msgf("disabled script directives from server config: %s", strings.Join(stripped, ", "))
	}
	if err := os.WriteFile(daemon.ConfigPath(), config, 0o600); err != nil {
		return err
	}

	mgmtAddr, err := reserveLoopbackAddr()
	if err != nil {
		return err
	}

	cmd, err := vpn.ConnectDetached(daemon.ConfigPath(), mgmtAddr, s.logFile, daemon.DetachAttr())
	if err != nil {
		return fmt.Errorf("starting openvpn: %w", err)
	}

	mgmt, err := waitForManagement(mgmtAddr, 30*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	// The state file (and the foreground's "Connected" message) should
	// mean the tunnel is up — not just that the management interface
	// answered.
	if err := waitForConnectedState(mgmt, 30*time.Second); err != nil {
		_ = mgmt.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	startedAt := time.Now()
	s.mu.Lock()
	if s.stopping {
		// disconnect() arrived while we were still connecting — undo
		// this attempt instead of publishing state for a connection
		// nobody asked for.
		s.mu.Unlock()
		_ = mgmt.Disconnect()
		_ = mgmt.Close()
		_ = cmd.Wait()
		return nil
	}
	s.server = server
	s.startedAt = startedAt
	s.mgmt = mgmt
	s.mu.Unlock()

	if err := daemon.Save(daemon.State{
		PID:         os.Getpid(),
		OpenVPNPID:  cmd.Process.Pid,
		ControlAddr: s.control.Addr(),
		HostName:    server.HostName,
		IPAddr:      server.IPAddr,
		CountryLong: server.CountryLong,
		StartedAt:   startedAt,
	}); err != nil {
		_ = mgmt.Close()
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return err
	}

	log.Info().Msgf("Connected in background to %s (%s) in %s", server.HostName, server.IPAddr, server.CountryLong)

	waitErr := cmd.Wait()

	s.mu.Lock()
	if s.mgmt != nil {
		_ = s.mgmt.Close()
		s.mgmt = nil
	}
	s.mu.Unlock()

	return waitErr
}

// reserveLoopbackAddr picks a free loopback TCP port by opening then
// immediately closing a listener, so the caller can hand the address to
// a separate process (openvpn) rather than a listener it can't pass on.
func reserveLoopbackAddr() (string, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		return "", err
	}
	return addr, nil
}

// waitForManagement polls addr until OpenVPN's management interface
// accepts a connection, or timeout elapses.
func waitForManagement(addr string, timeout time.Duration) (*daemon.Management, error) {
	deadline := time.Now().Add(timeout)
	for {
		mgmt, err := daemon.DialManagement(addr, time.Second)
		if err == nil {
			return mgmt, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timed out waiting for openvpn management interface: %w", err)
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// waitForConnectedState polls the management interface until openvpn
// reports CONNECTED, fails fast if openvpn starts exiting, or timeout
// elapses.
func waitForConnectedState(mgmt *daemon.Management, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		state, err := mgmt.State()
		if err != nil {
			return fmt.Errorf("reading openvpn connection state: %w", err)
		}
		switch state {
		case "CONNECTED":
			return nil
		case "EXITING":
			return fmt.Errorf("openvpn exited while connecting")
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for openvpn to report CONNECTED (last state %q)", state)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// handleStatus answers a STATUS control request with the supervisor's
// current view of the connection.
func (s *supervisor) handleStatus() (daemon.Snapshot, error) {
	s.mu.Lock()
	mgmt := s.mgmt
	server := s.server
	startedAt := s.startedAt
	s.mu.Unlock()

	state := "CONNECTING"
	if mgmt != nil {
		if st, err := mgmt.State(); err == nil {
			state = st
		}
	}

	return daemon.Snapshot{
		State:       state,
		HostName:    server.HostName,
		IPAddr:      server.IPAddr,
		CountryLong: server.CountryLong,
		StartedAt:   startedAt,
		PID:         os.Getpid(),
	}, nil
}

// handleStop answers a STOP control request: it marks the supervisor as
// stopping (so the reconnect loop in run() won't respawn openvpn) and,
// if openvpn is currently connected, asks it to exit cleanly. run()
// observes s.stopping either when connectOnce's cmd.Wait() returns or,
// if no openvpn is running yet, on its next loop iteration.
func (s *supervisor) handleStop() {
	s.mu.Lock()
	s.stopping = true
	mgmt := s.mgmt
	s.mu.Unlock()

	// Interrupt a backoff sleep between attempts so disconnect stays
	// prompt even when no openvpn process is currently running.
	select {
	case s.wake <- struct{}{}:
	default:
	}

	if mgmt != nil {
		_ = mgmt.Disconnect()
	}
}
