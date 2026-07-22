# Changelog

## 0.6.0-rc1

- Add `connect -d`/`--daemon` to run a vpn connection in the background.
- Add `vpngate status` to check on a background connection started with `connect -d`.
- Add `vpngate disconnect` to tear down a background connection started with `connect -d`.
- Add winget packaging for Windows (manifest publishing is currently disabled pending fork/token setup).

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
