// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"os/exec"
	"strings"
	"testing"
)

// The edge allowlist (#1623, audit B12): every import between first-party package trees
// must be declared here, so a new dependency edge is an explicit architecture decision
// (a row in this table, reviewed) instead of a convenient import nobody notices. The
// table is seeded from the INTENDED architecture — the remaining edge the 2026-07 audit found
// inverted is marked TODO with the issue that owns removing it, so the debt stays visible where
// it is enforced (the model->persistence inversion was removed by B4 #1615).
//
// Granularity: one row per top-level tree (addin/* split per subpackage — the router's
// allowances must not leak to the other add-in infrastructure). Direct imports are
// checked, which fully governs the transitive graph too: third-party packages cannot
// import oblikovati.org, so every first-party dependency chain is a walk over the edges
// in this table. TestDomainPackagesAreGpuFree (boundary_test.go) stays as the transitive
// belt-and-braces for the domain roots.
var allowedTreeImports = map[string][]string{
	// Domain (ADR-0014: pure, headless, cgo-free; also held by TestDomainPackagesAreGpuFree).
	// The kernel's rows are one level deep (#2194) because the ground rules pin its INTERNAL
	// direction — geom → topo → brep → ops → model — and a single "kernel" row hid every edge
	// inside it. Read down the list: each package may depend only on the ones below it.
	"kernel/diag":       {},
	"kernel/predicates": {},
	"kernel/geom":       {"api", "build", "math"},
	"kernel/topo":       {"kernel/diag", "kernel/geom", "math"},
	"kernel/brep":       {"kernel/diag", "kernel/geom", "kernel/topo", "math"},
	"kernel/subd":       {"kernel/geom", "kernel/topo", "math"},
	"kernel/fit":        {"kernel/geom", "math"},
	"kernel/blend":      {"kernel/geom", "kernel/topo", "math"},
	"kernel/meshbool":   {"kernel/predicates", "math"},
	"kernel/shading":    {"math"},
	"kernel/geomapi":    {"api", "kernel/geom", "math"},

	// kernel/ops is packaged BY OPERATION (#2183), so each family is its own row. The split
	// only buys anything — a change to one family that neither rebuilds nor re-tests the
	// others — while these edges stay declared, and NO family may name the kernel/ops façade:
	// the façade forwards to them, so that edge is a cycle waiting to happen.
	//
	// The substrate leaves, lowest first.
	"kernel/ops/internal/probe":    {"kernel/geom", "kernel/topo", "math"},
	"kernel/ops/internal/disjoint": {},
	"kernel/mesh":                  {"kernel/diag", "kernel/geom", "kernel/topo", "math"},
	"kernel/ops/internal/tol":      {"kernel/geom", "kernel/mesh", "kernel/topo", "math"},
	"kernel/ops/internal/retopo": {"kernel/geom", "kernel/mesh", "kernel/ops/internal/probe",
		"kernel/ops/internal/tol", "kernel/topo", "math"},

	// validate is the post-condition every other family runs, so it sits lowest of the families;
	// tessellate is a DERIVED view of the B-rep, below everything that models with it.
	"kernel/ops/validate": {"kernel/brep", "kernel/geom", "kernel/mesh",
		"kernel/ops/internal/probe", "kernel/topo", "math"},
	"kernel/ops/tessellate": {"kernel/brep", "kernel/diag", "kernel/geom", "kernel/meshbool",
		"kernel/mesh", "kernel/ops/internal/probe", "kernel/ops/validate",
		"kernel/predicates", "kernel/topo", "math"},
	"kernel/ops/transform": {"kernel/geom", "kernel/ops/internal/retopo", "kernel/topo", "math"},
	// blend is the one fillet/chamfer/draft engine (ADR-0050/0051).
	"kernel/ops/blend": {"api", "kernel/blend", "kernel/brep", "kernel/diag", "kernel/geom",
		"kernel/mesh", "kernel/ops/internal/probe", "kernel/ops/internal/retopo",
		"kernel/ops/internal/tol", "kernel/ops/tessellate", "kernel/ops/transform",
		"kernel/ops/validate", "kernel/topo", "math"},
	// query asks read-only questions of a body, and reads blend's edge convexity.
	"kernel/ops/query": {"kernel/brep", "kernel/diag", "kernel/geom", "kernel/ops/blend",
		"kernel/ops/internal/disjoint", "kernel/mesh", "kernel/ops/internal/probe",
		"kernel/ops/internal/tol", "kernel/ops/tessellate", "kernel/predicates", "kernel/topo", "math"},
	// boolean is the top of the modelling stack: it certifies each result face against query's
	// analytic oracle, so nothing below it may import boolean.
	"kernel/ops/boolean": {"kernel/brep", "kernel/diag", "kernel/geom", "kernel/meshbool",
		"kernel/mesh", "kernel/ops/internal/probe", "kernel/ops/internal/tol",
		"kernel/ops/query", "kernel/ops/tessellate", "kernel/ops/validate", "kernel/topo", "math"},
	// heal repairs a body on a copy, independent of the modelling stack; surface rebuilds faces
	// and heals what it rebuilt.
	"kernel/ops/heal": {"kernel/brep", "kernel/geom", "kernel/ops/internal/disjoint",
		"kernel/mesh", "kernel/ops/internal/probe", "kernel/ops/internal/retopo",
		"kernel/ops/internal/tol", "kernel/ops/tessellate", "kernel/ops/validate", "kernel/topo", "math"},
	"kernel/ops/surface": {"kernel/brep", "kernel/fit", "kernel/geom", "kernel/ops/heal",
		"kernel/mesh", "kernel/ops/internal/probe", "kernel/ops/internal/retopo",
		"kernel/ops/internal/tol", "kernel/ops/transform", "kernel/ops/validate", "kernel/topo", "math"},
	// The façade: it forwards the boolean enum and entry points, and keeps the general
	// operations that belong to no family (section, shell, split, thicken).
	"kernel/ops": {"kernel/brep", "kernel/diag", "kernel/geom", "kernel/ops/boolean",
		"kernel/mesh", "kernel/ops/internal/probe", "kernel/ops/internal/retopo",
		"kernel/ops/internal/tol", "kernel/ops/query", "kernel/ops/tessellate",
		"kernel/ops/validate", "kernel/topo", "math"},

	// exchange and hlr are CONSUMERS of geometry, so neither may name the kernel/ops façade —
	// that would let either reach any operation (#2195, #2196). Each now depends only on the
	// families it genuinely needs, and both take their VALUE types from kernel/mesh rather than
	// from an operation package. The kernel/ops/tessellate edge is deliberate and is the rule's
	// own doing: the ground rules place the tessellator in that family, and a mesh exporter and
	// an image-space hidden-line engine cannot do their job without it.
	"kernel/exchange": {"api", "kernel/geom", "kernel/mesh", "kernel/ops/query",
		"kernel/ops/tessellate", "kernel/ops/validate", "kernel/subd", "kernel/topo", "math"},
	"kernel/hlr": {"kernel/mesh", "kernel/ops/tessellate", "kernel/topo", "math"},
	// model -> yamlcodec: the domain serializes recipes/materials through the neutral YAML
	// leaf (yamlcodec wraps yaml.v3 and imports nothing first-party), NOT the persistence
	// layer — the B4 (#1615) inversion is gone. Held transitively by TestModelDoesNotImportPersistence.
	"model": {"api", "build", "event", "kernel", "math", "solve", "yamlcodec"},
	"math":  {},
	"solve": {"math"},

	// Application & orchestration.
	"app": {"addincat", "api", "build", "clientgraphics", "command", "event", "kernel",
		"math", "model", "osfont", "persistence", "renderer", "report", "scene", "theme",
		"update", "userconfig", "yamlcodec"},
	"command": {"model"},
	"event":   {},
	// script -> math: the console editor clamps cursor/column indices with the shared
	// math.Clamp instead of re-rolling clampInt locally (G4 #1652).
	"script": {"addin/dispatch", "api", "app", "math"},

	// Persistence & small leaves.
	"persistence": {"api", "model", "userconfig", "yamlcodec"},
	"userconfig":  {},
	// yamlcodec is the neutral YAML leaf (ADR-0020): it wraps gopkg.in/yaml.v3 and imports
	// nothing first-party, so both the domain and persistence may depend on it (B4 #1615).
	"yamlcodec": {},
	"build":     {},
	"crcpost":   {},
	"perf":      {},
	"release":   {},
	"report":    {"crcpost"},
	// theme -> math: blendermap's alpha-scale clamp routes through the shared math.Clamp
	// instead of a hand-rolled (and previously buggy — it never clamped the lower bound)
	// local clamp01 (G15 #2176).
	"theme":      {"api", "math", "persistence", "userconfig", "yamlcodec"},
	"update":     {},
	"usagestats": {"crcpost", "persistence"},
	"addincat":   {"persistence"},
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
	// B10 (#1621, RESOLVED): the router no longer reaches renderer/scene — lighting/
	// environment/camera state crosses the wall as app-owned value types
	// (app.Light/ShadowRig/EnvironmentState/CameraFrame), so renderer and scene are
	// deliberately ABSENT here; re-adding either import fails this test.
	"addin/router": {"addin/modelaccess", "addin/opregistry", "addin/trace", "api", "app",
		"clientgraphics", "event", "kernel", "math", "model", "osfont", "script"},
	"addin/trace": {"api"},

	// Composition roots may reach anything below them. cmd -> test-utilities is the
	// developer test tooling only: cmd/testimpact and cmd/testslowest are thin CLIs over
	// test-utilities/testimpact and /testtiming, which ship in no binary the user runs
	// (architecture/testing/03-test-tiers-and-selection.md).
	"cmd": {"addin/opregistry", "addin/router", "api", "app", "build", "kernel", "math",
		"model", "perf", "persistence", "release", "script", "test-utilities", "update"},

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
		if !edgeAllowed(allowed, dst) {
			t.Errorf("%s imports %s: edge %q -> %q is not in allowedTreeImports — either the "+
				"import is an architecture violation, or the edge is a deliberate decision that "+
				"belongs in the table with a comment (#1623).", pkg, imp, src, dst)
		}
	}
}

// edgeAllowed reports whether dst is covered by the row. A row entry matches the destination
// tree exactly, or as a PARENT of it: "kernel" covers every kernel subtree, so a consumer that
// may depend on the kernel says so once instead of listing the eleven packages it happens to
// reach today. Inside a tree the rows are exact, which is what makes the direction enforceable.
func edgeAllowed(allowed []string, dst string) bool {
	for _, a := range allowed {
		if a == dst || strings.HasPrefix(dst, a+"/") {
			return true
		}
	}
	return false
}

// packageTree maps an import path to its allowlist row: the top-level directory, except
// where a tree's internal direction is itself an architecture decision and one row for the
// whole tree would hide it (#2194).
//
//   - addin/* — the router's allowances must not leak to the other add-in infrastructure.
//   - kernel/* — the ground rules pin the direction geom → topo → brep → ops → model and
//     say archguard enforces it; with one "kernel" row every edge inside the kernel, and
//     every inversion, is invisible.
//   - kernel/ops/* — kernel/ops is packaged by OPERATION (#2183). The split only buys
//     anything while the edges between the families stay declared, so each is its own row.
func packageTree(importPath string) string {
	p := strings.TrimPrefix(importPath, "oblikovati.org/")
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return parts[0]
	}
	if parts[0] == "kernel" && parts[1] == "ops" && len(parts) > 2 {
		if parts[2] == "internal" && len(parts) > 3 {
			return "kernel/ops/internal/" + parts[3]
		}
		return "kernel/ops/" + parts[2]
	}
	if parts[0] == "addin" || parts[0] == "kernel" {
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
	for line := range strings.SplitSeq(strings.TrimSpace(out), "\n") {
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
