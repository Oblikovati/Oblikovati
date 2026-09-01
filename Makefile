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
# golangci-lint v2 lives at the /v2 module path; v1.59.1 and gotestsum v1.12.0
# no longer compile under Go 1.26 (x/tools tokeninternal array-length error).
# golangci-lint refuses to run against a module targeting a newer Go than the
# toolchain it was itself built with, so the pin must track this module's `go`
# directive: v2.13.1 is the first release built with go1.27.0.
GOLANGCI_VERSION  ?= v2.13.1
GOTESTSUM_VERSION ?= v1.13.0

# Cross-build matrix for `make build-all`.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Minimum total coverage gate for `make cover` (raise as the suite grows).
COVER_MIN ?= 0

# Revision `make test-impacted` compares against to find the change set.
IMPACT_BASE ?= origin/develop

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

# The suite runs in two tiers (architecture/testing/03-test-tiers-and-selection.md).
# TIER 1 (`make test`, -short) skips the corpus and oracle tests. TIER 2 (`make
# test-corpus`) runs everything. 152 tests hold 94% of the suite's cost, so the split
# is what keeps the inner loop in seconds instead of ten minutes.
#
# CORPUS_TIMEOUT overrides `go test`'s 10m PER-PACKAGE default, which model/feature/
# occtparity exceeds on its own. A timed-out package reports as a FAILURE that reads
# exactly like a real one, so tier 2 must stay generous.
CORPUS_TIMEOUT ?= 60m

# Seconds a TIER 1 package may take. The gate exists to catch a new corpus test that
# forgot its testing.Short() guard: such a test costs 100s+ on its own, while the
# slowest honest tier-1 package sits near 15s on 32 cores and near 40s on a 4-core CI
# runner. Package WALL time is the only budget that survives t.Parallel() — per-test
# elapsed stretches with core contention (see cmd/testslowest).
TIER1_PACKAGE_BUDGET ?= 90

.PHONY: test
test: ## Tier 1: fast cgo-free unit tests (skips the corpus tier)
	CGO_ENABLED=0 $(GO) test -short $(PKG)

.PHONY: test-corpus
test-corpus: ## Tier 2: the whole suite, corpus and oracle tests included
	CGO_ENABLED=0 $(GO) test -timeout $(CORPUS_TIMEOUT) $(PKG)

.PHONY: test-impacted
test-impacted: ## Tier 1 on only the packages the current change set can affect
	@pkgs=$$($(GO) run ./cmd/testimpact -base $(IMPACT_BASE)); \
	  if [ -z "$$pkgs" ]; then echo "no package owns the change — nothing to test"; exit 0; fi; \
	  echo "$$pkgs" | sed 's/^/  → /'; \
	  CGO_ENABLED=0 $(GO) test -short $$pkgs

.PHONY: test-impacted-corpus
test-impacted-corpus: ## Tier 2 on only the packages the current change set can affect
	@pkgs=$$($(GO) run ./cmd/testimpact -base $(IMPACT_BASE)); \
	  if [ -z "$$pkgs" ]; then echo "no package owns the change — nothing to test"; exit 0; fi; \
	  echo "$$pkgs" | sed 's/^/  → /'; \
	  CGO_ENABLED=0 $(GO) test -timeout $(CORPUS_TIMEOUT) $$pkgs

.PHONY: test-budget
test-budget: ## Tier 1 with the per-package time budget enforced (the CI gate)
	@CGO_ENABLED=0 $(GO) test -short -json $(PKG) \
	  | $(GO) run ./cmd/testslowest -top 15 -package-budget $(TIER1_PACKAGE_BUDGET)

.PHONY: test-slowest
test-slowest: ## Rank the whole suite by test time (what to guard next)
	@CGO_ENABLED=0 $(GO) test -timeout $(CORPUS_TIMEOUT) -json $(PKG) \
	  | $(GO) run ./cmd/testslowest -top 40

.PHONY: test-slowest-serial
test-slowest-serial: ## Rank tier 2 with NO parallelism — the measurement the 2s guard rule uses
	@CGO_ENABLED=0 $(GO) test -timeout $(CORPUS_TIMEOUT) -p 1 -parallel 1 -json $(PKG) \
	  | $(GO) run ./cmd/testslowest -top 40

.PHONY: test-race
test-race: ## Run the suite under the race detector (needs cgo)
	CGO_ENABLED=1 $(GO) test -race -timeout $(CORPUS_TIMEOUT) $(PKG)

.PHONY: cover
cover: ## Run tests with coverage and enforce COVER_MIN
	CGO_ENABLED=0 $(GO) test -covermode=count -coverprofile=coverage.out -timeout $(CORPUS_TIMEOUT) $(PKG)
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

# `cover` already runs the whole suite, so `ci` must not also run `test`/`test-corpus`.
# `test-race` is the release gate (ci.yml runs it only on push to `release`), not a
# per-change one; use `make ci-race` before cutting a release.
.PHONY: ci
ci: fmt-check vet lint cover ## Everything a PR must pass, locally

.PHONY: ci-race
ci-race: ci test-race ## `make ci` plus the race detector (the pre-release gate)

.PHONY: tools
tools: ## Install pinned dev tools into $GOBIN
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
	$(GO) install gotest.tools/gotestsum@$(GOTESTSUM_VERSION)

.PHONY: hooks
hooks: ## Point git at the repo's pre-commit hook
	git config core.hooksPath .githooks
	@echo "git hooksPath set to .githooks"

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST) coverage.out
