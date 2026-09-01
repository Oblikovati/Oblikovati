// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// "Output is byte-identical across runs and platforms: explicit total orders for every tie-break,
// no map-iteration or pointer/hash ordering" (kernel ground rules).
//
// Go randomises map iteration deliberately, so `for k := range m { out = append(out, ...) }`
// produces a different slice on every run. When that slice feeds naming, a loop walk or a
// serialized recipe, the same input builds a different body twice — the failure mode that is
// hardest to reproduce and easiest to mistake for a geometry bug.
//
// The hazard is already known and handled AD HOC: assignProvNames (kernel/topo/provenance.go)
// keeps a parallel `order []string` slice precisely so it never ranges its map, and
// buildSharedEdges sorts explicitly. Nothing required either (#2192).
//
// Not every site is a defect — appending into a set that is sorted afterwards, or accumulating a
// count, is fine. So a site can be cleared two ways: remove the map-order dependence, or annotate
// the `range` line `// order:<why>` the way a justified tolerance carries `// tol:<kind>`. An
// annotation must state why the order cannot reach the output; "it is probably fine" is not a
// reason.

// mapOrderDebt counts, per kernel file, the range-over-map loops that append and carry no
// `// order:` justification. Baseline 2026-09-01: 26 files, 32 sites. It may only shrink.
var mapOrderDebt = map[string]int{
	"brep/arrange2d.go":                          1,
	"brep/boolean_orient.go":                     1,
	"brep/boolean_provenance.go":                 1,
	"brep/boolean_radial_edge.go":                2,
	"brep/boolean_stitch.go":                     1,
	"brep/curved_reorient.go":                    1,
	"brep/reconstruct_dissolve.go":               1,
	"brep/unify_coplanar.go":                     2,
	"mesh/cage.go":                               1,
	"meshbool/merge.go":                          2,
	"ops/blend/fillet_corner_classify.go":        2,
	"ops/blend/fillet_corner_partial.go":         1,
	"ops/blend/fillet_corner_setback.go":         1,
	"ops/blend/fillet_corner_setback_unified.go": 1,
	"ops/blend/fillet_corner_torus.go":           1,
	"ops/blend/fillet_faces.go":                  1,
	"ops/blend/fillet_farend_chain.go":           1,
	"ops/blend/fillet_orient.go":                 1,
	"ops/blend/fillet_setback.go":                1,
	"ops/heal/stitch.go":                         2,
	"ops/internal/retopo/retopo.go":              1,
	"ops/tessellate/mesh_orient.go":              1,
	"ops/tessellate/orient_heal.go":              1,
	"ops/tessellate/union_holes.go":              2,
	"ops/validate/fold_repair.go":                1,
	"subd/subdivide.go":                          1,
}

func TestNoMapIterationOrderedOutput(t *testing.T) {
	t.Parallel()
	got := scanMapOrderDebt(t)
	var rose, fell, stale []string
	for path, n := range got {
		switch owed := mapOrderDebt[path]; {
		case n > owed:
			rose = append(rose, path+": "+strconv.Itoa(n)+" map-ordered append(s), budget "+strconv.Itoa(owed))
		case n < owed:
			fell = append(fell, `"`+path+`": `+strconv.Itoa(n)+",")
		}
	}
	for path := range mapOrderDebt {
		if _, ok := got[path]; !ok {
			stale = append(stale, path)
		}
	}
	sort.Strings(rose)
	sort.Strings(fell)
	sort.Strings(stale)
	if len(rose) > 0 {
		t.Errorf("output built by ranging a MAP — Go randomises that order, so the same input "+
			"builds a different body twice. Collect the keys and sort them, keep a parallel order "+
			"slice (see assignProvNames), or annotate the range line `// order:<why>` if the order "+
			"cannot reach the output (#2192):\n%s", strings.Join(rose, "\n"))
	}
	if len(fell) > 0 {
		t.Errorf("map-order debt FELL — good; lower these mapOrderDebt entries so the ratchet "+
			"holds the new floor:\n%s", strings.Join(fell, "\n"))
	}
	if len(stale) > 0 {
		t.Errorf("these mapOrderDebt files no longer range a map into an append — DELETE their "+
			"entries. If ALL of them are listed here, the type checker resolved nothing instead: "+
			"run `go build ./...` so the export data this guard reads exists.\n  %s",
			strings.Join(stale, "\n  "))
	}
}

// scanMapOrderDebt type-checks each kernel package and counts range-over-map loops whose body
// appends. Type information is required — "is this a map?" is not a syntactic question — and it
// comes from compiler export data, which is why this costs about a second rather than the twenty
// a from-source check takes.
func scanMapOrderDebt(t *testing.T) map[string]int {
	t.Helper()
	fset := token.NewFileSet()
	got := map[string]int{}
	root := filepath.Join("..", "kernel")
	var dirs []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() {
			dirs = append(dirs, p)
		}
		return err
	})
	if err != nil {
		t.Fatalf("walking kernel/: %v", err)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, parser.ParseComments)
		if err != nil {
			continue
		}
		for _, pkg := range pkgs {
			files := make([]*ast.File, 0, len(pkg.Files))
			names := make([]string, 0, len(pkg.Files))
			for n := range pkg.Files {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				files = append(files, pkg.Files[n])
			}
			info := &types.Info{Types: map[ast.Expr]types.TypeAndValue{}}
			conf := types.Config{Importer: importer.Default(), Error: func(error) {}}
			_, _ = conf.Check(dir, fset, files, info)
			for _, f := range files {
				countMapOrderedAppends(fset, f, info, got)
			}
		}
	}
	return got
}

// countMapOrderedAppends tallies, per file, the range-over-map loops that append without an
// `// order:` justification on the range line.
func countMapOrderedAppends(fset *token.FileSet, f *ast.File, info *types.Info, got map[string]int) {
	ast.Inspect(f, func(n ast.Node) bool {
		rs, ok := n.(*ast.RangeStmt)
		if !ok {
			return true
		}
		tv, ok := info.Types[rs.X]
		if !ok || tv.Type == nil {
			return true
		}
		if _, isMap := tv.Type.Underlying().(*types.Map); !isMap {
			return true
		}
		if !appendsInBlock(rs.Body) {
			return true
		}
		pos := fset.Position(rs.Pos())
		if justifiedOrder(pos.Filename, pos.Line) {
			return true
		}
		rel, err := filepath.Rel(filepath.Join("..", "kernel"), pos.Filename)
		if err != nil {
			return true
		}
		got[filepath.ToSlash(rel)]++
		return true
	})
}

// appendsInBlock reports whether the block calls append.
func appendsInBlock(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if id, ok := c.Fun.(*ast.Ident); ok && id.Name == "append" {
				found = true
			}
		}
		return true
	})
	return found
}

// justifiedOrder reports whether the range line carries an `// order:` justification.
func justifiedOrder(file string, line int) bool {
	src, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	lines := strings.Split(string(src), "\n")
	if line-1 < 0 || line-1 >= len(lines) {
		return false
	}
	return strings.Contains(lines[line-1], "// order:")
}
