// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// The project rests on four import-boundary invariants. TestSourceDoesNotDependOnAnAddIn (this package)
// holds the first (source must not require an add-in module). These tests add two more with the actual
// import GRAPH (go list -deps / go.mod), so a leak becomes a red build instead of a convention nobody
// checks:
//
//   - DOMAIN PURITY: kernel/model/math/solve must not import the presentation layer (head/renderer/scene/
//     addin-router). Keeps the kernel cgo-free and headless-testable and the renderer swappable (ADR-0014).
//   - API PURITY: the Apache-2.0 api module must not depend on the GPL source. The license split (ADR-0018)
//     depends on the dependency flowing only one way (source -> api).
//
// The fourth invariant — an add-in's shipped (non-test) code must not import the GPL source — IS
// enforced per add-in repo (each add-in is its own module; MCPBridge legitimately requires source for
// _test.go drivers): every Oblikovati.AddIns.* repo carries gplpurity/gplpurity_test.go, a go-list
// guard over its non-test import graph, running in its ordinary CI test sweep (#1614, audit B3).

// presentationPrefixes are the import paths the domain must never reach.
var presentationPrefixes = []string{
	"oblikovati.org/head",
	"oblikovati.org/renderer",
	"oblikovati.org/scene",
	"oblikovati.org/addin/router",
}

// domainRoots are the pure-domain package trees this guard holds GPU-free.
var domainRoots = []string{"./kernel/...", "./model/...", "./math/...", "./solve/..."}

// TestDomainPackagesAreGpuFree fails if any package under the domain roots TRANSITIVELY imports the
// presentation layer. It uses `go list -deps` (the full transitive import graph), so a leak two hops deep
// (domain -> helper -> renderer) is caught, not only a direct import.
func TestDomainPackagesAreGpuFree(t *testing.T) {
	deps := listDeps(t, "..", domainRoots...)
	for _, dep := range deps {
		for _, p := range presentationPrefixes {
			if dep == p || strings.HasPrefix(dep, p+"/") {
				t.Errorf("a domain package depends on %q — kernel/model/math/solve must not import the "+
					"presentation layer (head/renderer/scene/addin-router). Relocate the adapter out of the "+
					"domain (see #1500, ADR-0014).", dep)
			}
		}
	}
}

// listDeps returns the deduped transitive dependency import paths of the given package patterns, as
// reported by `go list -deps` run in dir (the module root for those patterns).
func listDeps(t *testing.T, dir string, patterns ...string) []string {
	t.Helper()
	cmd := exec.Command("go", append([]string{"list", "-deps"}, patterns...)...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -deps %v (dir %s): %v", patterns, dir, err)
	}
	var deps []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line != "" {
			deps = append(deps, line)
		}
	}
	return deps
}

// apiModuleGoMod is the sibling Apache-2.0 contract module's go.mod, resolved via the workspace layout.
const apiModuleGoMod = "../../Oblikovati.API/go.mod"

// TestApiModuleDoesNotRequireSource fails if the Apache-2.0 api module requires ANY oblikovati.org module
// — it is the root of the dependency graph, so it must require none (the GPL source requires it, never the
// reverse). The license split (ADR-0018) depends on this direction. Skips when the sibling is not checked
// out (a source-only build).
func TestApiModuleDoesNotRequireSource(t *testing.T) {
	if _, err := os.Stat(apiModuleGoMod); err != nil {
		t.Skipf("api module not checked out at %s: %v", apiModuleGoMod, err)
	}
	for _, mod := range requiredInternalModules(t, apiModuleGoMod) {
		t.Errorf("Oblikovati.API/go.mod requires %q — the Apache-2.0 contract must require no oblikovati.org "+
			"module (the dependency flows source -> api, never the reverse). ADR-0018.", mod)
	}
}
