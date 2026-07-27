bin := "dist/vpngate"

export CGO_ENABLED := "0"

# Builds the binary
build:
    go build -o {{bin}}

# Run unit tests
test:
    go test -v ./...

# Run lint
lint:
    @go get github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2
    golangci-lint run

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
