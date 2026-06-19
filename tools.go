//go:build tools

// SPDX-License-Identifier: GPL-2.0-only

// Package tools pins the versions of build/test tooling in go.mod and go.sum so CI
// installs a locked, reproducible version rather than resolving one ad hoc at run time
// (SonarCloud githubactions:S8545). The blank import is never compiled into a binary —
// the `tools` build tag excludes this file from every normal build — it exists only so
// `go mod` records the tool as a dependency. Install with `go install <import path>`
// from this module; the version comes from go.mod.
package tools

import (
	_ "gotest.tools/gotestsum"
)
