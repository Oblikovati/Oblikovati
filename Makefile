# Oblikovati developer task runner.
# `make help` lists targets. The core (math/kernel/model) is cgo-free, so most
# targets pin CGO_ENABLED=0 for fast, hardware-independent, cross-buildable runs
# (architecture/ADR-0008). The race detector needs cgo, so `test-race` is separate.

GO          ?= go
MODULE      := oblikovati
PKG         := ./...
DIST        := dist

# VERSION is {MANUAL_MAJOR}.{API_VERSION}.{MINOR}.{PATCH}, computed by cmd/obkversion
# from version.yaml + the api pin + git tags/commit scope, so local builds match the
# release scheme. CI overrides it with the exact release version. Falls back to git
# describe, then "dev" (e.g. on a shallow checkout without tags).
VERSION     ?= $(shell go run ./cmd/obkversion stable 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X $(MODULE)/build.Version=$(VERSION) \
	-X $(MODULE)/build.Commit=$(COMMIT) \
	-X $(MODULE)/build.Date=$(DATE)

# Pinned dev tools (installed on demand into $GOBIN by `make tools`).
GOLANGCI_VERSION  ?= v1.59.1
GOTESTSUM_VERSION ?= v1.12.0

# Cross-build matrix for `make build-all`.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Minimum total coverage gate for `make cover` (raise as the suite grows).
COVER_MIN ?= 0

.DEFAULT_GOAL := help

.PHONY: help
help: ## List available targets
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) \
	  | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

.PHONY: tidy
tidy: ## Sync go.mod / go.sum
	$(GO) mod tidy

.PHONY: fmt
fmt: ## Format the code (gofumpt via golangci-lint, falls back to gofmt)
	@gofmt -w .

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofmt-clean
	@out=$$(gofmt -l .); if [ -n "$$out" ]; then echo "gofmt needed:"; echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## Run go vet (cgo-free)
	CGO_ENABLED=0 $(GO) vet $(PKG)

.PHONY: lint
lint: ## Run golangci-lint (install with `make tools`)
	golangci-lint run
	$(MAKE) -C head lint

.PHONY: docs-lint
docs-lint: ## Lint the docs (markdownlint via npx; needs node). Link check runs in CI (lychee).
	npx --yes markdownlint-cli2

.PHONY: test
test: ## Tier 1: fast cgo-free unit tests
	CGO_ENABLED=0 $(GO) test $(PKG)

.PHONY: test-race
test-race: ## Run the suite under the race detector (needs cgo)
	CGO_ENABLED=1 $(GO) test -race $(PKG)

.PHONY: cover
cover: ## Run tests with coverage and enforce COVER_MIN
	CGO_ENABLED=0 $(GO) test -covermode=count -coverprofile=coverage.out $(PKG)
	@total=$$($(GO) tool cover -func=coverage.out | awk '/total:/ {print $$3}' | tr -d '%'); \
	  echo "total coverage: $$total% (min $(COVER_MIN)%)"; \
	  awk "BEGIN{exit !($$total >= $(COVER_MIN))}" \
	  || { echo "coverage below threshold"; exit 1; }

.PHONY: build
build: ## Build both binaries into $(DIST) with version stamping
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST)/oblikovati-cli ./cmd/oblikovati-cli
	CGO_ENABLED=0 $(GO) build -ldflags "$(LDFLAGS)" -o $(DIST)/oblikovati     ./cmd/oblikovati

.PHONY: build-all
build-all: ## Cross-compile oblikovati-cli for every target in PLATFORMS
	@for p in $(PLATFORMS); do \
	  os=$${p%/*}; arch=$${p#*/}; ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
	  echo "→ $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" \
	    -o $(DIST)/oblikovati-cli-$$os-$$arch$$ext ./cmd/oblikovati-cli || exit 1; \
	done

.PHONY: run-cli
run-cli: ## Run the headless CLI
	CGO_ENABLED=0 $(GO) run -ldflags "$(LDFLAGS)" ./cmd/oblikovati-cli

.PHONY: ci
ci: fmt-check vet lint test-race cover ## Everything CI runs, locally

.PHONY: tools
tools: ## Install pinned dev tools into $GOBIN
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION)
	$(GO) install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

.PHONY: hooks
hooks: ## Point git at the repo's pre-commit hook
	git -C .. config core.hooksPath .githooks
	@echo "git hooksPath set to .githooks"

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST) coverage.out
