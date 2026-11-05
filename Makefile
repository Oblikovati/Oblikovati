# SPDX-License-Identifier: GPL-2.0-only
export PATH := $(CURDIR)/.tools:$(HOME)/.cargo/bin:$(PATH)

BUILD_CACHE ?= kache
KACHE_CONFIG ?= $(CURDIR)/.kache.toml
KACHE_GIT_DIR := $(shell git rev-parse --path-format=absolute --git-common-dir 2>/dev/null)
KACHE_CACHE_DIR ?= $(if $(KACHE_GIT_DIR),$(KACHE_GIT_DIR)/oblikovati-kache,$(CURDIR)/.cache/kache)
CACHED_TARGETS := gate build test lint docs-lint package release-check script-docs cache-stats
ifeq ($(BUILD_CACHE),kache)
$(CACHED_TARGETS): export RUSTC_WRAPPER ?= kache
$(CACHED_TARGETS): export KACHE_CONFIG := $(KACHE_CONFIG)
$(CACHED_TARGETS): export KACHE_CACHE_DIR := $(KACHE_CACHE_DIR)
else ifeq ($(BUILD_CACHE),off)
$(CACHED_TARGETS): export RUSTC_WRAPPER :=
else
$(error BUILD_CACHE=$(BUILD_CACHE): expected kache or off)
endif

.PHONY: tools gate build test lint docs-lint package release-check
tools:
	npm ci
	cargo install cargo-llvm-cov --locked --version 0.9.1
	cargo xtask tools

gate:
	cargo xtask gate

build:
	cargo xtask build

test:
	cargo xtask test

lint:
	cargo xtask lint

docs-lint:
	npm exec --offline -- markdownlint-cli2
	cargo xtask docs

package:
	cargo xtask package "$(TARGET)" "$(VERSION)"

release-check:
	cargo xtask preflight publish

.PHONY: script-docs
script-docs:
	cargo run --locked -p oblikovati-build -- script-docs

.PHONY: cache-stats
cache-stats:
	kache stats --json

.PHONY: cache-tools
cache-tools:
	cargo xtask cache-tools
