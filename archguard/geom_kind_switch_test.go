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

// "Behaviour many operations need (SSI, offset, closest point, seam/periodicity, curvature
// bounds) is a method on geom.Surface/geom.Curve. Type switches on geometry kinds live only in
// kernel/geom." (kernel ground rules.)
//
// A type switch on a geometry kind outside kernel/geom is how the N-squared special-case
// explosion starts: adding a surface type means finding every switch that must learn about it,
// and the ones nobody finds silently take their default branch. The method form has the opposite
// property — a new type cannot compile until it answers.
//
// Nothing measured this (#2188). It is 786 switch cases across 22 packages, so this is a
// ratchet, and it is per PACKAGE rather than per file: 257 files would be a list nobody reads,
// and the unit that should shrink is the package that owns the behaviour.
//
// kernel/ops/blend at 400 is over half the total, and it is the same debt ADR-0050's strangler
// migration is meant to retire — the analytic fillet catalog dispatching on host surface kind.
// It shrinks when that engine does.

// geomSwitchDebt is the per-package cap on type switches and assertions naming a geom.* type,
// outside kernel/geom itself. Baseline 2026-09-01. It may only shrink.
//
// Two entries are boundary mappers where a kind switch may be irreducible — kernel/geomapi
// (the API surface) and kernel/exchange/step/geommap (STEP entity mapping) both translate
// geometry into a foreign vocabulary, which a method on geom.Surface cannot do without teaching
// the kernel that vocabulary. They are capped rather than exempted: if the right answer is an
// exemption, that is an ADR, not a silent hole in a guard.
var geomSwitchDebt = map[string]int{
	"addin/opregistry":             1,
	"addin/router":                 5,
	"app":                          13,
	"kernel/blend":                 15,
	"kernel/brep":                  102,
	"kernel/exchange/step/geommap": 11,
	"kernel/geomapi":               9,
	"kernel/ops":                   1,
	"kernel/ops/blend":             400,
	"kernel/ops/boolean":           22,
	"kernel/ops/heal":              12,
	"kernel/ops/internal/probe":    3,
	"kernel/ops/surface":           21,
	"kernel/ops/tessellate":        53,
	"kernel/ops/transform":         4,
	"kernel/ops/validate":          9,
	"kernel/topo":                  3,
	"model/assembly":               4,
	"model/compdef":                1,
	"model/drawing":                20,
	"model/feature":                50,
	"model/sketch":                 15,
}

func TestGeometryKindSwitchesLiveInGeom(t *testing.T) {
	t.Parallel()
	got := scanGeomSwitches(t)
	var rose, fell, stale []string
	for pkg, n := range got {
		switch owed := geomSwitchDebt[pkg]; {
		case n > owed:
			rose = append(rose, pkg+": "+strconv.Itoa(n)+" geom type switch case(s)/assertion(s), budget "+strconv.Itoa(owed))
		case n < owed:
			fell = append(fell, `"`+pkg+`": `+strconv.Itoa(n)+",")
		}
	}
	for pkg := range geomSwitchDebt {
		if _, ok := got[pkg]; !ok {
			stale = append(stale, pkg)
		}
	}
	sort.Strings(rose)
	sort.Strings(fell)
	sort.Strings(stale)
	if len(rose) > 0 {
		t.Errorf("a type switch on a GEOMETRY KIND grew outside kernel/geom — adding a surface "+
			"type then means finding every switch that must learn about it, and the ones nobody "+
			"finds take their default branch silently. Put the behaviour on geom.Surface/geom.Curve "+
			"instead (#2188):\n%s", strings.Join(rose, "\n"))
	}
	if len(fell) > 0 {
		t.Errorf("geom-switch debt FELL — good; lower these geomSwitchDebt entries so the ratchet "+
			"holds the new floor:\n%s", strings.Join(fell, "\n"))
	}
	if len(stale) > 0 {
		t.Errorf("these packages no longer switch on a geometry kind — DELETE their entries:\n  %s",
			strings.Join(stale, "\n  "))
	}
}

// scanGeomSwitches counts, per package, the type-switch cases and type assertions naming a
// geom.* type. kernel/geom is skipped: it is the one place the rules allow them.
func scanGeomSwitches(t *testing.T) map[string]int {
	t.Helper()
	fset := token.NewFileSet()
	got := map[string]int{}
	for _, root := range geomSwitchRoots {
		err := filepath.WalkDir(filepath.Join("..", root), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
				return err
			}
			rel := filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(p), "../"))
			if strings.HasPrefix(rel, "kernel/geom/") {
				return nil
			}
			f, err := parser.ParseFile(fset, p, nil, 0)
			if err != nil {
				t.Fatalf("parsing %s: %v", p, err)
			}
			pkg := filepath.ToSlash(filepath.Dir(rel))
			got[pkg] += countGeomSwitches(f)
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
	for pkg, n := range got {
		if n == 0 {
			delete(got, pkg)
		}
	}
	return got
}

// geomSwitchRoots are the first-party trees that consume geometry. head/ is a separate module
// and is covered by its own build.
var geomSwitchRoots = []string{"kernel", "model", "app", "addin", "renderer", "scene", "command"}

// countGeomSwitches returns the number of geom.* type-switch cases and type assertions in f.
func countGeomSwitches(f *ast.File) int {
	n := 0
	ast.Inspect(f, func(node ast.Node) bool {
		switch t := node.(type) {
		case *ast.TypeAssertExpr:
			if t.Type != nil && namesGeomType(t.Type) {
				n++
			}
		case *ast.TypeSwitchStmt:
			for _, c := range t.Body.List {
				cc, ok := c.(*ast.CaseClause)
				if !ok {
					continue
				}
				for _, e := range cc.List {
					if namesGeomType(e) {
						n++
					}
				}
			}
		}
		return true
	})
	return n
}

// namesGeomType reports whether an expression names a type in package geom, pointer or value.
func namesGeomType(e ast.Expr) bool {
	if s, ok := e.(*ast.StarExpr); ok {
		e = s.X
	}
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "geom"
}
