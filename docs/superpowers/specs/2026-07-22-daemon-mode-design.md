# Daemon mode (`connect -d`, `status`, `disconnect`)

Implements [#53](https://github.com/davegallant/vpngate/issues/53).

## Goal

Let `vpngate connect` run in the background instead of blocking the
terminal, and add `vpngate status` / `vpngate disconnect` to inspect and
tear down that background connection.

## Scope

- A detached background process (dies on logout/reboot like any other
  process) — not an OS-managed persistent service (no systemd/launchd/
  Windows Service integration, no install/uninstall commands).
- Single active daemon at a time (one VPN interface).
- `status` is plain human-readable text (no `--output json/csv`).
- `-t/--reconnect` continues to work in daemon mode.

## Architecture

### New package `pkg/daemon`

- `State` struct + `Save`/`Load`/`Remove` — JSON file at
  `~/.vpngate/state.json` holding: supervisor PID, management host:port,
  connected server (hostname/IP/country), config file path, log file
  path, started-at timestamp.
- `Management` client — minimal client for openvpn's plaintext management
  protocol over TCP: `Connect()`, `State()` (parses the `>STATE:` line for
  CONNECTING/CONNECTED/RECONNECTING/EXITING), `Disconnect()` (sends
  `signal SIGTERM`).
- `Spawn`/`Detach` helpers — OS-specific process detachment, split into
  `spawn_unix.go` (`SysProcAttr{Setsid: true}`) and `spawn_windows.go`
  (`CreationFlags: CREATE_NEW_PROCESS_GROUP|DETACHED_PROCESS`), matching
  the existing platform-specific pattern in `pkg/vpn/client.go`.

### `cmd/connect.go` changes

- New `-d/--daemon` bool flag.
- New hidden flag `--__daemon-run` (not shown in `--help`, internal-only)
  carrying the already-resolved server hostname so the re-exec'd child
  doesn't re-prompt.
- When `-d` is set: run the existing selection/filtering logic in the
  foreground as today, then re-exec `os.Args[0]` with
  `--__daemon-run <hostname>` plus the original filter/proxy/reconnect
  flags, detached via `pkg/daemon.Spawn`. The parent waits (polling
  `state.json`, ~30s timeout) for confirmation of the first successful
  connect, then prints `Connected in background to <hostname> (PID <pid>)`
  and exits.
- When `--__daemon-run` is set (internal, child process): become the
  supervisor — write the log file, open the management port, run the
  *existing* connect/reconnect loop against openvpn as a normal child
  process, write `state.json` once connected, clean up on exit.

### New `cmd/status.go` and `cmd/disconnect.go`

Thin commands that call into `pkg/daemon`: load state, query/signal the
management socket, print the result (or `Not connected.`), remove state
on disconnect.

## Data flow

### `vpngate connect -d [flags]`

1. Foreground process: fetch + filter server list, resolve selection
   (survey prompt or positional arg) — identical to today.
2. Check `~/.vpngate/state.json`: if it exists and its PID is alive,
   error out (`already connected to <host>, run 'vpngate disconnect'
   first`).
3. Re-exec self detached with `--__daemon-run <hostname>` + the original
   flags (`--reconnect`, `--random`, `--country`, `--proxy`, etc.) so the
   child can reproduce filtering if `--random` needs to reselect on
   reconnect.
4. Child (supervisor): decode config to `~/.vpngate/config.ovpn`
   (persistent, not a one-shot tempfile, so it survives across
   reconnect-loop iterations), pick a free loopback port with
   `net.Listen("tcp", "127.0.0.1:0")` then close it, start
   `openvpn --management 127.0.0.1 <port> --config ... --data-ciphers
   AES-128-CBC` with stdout/stderr redirected to `~/.vpngate/daemon.log`,
   detached so it isn't killed if the parent's process group is
   signaled.
5. Child polls the management socket for `CONNECTED,SUCCESS` (falling
   back to scanning the log for `Initialization Sequence Completed` if
   management parsing misses it), then writes `state.json`.
6. Child enters the existing reconnect loop (reused from current
   `connect.go`): if `openvpn` exits and `--reconnect` was passed,
   restart it (reselecting randomly if `--random`); otherwise clean up
   `state.json`/config/log and exit.
7. Parent (still watching from step 3) sees `state.json` appear and
   prints success, or times out after ~30s and reports failure
   (surfacing the tail of `daemon.log`).

### `vpngate status`

1. Load `state.json`. Missing → print `Not connected.`
2. PID dead → stale file, remove it, print `Not connected.`
3. Otherwise connect to the management port, send `state`, parse the
   state line; print server, state (connected/reconnecting), uptime
   (`now - startedAt`), PID. Socket unreachable but PID alive → print
   `Status: unknown (management socket unreachable)` rather than failing
   hard.

### `vpngate disconnect`

1. Load `state.json`. Missing → print `Not connected.`
2. Send `signal SIGTERM` over the management socket for a clean shutdown
   (lets openvpn tear down the tun interface properly); poll for the PID
   to exit, up to ~5s.
3. If it doesn't exit in time (or the socket's unreachable), fall back to
   killing the PID directly.
4. Remove `state.json`, `config.ovpn`. Print `Disconnected.`

## Error handling & edge cases

- **Double-connect:** `connect -d` while already connected → clear error,
  no orphaned second openvpn process.
- **`status`/`disconnect` with no daemon running:** print
  `Not connected.` and exit 0 (not an error — scripts shouldn't need to
  special-case this).
- **Stale state file** (process crashed/was killed outside vpngate, e.g.
  `kill -9` or reboot): detected via dead PID in both `status` and
  `disconnect`; auto-cleaned so the user isn't stuck manually deleting
  `~/.vpngate/state.json`.
- **openvpn fails to start** (bad config, permissions, port in use):
  supervisor detects early exit before reaching `CONNECTED`, cleans up,
  and the parent's 30s wait surfaces the failure with the tail of
  `daemon.log` instead of a bare timeout message.
- **Requires elevated privileges:** unchanged from today — since the
  supervisor is a re-exec of the same binary, it inherits whatever
  privilege `connect -d` itself was run under (e.g. still needs `sudo`).
- **Concurrent `disconnect` calls / management socket races:** disconnect
  is idempotent — a second call after state is already removed just
  prints `Not connected.`

## Testing plan

- **Unit tests** (no real openvpn/network needed):
  - `pkg/daemon`: state file save/load/remove round-trip;
    management-protocol parsing against a fake TCP server that emits
    canned `>STATE:` lines, including malformed/partial responses.
  - `cmd/status`, `cmd/disconnect`: behavior against a fake state file +
    fake management server (not-connected, connected, stale-PID,
    socket-unreachable cases).
- **Manual verification** (real VPN connections can't run in
  CI/sandbox): `connect -d`, `status`, `disconnect` exercised by hand on
  macOS/Linux with real openvpn. The Windows detach path is reviewed
  carefully but can't be verified in this environment (no Windows box) —
  called out explicitly as unverified in the PR.
