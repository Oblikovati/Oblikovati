// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// fallbackDebt is the RATCHET for the CSG-fallback retirement (ADR-0061, #2251). Three numbers
// measure how much of the boolean still leaves the exact analytic pipeline for a triangle soup,
// and all three reach ZERO exactly when the retirement is done:
//
//	faceted-entry-sites   call sites that hand a boolean to a faceted engine
//	mixed-decline-returns named declines out of the mixed per-face boolean (ADR-0058)
//	faceted-engine-files  the non-test source files of the faceted engines themselves
//
// The first two are the doors; the third is the room. Closing the doors without deleting the
// room is a strangler that never strangles ("A generalization is complete only when the special
// cases it replaces are deleted"), so the file count is pinned alongside the call sites.
//
// Like kernelNetDeltaPin this fails on ANY move, up or down. A RISE needs a reason in the PR; a
// FALL means a stage of the retirement landed and the pin comes down with it, in the same commit.
// A floor nobody lowers stops being a floor.
//
// Baseline taken 2026-09-03, before stage 1.
var fallbackDebt = map[string]int{
	"faceted-entry-sites":   5,
	"mixed-decline-returns": 3,
	"faceted-engine-files":  38,
}

// facetedEngines are the functions that produce or adopt a faceted body in place of the exact
// analytic one. A call to any of them is one door out of the exact pipeline.
var facetedEngines = []string{
	"booleanCSG",                 // triangle-soup BSP CSG
	"booleanViaMeshbool",         // exact mesh arrangement (ADR-0052), faceted output
	"reconstructedCurvedBoolean", // mesh reconstruction (ADR-0056 L5)
}

// facetedEngineTrees hold the faceted engines' own sources. Their file counts are what the final
// stage of the retirement deletes.
var facetedEngineTrees = map[string][]string{
	"kernel/meshbool":    nil,                                  // whole package
	"kernel/ops/boolean": {"csg", "meshbool_", "mesh_brep.go"}, // prefixes within a shared package
}

func TestCSGFallbackDebt(t *testing.T) {
	t.Parallel()
	got := map[string]int{
		"faceted-entry-sites":   countFacetedEntrySites(t),
		"mixed-decline-returns": countMixedDeclineReturns(t),
		"faceted-engine-files":  countFacetedEngineFiles(t),
	}
	var moved []string
	keys := make([]string, 0, len(fallbackDebt))
	for k := range fallbackDebt {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		want, have := fallbackDebt[k], got[k]
		if want == have {
			continue
		}
		dir := "ROSE"
		if have < want {
			dir = "fell"
		}
		moved = append(moved, "  "+k+": "+strconv.Itoa(want)+" → "+strconv.Itoa(have)+
			"  ("+dir+" by "+strconv.Itoa(abs(have-want))+")")
	}
	if len(moved) > 0 {
		t.Errorf("CSG-fallback debt moved — update fallbackDebt in this commit and say which stage "+
			"of ADR-0061 moved it. A RISE is a new door out of the exact pipeline and needs a reason; "+
			"a FALL is a stage landing and needs the pin lowered so it keeps holding:\n%s",
			strings.Join(moved, "\n"))
	}
}

// countFacetedEntrySites counts calls to the faceted engines outside their own declarations and
// outside tests: every place a boolean can still leave the analytic pipeline.
func countFacetedEntrySites(t *testing.T) int {
	t.Helper()
	engines := map[string]bool{}
	for _, name := range facetedEngines {
		engines[name] = true
	}
	n := 0
	walkKernelSources(t, func(_ string, f *ast.File) {
		ast.Inspect(f, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok {
				return true
			}
			id, ok := call.Fun.(*ast.Ident)
			if ok && engines[id.Name] {
				n++
			}
			return true
		})
	})
	if n == 0 {
		t.Fatal("counted no faceted entry sites — facetedEngines named functions that no longer " +
			"exist under that name; re-point it or delete this guard because the retirement is done")
	}
	return n
}

// countMixedDeclineReturns counts the returns of ErrUnsupportedMixedBoolean — the mixed per-face
// boolean's named declines (ADR-0058), each one a configuration that routes to a faceted engine.
func countMixedDeclineReturns(t *testing.T) int {
	t.Helper()
	n := 0
	walkKernelSources(t, func(_ string, f *ast.File) {
		ast.Inspect(f, func(node ast.Node) bool {
			ret, ok := node.(*ast.ReturnStmt)
			if !ok {
				return true
			}
			for _, r := range ret.Results {
				if id, ok := r.(*ast.Ident); ok && id.Name == "ErrUnsupportedMixedBoolean" {
					n++
				}
			}
			return true
		})
	})
	return n
}

// countFacetedEngineFiles counts the non-test source files of the faceted engines. Stage 7 of the
// retirement takes this to zero by deleting them.
func countFacetedEngineFiles(t *testing.T) int {
	t.Helper()
	n := 0
	for dir, prefixes := range facetedEngineTrees {
		entries, err := filepath.Glob(filepath.Join("..", dir, "*.go"))
		if err != nil {
			t.Fatalf("globbing %s: %v", dir, err)
		}
		if len(entries) == 0 {
			continue // the tree is gone: that is the retirement, not a broken guard
		}
		for _, p := range entries {
			base := filepath.Base(p)
			if strings.HasSuffix(base, "_test.go") {
				continue
			}
			if prefixes == nil || hasAnyPrefix(base, prefixes) {
				n++
			}
		}
	}
	return n
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// walkKernelSources parses every non-test .go file under kernel/ once and hands it to visit.
func walkKernelSources(t *testing.T, visit func(path string, f *ast.File)) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join("..", "kernel"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		f, perr := parser.ParseFile(fset, p, nil, 0)
		if perr != nil {
			return perr
		}
		visit(p, f)
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel sources: %v", err)
	}
}
