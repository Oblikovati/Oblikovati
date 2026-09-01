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
// Nothing checked this (#2190). 45 exported operations that return a *topo.Body never reach
// Validate — notably the whole surface family (17) and the whole transform family (8), neither
// of which validates anything it builds.
//
// "Reaches Validate" is computed transitively: through same-package calls and through a
// qualified call into a sibling kernel/ops family, so a facade forwarder counts as validating
// when what it forwards to does. Without that, ops.Boolean and ops.Shell would read as
// violations when the packages behind them validate properly.

// validateDebt is the per-package count of exported functions returning *topo.Body that never
// reach Validate. Baseline 2026-09-01. It may only shrink.
var validateDebt = map[string]int{
	// The facade's own operations: silhouette wires and the two face splits.
	"kernel/ops": 3,
	// DraftFaces/DraftFacesNeutral and the two cylinder-fillet entry points.
	"kernel/ops/blend": 4,
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

// reachesValidate reports whether f calls something that validates, in its own package or in a
// sibling ops family (the facade-forwarder case).
func reachesValidate(f *opFunc, validates map[string]bool) bool {
	found := false
	ast.Inspect(f.decl.Body, func(n ast.Node) bool {
		c, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		switch fun := c.Fun.(type) {
		case *ast.Ident:
			if validates[f.dir+"::"+fun.Name] {
				found = true
			}
		case *ast.SelectorExpr:
			id, ok := fun.X.(*ast.Ident)
			if !ok {
				return true
			}
			for k := range validates {
				dir, name, _ := strings.Cut(k, "::")
				if name == fun.Sel.Name && path.Base(dir) == id.Name {
					found = true
				}
			}
		}
		return true
	})
	return found
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
