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

release: ## Tag and push a release, e.g. make release VERSION=0.5.0 (goreleaser publishes the GitHub release on tag push)
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required, e.g. make release VERSION=0.5.0"; \
		exit 1; \
	fi
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Working tree is not clean"; \
		exit 1; \
	fi
	git push origin HEAD
	git tag v$(VERSION)
	git push origin v$(VERSION)
.PHONY: release
