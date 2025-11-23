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
