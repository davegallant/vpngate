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
  `os.TempDir()/vpngate/state.json` (not `~/.vpngate/`: `$HOME` is
  unreliable under `sudo`, which would make `status`/`disconnect` look in
  the wrong place when run without `sudo`; `os.TempDir()` is stable
  across that and matches the already-agreed "dies on reboot" scope).
  Holds: supervisor PID, **control** host:port (see below), connected
  server (hostname/IP/country), config file path, log file path,
  started-at timestamp.
- `Management` client — minimal client for openvpn's plaintext management
  protocol over TCP: `Connect()`, `State()` (parses the `>STATE:` line for
  CONNECTING/CONNECTED/RECONNECTING/EXITING), `Disconnect()` (sends
  `signal SIGTERM`). **Used only internally by the supervisor** — it is
  the sole client of openvpn's management socket; external commands never
  connect to it directly.
- `Control` server/client — a second, separate loopback TCP socket that
  the *supervisor itself* listens on (not openvpn). Speaks a tiny
  line protocol: `STATUS` (supervisor replies with current state, server
  info, uptime by querying its own `Management` client) and `STOP`
  (supervisor sets an internal "stopping" flag, tells openvpn to exit via
  `Management.Disconnect()`, waits for it to exit, removes state, then
  exits itself). This exists because signaling openvpn's management
  socket directly can't distinguish "the user asked to disconnect" from
  "openvpn died and the `--reconnect` loop should respawn it" — only the
  supervisor knows which one is happening. It also sidesteps Windows not
  supporting arbitrary-process `SIGTERM`: `status`/`disconnect` always
  talk over TCP to the control port, never signal a PID directly.
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

Thin commands that call into `pkg/daemon`: load state, send `STATUS` or
`STOP` to the supervisor's control socket, print the result (or
`Not connected.`).

## Data flow

### `vpngate connect -d [flags]`

1. Foreground process: fetch + filter server list, resolve selection
   (survey prompt or positional arg) — identical to today.
2. Check `os.TempDir()/vpngate/state.json`: if it exists and its PID is
   alive, error out (`already connected to <host>, run 'vpngate
   disconnect' first`).
3. Re-exec self detached with `--__daemon-run <hostname>` + the original
   flags (`--reconnect`, `--random`, `--country`, `--proxy`, etc.) so the
   child can reproduce filtering if `--random` needs to reselect on
   reconnect.
4. Child (supervisor): decode config to
   `os.TempDir()/vpngate/config.ovpn` (persistent, not a one-shot
   tempfile, so it survives across reconnect-loop iterations). Open two
   loopback listeners: the **control** socket (`net.Listen("tcp",
   "127.0.0.1:0")`, kept open and served for the supervisor's whole
   lifetime) and, per openvpn instance, a fresh **management** port
   (opened-then-closed the same way to reserve a free port, then passed
   to openvpn) — start `openvpn --management 127.0.0.1 <port> --config
   ... --data-ciphers AES-128-CBC` with stdout/stderr redirected to
   `os.TempDir()/vpngate/daemon.log`, detached so it isn't killed if the
   parent's process group is signaled.
5. Child polls its `Management` client for `CONNECTED,SUCCESS` (falling
   back to scanning the log for `Initialization Sequence Completed` if
   management parsing misses it), then writes `state.json` (recording the
   **control** address, not the per-instance management address).
6. Child enters the existing reconnect loop (reused from current
   `connect.go`): if `openvpn` exits and `--reconnect` was passed *and*
   the supervisor wasn't told to stop, restart it on a newly-reserved
   management port (reselecting randomly if `--random`); otherwise clean
   up `state.json`/config/log and exit. The control listener from step 4
   stays up across every iteration of this loop.
7. Parent (still watching from step 3) sees `state.json` appear and
   prints success, or times out after ~30s and reports failure
   (surfacing the tail of `daemon.log`).

### `vpngate status`

1. Load `state.json`. Missing → print `Not connected.`
2. PID dead → stale file, remove it, print `Not connected.`
3. Otherwise dial the control address and send `STATUS`; the supervisor
   replies with its current state (queried from its own `Management`
   client), server info, and uptime (`now - startedAt`). Print those.
   Control socket unreachable but PID alive → print `Status: unknown
   (control socket unreachable)` rather than failing hard.

### `vpngate disconnect`

1. Load `state.json`. Missing → print `Not connected.`
2. Dial the control address and send `STOP`. The supervisor sets its
   "stopping" flag (so the reconnect loop won't respawn), signals openvpn
   to exit via its `Management` client, waits for it to exit, removes
   `state.json`/`config.ovpn`, and exits itself — `disconnect` waits for
   that reply (or the socket closing) up to ~5s.
3. If the control socket is unreachable (e.g. supervisor crashed
   leaving a stale PID), fall back to killing the PID directly and
   removing `state.json`/`config.ovpn` from the `disconnect` side.
4. Print `Disconnected.`

## Error handling & edge cases

- **Double-connect:** `connect -d` while already connected → clear error,
  no orphaned second openvpn process.
- **`status`/`disconnect` with no daemon running:** print
  `Not connected.` and exit 0 (not an error — scripts shouldn't need to
  special-case this).
- **Stale state file** (process crashed/was killed outside vpngate, e.g.
  `kill -9` or reboot): detected via dead PID in both `status` and
  `disconnect`; auto-cleaned so the user isn't stuck manually deleting
  `os.TempDir()/vpngate/state.json`.
- **openvpn fails to start** (bad config, permissions, port in use):
  supervisor detects early exit before reaching `CONNECTED`, cleans up,
  and the parent's 30s wait surfaces the failure with the tail of
  `daemon.log` instead of a bare timeout message.
- **Requires elevated privileges:** unchanged from today — since the
  supervisor is a re-exec of the same binary, it inherits whatever
  privilege `connect -d` itself was run under (e.g. still needs `sudo`).
  `status`/`disconnect` themselves never need elevated privileges: they
  only ever talk to the supervisor's control socket over loopback TCP,
  which any local user can dial regardless of who owns the process.
- **Concurrent `disconnect` calls / control-socket races:** disconnect
  is idempotent — a second call after state is already removed just
  prints `Not connected.`

## Testing plan

- **Unit tests** (no real openvpn/network needed):
  - `pkg/daemon`: state file save/load/remove round-trip;
    management-protocol parsing against a fake TCP server that emits
    canned `>STATE:` lines, including malformed/partial responses;
    control-protocol `STATUS`/`STOP` request-response against a real
    `Control` server backed by a fake `Management` client.
  - `cmd/status`, `cmd/disconnect`: behavior against a fake state file +
    fake control server (not-connected, connected, stale-PID,
    socket-unreachable cases).
- **Manual verification** (real VPN connections can't run in
  CI/sandbox): `connect -d`, `status`, `disconnect` exercised by hand on
  macOS/Linux with real openvpn. The Windows detach path is reviewed
  carefully but can't be verified in this environment (no Windows box) —
  called out explicitly as unverified in the PR.
