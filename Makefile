BIN ?= dist/vpngate

export CGO_ENABLED := 0

build: ## Builds the binary
	go build -o $(BIN)
.PHONY: build

test: ## Run unit tests
	go test -v ./...
.PHONY: test

lint: ## Run lint
	@go get github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.2
	golangci-lint run

docs: ## Regenerate CLI reference docs into docs/cli/
	go run ./tools/gendocs
.PHONY: docs

release: ## Tag and push a release using the CHANGELOG.md entry as the tag message, e.g. make release VERSION=0.5.0
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required, e.g. make release VERSION=0.5.0"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree is not clean"; \
		exit 1; \
	fi
	@notes="$$(awk -v ver="## $(VERSION)" '$$0==ver{f=1;next} /^## /{f=0} f' CHANGELOG.md | sed '/^$$/d')"; \
	if [ -z "$$notes" ]; then \
		echo "No CHANGELOG.md entry found for version $(VERSION) (expected a '## $(VERSION)' heading)"; \
		exit 1; \
	fi; \
	git push origin HEAD; \
	git tag -a v$(VERSION) -m "$$notes"; \
	git push origin v$(VERSION)
.PHONY: release
