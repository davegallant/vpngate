bin := "dist/vpngate"

# Version is read from the latest versioned CHANGELOG.md entry and injected
# at build time, so `vpngate --version` always matches the release being
# built.
version := `awk '/^## [0-9]/{print $2; exit}' CHANGELOG.md`

export CGO_ENABLED := "0"

# Builds the binary
build:
    go build -ldflags "-X github.com/davegallant/vpngate/cmd.version={{version}}" -o {{bin}}

# Run unit tests
test:
    go test -v ./...

# Run lint (installs golangci-lint into .bin/ when missing — `go install`
# with @version doesn't touch go.mod, unlike `go get`)
lint:
    #!/usr/bin/env bash
    set -euo pipefail
    if ! command -v golangci-lint >/dev/null 2>&1; then
        echo "installing golangci-lint into .bin/"
        GOBIN="$PWD/.bin" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2
    fi
    PATH="$PWD/.bin:$PATH" golangci-lint run

# Regenerate CLI reference docs into docs/cli/
docs:
    go run ./tools/gendocs

# Tag and push a release using the CHANGELOG.md entry as the tag message, e.g. just release 0.5.0
release version:
    #!/usr/bin/env bash
    set -euo pipefail
    if [ -z "{{version}}" ]; then
        echo "version is required, e.g. just release 0.5.0"
        exit 1
    fi
    if [ -n "$(git status --porcelain)" ]; then
        echo "Working tree is not clean"
        exit 1
    fi
    notes="$(awk -v ver="## {{version}}" '$0==ver{f=1;next} /^## /{f=0} f' CHANGELOG.md | sed '/^$/d')"
    if [ -z "$notes" ]; then
        echo "No CHANGELOG.md entry found for version {{version}} (expected a '## {{version}}' heading)"
        exit 1
    fi
    git push origin HEAD
    git tag -a "v{{version}}" -m "$notes"
    git push origin "v{{version}}"
