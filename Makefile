# SPDX-License-Identifier: GPL-2.0-only
export PATH := $(CURDIR)/.tools:$(HOME)/.cargo/bin:$(PATH)

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
