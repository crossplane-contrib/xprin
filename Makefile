.DEFAULT_GOAL:=local

GOLANGCI_LINT_VERSION?=v2.5.0
GOFUMPT_VERSION=v0.7.0

help: ## displays this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: local
local: ## runs fmt, lint, and test commands for local development (default)
	@echo run "\033[36m"make help"\033[0m" to see available targets
	@make fmt lint test

.PHONY: reviewable
reviewable: local ## same as local

.PHONY: clean
clean: ## cleans generated stuff and binaries in use
	rm -rf .bin/

.PHONY: version
version: ## gets the version
	@GIT_COMMIT_TIMESTAMP=$$(git show -s --format=%ct HEAD); \
		GIT_COMMIT_SHORT_HASH=$$(git rev-parse --short HEAD); \
	echo v0.0.0-$${GIT_COMMIT_TIMESTAMP}-$${GIT_COMMIT_SHORT_HASH}

.PHONY: install
install: ## installs xprin only
	@XPRIN_VERSION=$$(make --no-print-directory version); \
	CGO_ENABLED=0 go install -ldflags="-s -w -X=github.com/crossplane-contrib/xprin/internal/version.version=$$XPRIN_VERSION" ./cmd/xprin

.PHONY: install-all
install-all: ## installs xprin and xprin-helpers
	@XPRIN_VERSION=$$(make --no-print-directory version); \
	CGO_ENABLED=0 go install -ldflags="-s -w -X=github.com/crossplane-contrib/xprin/internal/version.version=$$XPRIN_VERSION" ./...

.PHONY: lint
lint: .bin/golangci-lint ## runs lint
	./.bin/golangci-lint run --timeout 5m

.PHONY: test
test: ## runs tests
	go test ./...

.PHONY: fmt
fmt: .bin/gofumpt ## formats go and cue files
	@./.bin/gofumpt -w .

.PHONY: install-tools
install-tools: .bin/golangci-lint .bin/gofumpt ## installs tools

.bin/gofumpt:
	GOBIN="$$(pwd)/.bin" go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)

.bin/golangci-lint:
	GOBIN="$$(pwd)/.bin" go install  github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
