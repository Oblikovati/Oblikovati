# SPDX-License-Identifier: GPL-2.0-only
export PATH := $(CURDIR)/.tools:$(HOME)/.cargo/bin:$(PATH)
CARGO ?= cargo

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
	"$(CARGO)" install cargo-llvm-cov --locked --version 0.9.1
	"$(CARGO)" xtask tools

gate:
	"$(CARGO)" xtask gate

build:
	"$(CARGO)" xtask build

test:
	"$(CARGO)" xtask test

lint:
	"$(CARGO)" xtask lint

docs-lint:
	npm exec --offline -- markdownlint-cli2
	"$(CARGO)" xtask docs

package:
	"$(CARGO)" xtask package "$(TARGET)" "$(VERSION)"

release-check:
	"$(CARGO)" xtask preflight publish

.PHONY: script-docs
script-docs:
	"$(CARGO)" run --locked -p oblikovati-build -- script-docs

.PHONY: cache-stats
cache-stats:
	kache stats --json

.PHONY: cache-tools
cache-tools:
	"$(CARGO)" xtask cache-tools
