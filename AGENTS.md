# AGENTS.md

This file provides guidance for AI coding agents operating in this repository.

## Project Overview

`vpngate` is a Go CLI client for [vpngate.net](https://www.vpngate.net/) that fetches
VPN relay servers, lets you filter/select one, and connects via OpenVPN. It uses Cobra
for CLI, zerolog for logging, and supports macOS, Linux, and Windows.

## Build / Lint / Test Commands

```bash
make build                # outputs to dist/vpngate (CGO_ENABLED=0)
make test                 # go test -v ./...
make lint                 # golangci-lint run (installs v2.6.2 if needed)

# Run a single test by name
go test -v -run ^TestParseVpnList$ ./pkg/vpn/

# Run tests in a specific package
go test -v ./pkg/vpn/

# Generate CLI docs into README
go run main.go docs --path README.md
```

CGO is disabled globally (`CGO_ENABLED=0` in Makefile, `.goreleaser.yaml`, and `flake.nix`).

## Project Structure

```
main.go              Entry point (logging setup, calls cmd.Execute())
cmd/
  root.go            Root cobra command + Execute()
  connect.go         "connect" subcommand with interactive server selection
  list.go            "list" subcommand (table display)
  docs.go            Hidden "docs" subcommand (README generation)
pkg/
  vpn/
    list.go          Server struct, GetList(), CSV parsing, HTTP client
    list_test.go     Tests for list.go
    client.go        Connect() - invokes openvpn binary
    cache.go         File-based JSON cache (~/.vpngate/cache/)
  exec/run.go        Generic command executor with logging
  util/retry.go      Retry utility function
test_data/
  vpn_list.csv       Test fixture (sample CSV with 98 servers)
```

## Code Style Guidelines

### Formatting

Use `gofmt`/`goimports`. No `.editorconfig` or custom formatter config.
The project relies on golangci-lint defaults (no `.golangci.yml`).

### Imports

Use three groups separated by blank lines:

1. Standard library
2. Third-party packages
3. Local packages (`github.com/davegallant/vpngate/...`)

```go
import (
    "encoding/base64"
    "fmt"
    "os"

    "github.com/rs/zerolog/log"
    "github.com/spf13/cobra"

    "github.com/davegallant/vpngate/pkg/vpn"
)
```

Import aliases are used sparingly (e.g., `tw "github.com/olekukonko/tablewriter"`).

### Naming Conventions

- **Files**: lowercase, single word (`client.go`, `cache.go`). Underscores only in
  test files (`list_test.go`) and test data dirs (`test_data/`).
- **Exported functions/types**: PascalCase (`Execute`, `Connect`, `GetList`, `Server`).
- **Unexported functions**: camelCase (`parseVpnList`, `createHTTPClient`).
- **Constants**: camelCase for unexported (`vpnList`, `httpClientTimeout`, `dialTimeout`).
- **CLI flag variables**: `flagXxx` prefix (`flagRandom`, `flagReconnect`, `flagProxy`).
- **Cobra commands**: `xxxCmd` suffix (`rootCmd`, `connectCmd`, `listCmd`).

### Types

There is one main struct (`Server` in `pkg/vpn/list.go`) with CSV struct tags.
Functions return `*[]Server` (pointer to slice) throughout the codebase. While
non-idiomatic, maintain this pattern for consistency unless refactoring broadly.

### Error Handling

The codebase uses different patterns by layer:

**CLI layer (`cmd/`)** - terminate on error:
```go
log.Fatal().Msg(err.Error())
```

**Package layer (`pkg/vpn/list.go`)** - wrap errors with context using `juju/errors`:
```go
return nil, errors.Annotate(err, "Unable to read stream")
return nil, errors.Annotatef(err, "Unexpected status code: %d", resp.StatusCode)
```

**Utility layer (`pkg/vpn/cache.go`, `pkg/exec/`)** - return bare errors.

**Cleanup operations** - explicitly discard errors:
```go
_ = resp.Body.Close()
_ = os.Remove(tmpfile.Name())
```

**Deferred cleanup** uses anonymous functions:
```go
defer func() { _ = resp.Body.Close() }()
```

### Logging

Uses `github.com/rs/zerolog` with `ConsoleWriter` to stderr (configured in `main.go`).
Levels: `Fatal` (unrecoverable CLI errors), `Error` (retryable/command failures),
`Warn` (non-critical), `Info` (status), `Debug` (detailed output).
Pattern: `log.Fatal().Msg(err.Error())` or `log.Info().Msgf("format %s", val)`.

### CLI Flags

Flags are registered in `init()` functions using `cobra` `Flags()` methods.
Flag variables are package-level `var` declarations in `cmd/`. Some flag vars
(`flagProxy`, `flagSocks5Proxy`) are shared across command files within the `cmd`
package. Argument validation uses cobra validators (`cobra.RangeArgs`, `cobra.NoArgs`).

### Testing

- Framework: `github.com/stretchr/testify/assert`
- Test names: descriptive PascalCase (`TestParseVpnList`, `TestGetListReal`)
- Each test has a single-line doc comment above it
- Test data lives in `test_data/` at the repo root, accessed via relative paths
- `TestGetListReal` is an integration test that hits the live API (requires network)
- `TestParseVpnList` is a unit test using the local CSV fixture
- Assertion style in existing code uses `assert.Equal(t, actual, expected)` - note
  this is inverted from testify's recommended `assert.Equal(t, expected, actual)`

### Dependencies

Key dependencies (see `go.mod`):
- `github.com/spf13/cobra` - CLI framework
- `github.com/rs/zerolog` - Structured logging
- `github.com/juju/errors` - Error annotation
- `github.com/stretchr/testify` - Test assertions
- `github.com/AlecAivazis/survey/v2` - Interactive prompts
- `github.com/jszwec/csvutil` - CSV parsing
- `github.com/olekukonko/tablewriter` - Table output
- `golang.org/x/net` - SOCKS5 proxy support

### CI/CD

GitHub Actions workflows in `.github/workflows/`:
- `golangci-lint.yml` - Lint + test on push/PR
- `release.yml` - GoReleaser on tag push (cross-compiles for linux/darwin/windows)
- `update-vendor-hash.yml` - Auto-updates Nix flake vendorHash
- `update-docs.yml` - Auto-updates README CLI docs
