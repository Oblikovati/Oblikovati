// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// "Validate is a post-condition of every public kernel operation. An invalid body is an error,
// never a return value." (kernel ground rules.)
//
// An operation that returns an invalid body without saying so moves the failure downstream: the
// next operation consumes it, and the defect surfaces as a crash or a wrong result in code that
// did nothing wrong. Validate at the exit is what keeps a failure local to the operation that
// caused it.
//
// Nothing checked this (#2190). 54 exported operations that return a *topo.Body never reach
// Validate — notably the whole surface family (17) and the whole transform family (8), neither
// of which validates anything it builds, and boolean.Facet (#3329), whose faceted cage the
// planar boolean then trusts as a valid operand.
//
// "Reaches Validate" follows only the calls whose RESULT IS RETURNED — `return join(...)`,
// `return boolean.Boolean(...)` — because the thing that must be validated is the thing the
// caller gets back. Following every call instead credits validation that did not happen: an
// early version of this guard reported boolean.Facet clean because it calls
// tessellate.TessellateBody, which validates the MESH it builds on the way. Validating something
// en route is not validating the result.
//
// That rule keeps the real delegation chains honest — ops.Boolean returns
// BooleanWithDiagnostics, which returns join/cut/intersect, which return booleanGeneral, which
// does validate — without extending credit to every caller of a validating helper.

// validateDebt is the per-package count of exported functions returning *topo.Body that never
// reach Validate. Baseline 2026-09-01. It may only shrink.
var validateDebt = map[string]int{
	// The facade's own operations, plus its Facet/MeshToBRep forwarders to boolean.
	"kernel/ops": 6,
	// Draft, the two cylinder-fillet entry points, and all four FilletEdges* entries.
	"kernel/ops/blend": 8,
	// Facet and MeshToBRep: both BUILD a body from a mesh and hand it back unchecked. Facet is
	// #3329 — the faceted cage is what the planar boolean then trusts, so an invalid cage is a
	// defect laundered into a valid-looking operand.
	"kernel/ops/boolean": 2,
	// Healing builds a body FROM a broken one, so its post-condition matters most of all: a
	// repair that leaves the body invalid has done nothing but hide the original defect.
	"kernel/ops/heal":            8,
	"kernel/ops/internal/retopo": 3,
	"kernel/ops/query":           2,
	// The whole surface family. Every one of these returns a body built from new geometry.
	"kernel/ops/surface": 17,
	// The whole transform family — deform, move/rotate faces, replace a face's surface.
	"kernel/ops/transform": 8,
}

func TestExportedOpsValidateTheirResult(t *testing.T) {
	t.Parallel()
	got := scanValidateDebt(t)
	var rose, fell, stale []string
	for pkg, names := range got {
		switch owed := validateDebt[pkg]; {
		case len(names) > owed:
			sort.Strings(names)
			rose = append(rose, pkg+": "+strconv.Itoa(len(names))+" un-validated export(s), budget "+
				strconv.Itoa(owed)+" — "+strings.Join(names, ", "))
		case len(names) < owed:
			fell = append(fell, `"`+pkg+`": `+strconv.Itoa(len(names))+",")
		}
	}
	for pkg := range validateDebt {
		if _, ok := got[pkg]; !ok {
			stale = append(stale, pkg)
		}
	}
	sort.Strings(rose)
	sort.Strings(fell)
	sort.Strings(stale)
	if len(rose) > 0 {
		t.Errorf("an exported operation returns a *topo.Body it never validates — an invalid body "+
			"is an error, not a return value, and one returned silently surfaces later as a crash "+
			"in code that did nothing wrong (#2190):\n%s", strings.Join(rose, "\n"))
	}
	if len(fell) > 0 {
		t.Errorf("validate debt FELL — good; lower these validateDebt entries so the ratchet holds "+
			"the new floor:\n%s", strings.Join(fell, "\n"))
	}
	if len(stale) > 0 {
		t.Errorf("these packages now validate every exported body-returning operation — DELETE "+
			"their entries:\n  %s", strings.Join(stale, "\n  "))
	}
}

// opFunc is one top-level function in a kernel/ops package, keyed by "<pkg dir>::<name>".
type opFunc struct {
	dir  string
	name string
	decl *ast.FuncDecl
}

// scanValidateDebt returns, per package, the exported body-returning functions that never reach
// Validate.
func scanValidateDebt(t *testing.T) map[string][]string {
	t.Helper()
	all := parseOpsFuncs(t)
	validates := map[string]bool{}
	for k, f := range all {
		if mentionsValidate(f.decl.Body) {
			validates[k] = true
		}
	}
	for changed := true; changed; {
		changed = false
		for k, f := range all {
			if !validates[k] && reachesValidate(f, validates) {
				validates[k] = true
				changed = true
			}
		}
	}
	out := map[string][]string{}
	for k, f := range all {
		if ast.IsExported(f.name) && returnsTopoBody(f.decl.Type.Results) && !validates[k] {
			out[f.dir] = append(out[f.dir], f.name)
		}
	}
	return out
}

// parseOpsFuncs parses every kernel/ops package and returns its top-level functions.
func parseOpsFuncs(t *testing.T) map[string]*opFunc {
	t.Helper()
	fset := token.NewFileSet()
	all := map[string]*opFunc{}
	root := filepath.Join("..", "kernel", "ops")
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return err
		}
		pkgs, perr := parser.ParseDir(fset, p, func(fi os.FileInfo) bool {
			return !strings.HasSuffix(fi.Name(), "_test.go")
		}, 0)
		if perr != nil {
			return nil
		}
		dir := filepath.ToSlash(strings.TrimPrefix(filepath.ToSlash(p), "../"))
		for _, pkg := range pkgs {
			for _, f := range pkg.Files {
				for _, decl := range f.Decls {
					fd, ok := decl.(*ast.FuncDecl)
					if ok && fd.Body != nil && fd.Recv == nil {
						all[dir+"::"+fd.Name.Name] = &opFunc{dir, fd.Name.Name, fd}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking kernel/ops: %v", err)
	}
	if len(all) == 0 {
		t.Fatal("parsed no functions under kernel/ops — the guard would pass vacuously")
	}
	return all
}

// reachesValidate reports whether f returns the result of a call that itself counts as
// validating, in its own package or in a sibling ops family. Only RETURNED calls are followed —
// see the package comment for why crediting every call is wrong.
func reachesValidate(f *opFunc, validates map[string]bool) bool {
	for _, call := range returnedCalls(f.decl.Body) {
		switch fun := call.Fun.(type) {
		case *ast.Ident:
			if validates[f.dir+"::"+fun.Name] {
				return true
			}
		case *ast.SelectorExpr:
			id, ok := fun.X.(*ast.Ident)
			if !ok {
				continue
			}
			for k := range validates {
				dir, name, _ := strings.Cut(k, "::")
				if name == fun.Sel.Name && path.Base(dir) == id.Name {
					return true
				}
			}
		}
	}
	return false
}

// returnedCalls collects the top-level call in each returned expression: the value the caller
// actually receives, not every call made on the way to computing it.
func returnedCalls(b *ast.BlockStmt) []*ast.CallExpr {
	var out []*ast.CallExpr
	ast.Inspect(b, func(n ast.Node) bool {
		ret, ok := n.(*ast.ReturnStmt)
		if !ok {
			return true
		}
		for _, r := range ret.Results {
			if call, ok := r.(*ast.CallExpr); ok {
				out = append(out, call)
			}
		}
		return true
	})
	return out
}

// returnsTopoBody reports whether the result list contains a *topo.Body.
func returnsTopoBody(r *ast.FieldList) bool {
	if r == nil {
		return false
	}
	for _, f := range r.List {
		s, ok := f.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		if sel, ok := s.X.(*ast.SelectorExpr); ok && sel.Sel.Name == "Body" {
			return true
		}
	}
	return false
}

// mentionsValidate reports whether a body names Validate or ValidSolid.
func mentionsValidate(b *ast.BlockStmt) bool {
	found := false
	ast.Inspect(b, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.Ident:
			if strings.Contains(t.Name, "alidate") || strings.Contains(t.Name, "ValidSolid") {
				found = true
			}
		case *ast.SelectorExpr:
			if strings.Contains(t.Sel.Name, "alidate") || strings.Contains(t.Sel.Name, "ValidSolid") {
				found = true
			}
		}
		return true
	})
	return found
}
