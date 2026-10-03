# Changelog

## 0.8.0

- Enable GoReleaser to publish Winget manifest updates and open PRs on tagged releases.
- Fix `vpngate --version` reporting a stale hardcoded version: `cmd.version` is now injected at build time via ldflags, from the latest versioned CHANGELOG.md entry (`just build`) or the release tag (goreleaser).
- Add jittered exponential backoff to both reconnect loops (foreground `connect --reconnect` and the background supervisor), so a dead server no longer spins respawning openvpn; the sequence resets after a long-lived connection.
- Fix `--country us` also matching Russia: country filtering now prefers exact matches on the country code or full name, falling back to substring matching only when nothing matches exactly.
- The daemon now waits for openvpn to report CONNECTED (not just the management interface answering) before writing its state file, so "Connected in background" means the tunnel is up; `connect -d` fails fast with the daemon log tail when the daemon dies during startup instead of waiting out the full timeout.
- Fix `disconnect` leaving an orphaned openvpn behind when the supervisor crashed: the fallback path now also kills the recorded openvpn PID, and `connect -d` refuses to start a second daemon while one is still negotiating.
- Strip script/plugin directives (`up`, `down`, `plugin`, `script-security`, etc.) from volunteer-server OpenVPN configs before handing them to openvpn, with a warning naming what was disabled.
- Ctrl+C during a foreground `connect` no longer leaks the temp OpenVPN config in /tmp (SIGINT/SIGTERM handler removes it).
- Broaden `--data-ciphers` from CBC-only to `AES-256-GCM:AES-128-GCM:AES-256-CBC:AES-128-CBC` so modern servers negotiate GCM while legacy ones still work.
- On Windows, fall back to PATH lookup for openvpn when the default install path doesn't exist.
- Write daemon state and the server-list cache atomically (temp file + rename) so a crash can't leave half-written JSON behind.
- Replace `github.com/juju/errors` with stdlib `fmt.Errorf`/`%w` wrapping.
- Proxy HTTP transports now clone `http.DefaultTransport` (sane TLS/connection timeouts) instead of using a zero-value transport; the SOCKS5 dial is now properly bounded by a context timeout without a lingering goroutine per connection.
- `util.Retry` no longer sleeps after the final attempt.
- `just lint` installs golangci-lint into `.bin/` via `go install` instead of `go get`, which was polluting go.mod/go.sum.
- Rename `cmd/servers.go` to `cmd/filters.go` (it holds shared filter/sort helpers, not a command).
- Switch `math/rand` to `math/rand/v2`.
- Refresh AGENTS.md (error-handling docs, project structure, version injection) and fix the README's deprecated `go get` install line.

## 0.7.0

- Add `vpngate logs` to view the log for a background connection started with `connect -d`, with `-f`/`--follow` and `-n`/`--lines` options.
- Simplify the interactive server selection list to show a country flag emoji, country name, and IP address (e.g. `🇯🇵 Japan (219.100.37.4)`), dropping the hostname/ping/score columns and column-alignment padding.
- Alias vpngate.net's `Korea Republic of` and `Russian Federation` country names to `South Korea` and `Russia`.

## 0.6.0

- Add `connect -d`/`--daemon` to run a vpn connection in the background.
- Add `vpngate status` to check on a background connection started with `connect -d`.
- Add `vpngate disconnect` to tear down a background connection started with `connect -d`.
- Add Winget packaging for Windows, with GoReleaser opening manifest update PRs on tagged releases.
- Fix `connect -d` silently timing out with no useful error when OpenVPN isn't installed: the background supervisor now logs its own startup failures to `daemon.log` (previously discarded, since the detached process has no console), with the same "is required, please install it" message the foreground `connect` command already gives.

## 0.5.0

- Fix a nil-pointer panic when the vpngate.net server list API returns a non-200 status code.
- Fix the retry backoff between failed server-list fetch attempts, which was effectively instantaneous (1ns) instead of 1 second.
- Fix `connect --reconnect` handling so a single connection attempt (without `--reconnect`) no longer loops forever after a clean disconnect.
- Fix a potential deadlock when reading OpenVPN's stdout/stderr output.
- Fix a leftover temporary OpenVPN config file when writing or closing it failed.
- Return errors from CLI commands instead of calling `log.Fatal` directly, for cleaner and more consistent error output.
- Update golang.org/x/net to v0.55.0 [security].
- Add test coverage for retry logic and CLI helper functions.

## 0.4.0

- Add server filtering by country, maximum ping, and minimum score to list and connect commands.
- Add list sorting by score, ping, country, or hostname.
- Add JSON and CSV output formats for the list command.
- Add cache controls with refresh/no-cache flags and cache management commands.
- Improve interactive server selection labels with aligned hostname, country, IP, ping, and score details.
- Add usage examples for filtering, sorting, cache controls, and random filtered connections.

## 0.3.5

- chore: update vendorHash in flake.nix (7948580)
- Refactor codebase (bb88db9)

## 0.3.4

- chore(deps): update dependency go to v1.26.0 (#169) (6550901)
- Update module github.com/olekukonko/tablewriter to v1.1.3 (#171) (c03d27a)
- Update module golang.org/x/net to v0.50.0 (#170) (7da1504)

## 0.3.3

- Update dependency go to v1.25.6 (#167) (ff1d10e)
- Update module golang.org/x/net to v0.49.0 (#168) (9939da1)
- Update module github.com/spf13/afero to v1.15.0 (#143) (9fa908f)
- Update module github.com/spf13/cobra to v1.10.2 (#166) (3f7d49f)
- Update module golang.org/x/net to v0.48.0 (#164) (5ac7d49)

## 0.3.2

- Update dependency go to 1.25 (#156) (1de072b)
- Update module github.com/spf13/cobra to v1.10.1 (#159) (486fc18)
- Update module github.com/stretchr/testify to v1.11.1 (#158) (52cadc8)
- Update module github.com/rs/zerolog to v1.34.0 (#151) (98bb23e)
- Update module github.com/spf13/cobra to v1.9.1 (#147) (552a6e3)
- Update module golang.org/x/net to v0.35.0 (#145) (4bd470b)

## 0.3.1

- Add "386" goarch to .goreleaser.yaml (4c66b19)

## 0.3.0

- Add initial support and docs for Windows (#132) (3e819c5)
