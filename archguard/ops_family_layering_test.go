// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"slices"
	"sort"
	"strings"
	"testing"
)

// kernel/ops is packaged BY OPERATION, not by case (#2183): boolean, blend, tessellate,
// validate, heal, query, surface and transform, over the shared substrate in internal/.
// The split only buys anything — a change to one family that neither rebuilds nor re-tests
// the others — while the edges between the families stay declared. An undeclared edge is how
// a split silently collapses back into one package, so every one is a row here.
//
// Reading a row: the key may import the values, and nothing else under kernel/ops. Go's own
// cycle rule forbids the reverse edges, so this table is what pins the DIRECTION.
var allowedOpsFamilyImports = map[string][]string{
	// The substrate leaves. probe answers read-only geometric questions and is the bottom;
	// mesh and retopo build on it; tol and disjoint stand alone.
	"internal/probe": {},
	// tol reads mesh.Tri for the triangle-soup weld scale (ForTris).
	"internal/tol":      {"internal/mesh"},
	"internal/disjoint": {},
	"internal/mesh":     {"internal/probe"},
	"internal/retopo":   {"internal/mesh", "internal/probe", "internal/tol"},

	// validate is the post-condition every other family runs, so it sits lowest.
	"validate": {"internal/mesh", "internal/probe"},
	// tessellate is a DERIVED view of the B-rep, below everything that models with it.
	"tessellate": {"internal/mesh", "internal/probe", "validate"},
	// transform moves and re-surfaces a body; it models nothing.
	"transform": {"internal/mesh", "internal/probe", "internal/retopo", "internal/tol", "validate"},
	// blend is the one fillet/chamfer/draft engine (ADR-0050/0051).
	"blend": {"internal/mesh", "internal/probe", "internal/retopo", "internal/tol", "tessellate", "transform", "validate"},
	// query asks read-only questions of a body; it reads blend's edge convexity.
	"query": {"internal/disjoint", "internal/mesh", "internal/probe", "internal/tol", "blend", "tessellate"},
	// boolean is the top of the modelling stack: it certifies its result with query's
	// per-face analytic oracle, so nothing below it may import boolean.
	"boolean": {"internal/mesh", "internal/probe", "internal/tol", "query", "tessellate", "validate"},
	// heal repairs a body on a copy; it is independent of the modelling stack.
	"heal": {"internal/disjoint", "internal/mesh", "internal/probe", "internal/retopo", "internal/tol", "tessellate", "validate"},
	// surface rebuilds faces and heals what it rebuilt.
	"surface": {"internal/mesh", "internal/probe", "internal/retopo", "internal/tol", "heal", "transform", "validate"},
}

const opsPrefix = "oblikovati.org/kernel/ops"

// TestOpsFamilyImportsAreDeclared fails on any import between kernel/ops families that
// allowedOpsFamilyImports does not list, and on any family importing the kernel/ops façade
// (which would be a cycle the moment the façade forwards to it).
func TestOpsFamilyImportsAreDeclared(t *testing.T) {
	t.Parallel()
	for pkg, imports := range firstPartyImports(t) {
		fam, ok := opsFamilyOf(pkg)
		if !ok {
			continue
		}
		allowed, declared := allowedOpsFamilyImports[fam]
		if !declared {
			t.Errorf("kernel/ops family %q has no row in allowedOpsFamilyImports — add one "+
				"stating what it may depend on", fam)
			continue
		}
		for _, imp := range imports {
			dep, isOps := opsFamilyOf(imp)
			if !isOps || dep == fam {
				continue
			}
			if imp == opsPrefix {
				t.Errorf("kernel/ops/%s imports the kernel/ops façade — the façade forwards to "+
					"the families, so this edge inverts the layering", fam)
				continue
			}
			if !slices.Contains(allowed, dep) {
				t.Errorf("undeclared kernel/ops edge %s -> %s; allowed for %s: %v — add the row "+
					"deliberately or move the shared code down into kernel/ops/internal",
					fam, dep, fam, allowed)
			}
		}
	}
}

// TestOpsFamilyRowsAreLive fails on a row naming a family or a dependency that no longer
// exists, so the table cannot drift into describing an architecture the code left behind.
func TestOpsFamilyRowsAreLive(t *testing.T) {
	t.Parallel()
	live := map[string]bool{}
	for pkg := range firstPartyImports(t) {
		if fam, ok := opsFamilyOf(pkg); ok {
			live[fam] = true
		}
	}
	var stale []string
	for fam, deps := range allowedOpsFamilyImports {
		if !live[fam] {
			stale = append(stale, fam)
		}
		for _, d := range deps {
			if !live[d] {
				stale = append(stale, fam+" -> "+d)
			}
		}
	}
	sort.Strings(stale)
	if len(stale) > 0 {
		t.Errorf("allowedOpsFamilyImports names packages that do not exist: %v", stale)
	}
}

// opsFamilyOf maps a package path to its kernel/ops family ("blend", "internal/mesh"), and
// reports false for anything outside kernel/ops. The façade itself is not a family.
func opsFamilyOf(pkg string) (string, bool) {
	if pkg == opsPrefix || !strings.HasPrefix(pkg, opsPrefix+"/") {
		return "", false
	}
	return strings.TrimPrefix(pkg, opsPrefix+"/"), true
}
