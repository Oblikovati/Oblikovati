// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os/exec"
	"slices"
	"strings"
	"testing"
)

// The edge allowlist (#1623, audit B12): every import between first-party package trees
// must be declared here, so a new dependency edge is an explicit architecture decision
// (a row in this table, reviewed) instead of a convenient import nobody notices. The
// table is seeded from the INTENDED architecture — the two edges the 2026-07 audit found
// inverted are marked TODO with the issue that owns removing them, so the debt stays
// visible where it is enforced.
//
// Granularity: one row per top-level tree (addin/* split per subpackage — the router's
// allowances must not leak to the other add-in infrastructure). Direct imports are
// checked, which fully governs the transitive graph too: third-party packages cannot
// import oblikovati.org, so every first-party dependency chain is a walk over the edges
// in this table. TestDomainPackagesAreGpuFree (boundary_test.go) stays as the transitive
// belt-and-braces for the domain roots.
var allowedTreeImports = map[string][]string{
	// Domain (ADR-0014: pure, headless, cgo-free; also held by TestDomainPackagesAreGpuFree).
	"kernel": {"api", "build", "math"},
	"model": {"api", "build", "event", "kernel", "math", "solve",
		// TODO(B4 #1615): INVERSION — the domain must not depend on the persistence
		// layer; yamlcodec moves to a neutral leaf package, then this entry goes.
		"persistence"},
	"math":  {},
	"solve": {"math"},

	// Application & orchestration.
	"app": {"addincat", "api", "build", "clientgraphics", "command", "event", "kernel",
		"math", "model", "osfont", "persistence", "renderer", "report", "scene", "theme",
		"update", "userconfig"},
	"command": {"model"},
	"event":   {},
	"script":  {"addin/dispatch", "api", "app"},

	// Persistence & small leaves.
	"persistence": {"api", "model", "userconfig"},
	"userconfig":  {},
	"build":       {},
	"crcpost":     {},
	"perf":        {},
	"release":     {},
	"report":      {"crcpost"},
	"theme":       {"api", "persistence", "userconfig"},
	"update":      {},
	"usagestats":  {"crcpost", "persistence"},
	"addincat":    {"persistence"},
	// osfont adapts host-installed fonts to the pure text model (model/text); the
	// dependency points at the domain, never the reverse (ADR-0031).
	"osfont": {"model"},

	// Presentation (ADR-0014: renderer swappable, scene is its own leaf).
	"renderer":       {"kernel", "math", "scene"},
	"scene":          {"math"},
	"clientgraphics": {"api", "math", "renderer", "scene"},

	// Add-in infrastructure (thin-router rule: the router adapts wire to app/model,
	// nothing else grows presentation reach).
	"addin/dispatch":    {},
	"addin/events":      {"api", "app", "event", "math", "model"},
	"addin/modelaccess": {"app", "model"},
	"addin/opregistry":  {"addin/modelaccess", "api", "app", "kernel", "math", "model"},
	"addin/router": {"addin/modelaccess", "addin/opregistry", "addin/trace", "api", "app",
		"clientgraphics", "event", "kernel", "math", "model", "osfont", "script",
		// TODO(B10 #1621): the router reaches renderer/scene internals for lighting/
		// environment state; app-owned value types replace these, then both entries go.
		"renderer", "scene"},
	"addin/trace": {"api"},

	// Composition roots may reach anything below them.
	"cmd": {"addin/opregistry", "addin/router", "api", "app", "build", "kernel", "math",
		"model", "perf", "persistence", "release", "script", "update"},

	// The guard itself (shells out to go list; imports nothing first-party).
	"archguard": {},
}

// unenforcedTrees are checked-in trees that hold no shipped code: disposable experiments
// and test fixtures may import whatever they need without a table row.
var unenforcedTrees = map[string]bool{"experiments": true, "test-utilities": true}

// TestFirstPartyImportEdgesAreAllowlisted walks the DIRECT import edges of every
// first-party package (one `go list` invocation) and fails on any tree-to-tree edge
// missing from allowedTreeImports — including a tree with no row at all, so a new
// top-level package must declare its dependencies to exist.
func TestFirstPartyImportEdgesAreAllowlisted(t *testing.T) {
	for pkg, imports := range firstPartyImports(t) {
		src := packageTree(pkg)
		if unenforcedTrees[src] {
			continue
		}
		allowed, known := allowedTreeImports[src]
		if !known {
			t.Errorf("package %s belongs to tree %q which has no row in allowedTreeImports — "+
				"declare the tree's allowed imports in archguard/edge_allowlist_test.go (#1623).", pkg, src)
			continue
		}
		reportForbiddenEdges(t, pkg, src, imports, allowed)
	}
}

// reportForbiddenEdges fails on every import of pkg that crosses trees without a row.
func reportForbiddenEdges(t *testing.T, pkg, src string, imports, allowed []string) {
	for _, imp := range imports {
		dst := packageTree(imp)
		if dst == src {
			continue
		}
		if !slices.Contains(allowed, dst) {
			t.Errorf("%s imports %s: edge %q -> %q is not in allowedTreeImports — either the "+
				"import is an architecture violation, or the edge is a deliberate decision that "+
				"belongs in the table with a comment (#1623).", pkg, imp, src, dst)
		}
	}
}

// packageTree maps an import path to its allowlist row: the top-level directory, except
// addin/* which is split one level deeper (the router's allowances are its own).
func packageTree(importPath string) string {
	p := strings.TrimPrefix(importPath, "oblikovati.org/")
	parts := strings.Split(p, "/")
	if parts[0] == "addin" && len(parts) > 1 {
		return parts[0] + "/" + parts[1]
	}
	return parts[0]
}

// firstPartyImports returns, for every first-party package, its DIRECT first-party
// imports — one `go list` invocation over the module (F.I.R.S.T-fast, no per-package
// process spawns).
func firstPartyImports(t *testing.T) map[string][]string {
	t.Helper()
	cmd := exec.Command("go", "list", "-f", `{{.ImportPath}} {{join .Imports " "}}`, "./...")
	cmd.Dir = ".."
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list -f imports ./... : %v", err)
	}
	return parseImportLines(string(out))
}

// parseImportLines keeps only oblikovati.org packages and their oblikovati.org imports.
func parseImportLines(out string) map[string][]string {
	edges := map[string][]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 0 || !strings.HasPrefix(fields[0], "oblikovati.org/") {
			continue
		}
		var deps []string
		for _, imp := range fields[1:] {
			if strings.HasPrefix(imp, "oblikovati.org/") {
				deps = append(deps, imp)
			}
		}
		edges[fields[0]] = deps
	}
	return edges
}
