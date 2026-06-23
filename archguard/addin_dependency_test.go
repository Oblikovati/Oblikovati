// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"strings"
	"testing"
)

// allowedInternalModules are the only oblikovati.org module paths that the GPL
// application (/source) may `require`: the root app itself and the Apache-2.0 public
// API contract. Every OTHER oblikovati.org/* module is an ADD-IN, and /source must
// never depend on an add-in — the dependency flows the other way (add-in -> /api,
// with /source implementing /api). See ADR-0016 (C ABI boundary) and ADR-0018 (repo
// split). This guard exists because head/go.mod once carried a stray
// `require oblikovati.org/motor-designer` (from an untracked local capture driver,
// head/cmd/motorshot) that broke the macOS release build in CI.
var allowedInternalModules = map[string]bool{
	"oblikovati.org":     true, // the root application module (head requires it via replace)
	"oblikovati.org/api": true, // the Apache-2.0 contract, a sibling repo
}

// sourceGoModFiles lists the /source module files this guard inspects, relative to
// this package directory (a test's working directory is its own package dir).
var sourceGoModFiles = []string{"../go.mod", "../head/go.mod"}

// TestSourceDoesNotDependOnAnAddIn fails if any /source go.mod requires an
// oblikovati.org module other than the root app or the public API — i.e. an add-in.
func TestSourceDoesNotDependOnAnAddIn(t *testing.T) {
	for _, path := range sourceGoModFiles {
		for _, mod := range requiredInternalModules(t, path) {
			if !allowedInternalModules[mod] {
				t.Errorf("%s requires %q — /source must never depend on an add-in. "+
					"Allowed oblikovati.org modules are the root app and oblikovati.org/api; "+
					"move add-in-dependent code (e.g. capture drivers) into the add-in's own repo.",
					path, mod)
			}
		}
	}
}

// requiredInternalModules returns the oblikovati.org/* module paths that the go.mod at
// path requires, handling both the single-line `require x v1` and the block
// `require ( ... )` forms.
func requiredInternalModules(t *testing.T, path string) []string {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var mods []string
	inBlock := false
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "" || strings.HasPrefix(line, "//"):
			continue
		case strings.HasPrefix(line, "require ("):
			inBlock = true
		case inBlock && line == ")":
			inBlock = false
		case strings.HasPrefix(line, "require "):
			if mod := internalModuleFromRequire(strings.TrimPrefix(line, "require ")); mod != "" {
				mods = append(mods, mod)
			}
		case inBlock:
			if mod := internalModuleFromRequire(line); mod != "" {
				mods = append(mods, mod)
			}
		}
	}
	return mods
}

// internalModuleFromRequire pulls an oblikovati.org/* module path out of one require
// entry ("<module> <version> [// indirect]"), returning "" for third-party modules.
func internalModuleFromRequire(entry string) string {
	fields := strings.Fields(entry)
	if len(fields) < 2 {
		return ""
	}
	mod := fields[0]
	if mod == "oblikovati.org" || strings.HasPrefix(mod, "oblikovati.org/") {
		return mod
	}
	return ""
}
