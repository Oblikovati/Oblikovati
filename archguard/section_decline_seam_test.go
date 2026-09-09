// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// Two rules that keep Oblikovati/Oblikovati#3525 fixed, both derived from the source.
//
// The first is the SEAM: kernel/brep asks kernel/geom for a face pair's section through exactly one
// function (curvedImprint), so a refusal has one place to be named. Before #3525 six sites called the
// intersector directly and threw the reason away, which is how a torus pair, an ill-conditioned lane
// and a section that does not close all reached the user as one generic message. A comment asserting
// that invariant is worth nothing without a check — that was review round 1's finding 4.
//
// The second is the FRAME the AST guard in kernel/geom cannot see. That guard reads the return sites of
// functions whose signature carries a refusal; a pairing that declines the mixed boolean WITHOUT
// carrying one — a bare `return nil, false` — is the same defect one frame up, and it is the frame the
// next contributor touches. Every pairing mixedCurvedImprints combines must therefore be able to
// report: it must take a *diag.Recorder.

// sectionSeamFile is the ONE file in kernel/brep allowed to call geom's analytic intersector, and
// brepPackageDir is that package as this test sees it (archguard runs one directory below the root).
const (
	sectionSeamFile = "kernel/brep/curved_boolean_imprint.go"
	brepPackageDir  = "../kernel/brep"
)

// TestBrepAsksGeomForASectionThroughOneSeam: every kernel/brep call to geom.IntersectSurfacesAnalytic*
// goes through curved_boolean_imprint.go, so the reason a pair refused is named in one place instead
// of being dropped in six.
func TestBrepAsksGeomForASectionThroughOneSeam(t *testing.T) {
	t.Parallel()
	var offenders []string
	forEachGoFileUnder(t, brepPackageDir, func(path string, file *ast.File) {
		rel := strings.TrimPrefix(filepath.ToSlash(path), "../")
		if rel == sectionSeamFile {
			return
		}
		for _, name := range analyticIntersectorCalls(file) {
			offenders = append(offenders, rel+": "+name)
		}
	})
	sort.Strings(offenders)
	if len(offenders) > 0 {
		t.Errorf("kernel/brep must reach geom's analytic intersector only through %s, so a refusal is named "+
			"once instead of dropped at each call (#3525):\n  %s", sectionSeamFile, strings.Join(offenders, "\n  "))
	}
}

// analyticIntersectorCalls names every geom.IntersectSurfacesAnalytic* call in a file.
func analyticIntersectorCalls(file *ast.File) []string {
	var out []string
	ast.Inspect(file, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if sel, isSel := call.Fun.(*ast.SelectorExpr); isSel &&
			identNamed(sel.X, "geom") && strings.HasPrefix(sel.Sel.Name, "IntersectSurfacesAnalytic") {
			out = append(out, "geom."+sel.Sel.Name)
		}
		return true
	})
	return out
}

// TestEveryMixedImprintPairingCanReport: each function whose result mixedCurvedImprints ANDs into its
// own ok must take a *diag.Recorder. A pairing that cannot report declines the whole boolean in
// silence, which is #3525 one frame above the guard that catches it (review round 1, finding 5).
func TestEveryMixedImprintPairingCanReport(t *testing.T) {
	t.Parallel()
	pairings := pairingsCombinedBy(t, brepPackageDir, "mixedCurvedImprints")
	if len(pairings) < 5 {
		t.Fatalf("found %d pairings in mixedCurvedImprints (%v); the guard is not reading the function it "+
			"names and would pass vacuously", len(pairings), pairings)
	}
	recorderTakers := funcsTakingARecorder(t, brepPackageDir)
	for _, name := range pairings {
		if !recorderTakers[name] {
			t.Errorf("%s contributes to mixedCurvedImprints's ok but takes no *diag.Recorder, so its decline "+
				"cannot reach feature health, the API or the UI (#3525)", name)
		}
	}
	t.Logf("checked %d imprint pairings: %v", len(pairings), pairings)
}

// pairingsCombinedBy returns the names of the functions called inside fnName — the pairings whose ok
// results it combines. Derived from the AST, so a pairing added tomorrow is covered the day it lands.
func pairingsCombinedBy(t *testing.T, pkg, fnName string) []string {
	t.Helper()
	seen := map[string]bool{}
	forEachGoFileUnder(t, pkg, func(_ string, file *ast.File) {
		for _, decl := range file.Decls {
			fn, isFunc := decl.(*ast.FuncDecl)
			if !isFunc || fn.Name.Name != fnName || fn.Body == nil {
				continue
			}
			for _, name := range localCallNames(fn.Body) {
				seen[name] = true
			}
		}
	})
	out := make([]string, 0, len(seen))
	for name := range seen {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// localCallNames lists the same-package functions a body calls (a bare identifier callee), skipping
// builtins that take no part in the decline.
func localCallNames(body *ast.BlockStmt) []string {
	var out []string
	ast.Inspect(body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if id, isIdent := call.Fun.(*ast.Ident); isIdent && !goBuiltinNames[id.Name] {
			out = append(out, id.Name)
		}
		return true
	})
	return out
}

// goBuiltinNames are the callee identifiers that are not package functions.
var goBuiltinNames = map[string]bool{
	"append": true, "cap": true, "clear": true, "close": true, "copy": true, "delete": true,
	"len": true, "make": true, "max": true, "min": true, "new": true, "panic": true, "print": true,
	"println": true, "recover": true,
}

// funcsTakingARecorder is the set of functions in the package that take a *diag.Recorder parameter.
func funcsTakingARecorder(t *testing.T, pkg string) map[string]bool {
	t.Helper()
	out := map[string]bool{}
	forEachGoFileUnder(t, pkg, func(_ string, file *ast.File) {
		for _, decl := range file.Decls {
			if fn, isFunc := decl.(*ast.FuncDecl); isFunc && takesARecorder(fn.Type.Params) {
				out[fn.Name.Name] = true
			}
		}
	})
	return out
}

// takesARecorder reports a parameter list carrying a *diag.Recorder.
func takesARecorder(params *ast.FieldList) bool {
	for _, field := range params.List {
		star, isStar := field.Type.(*ast.StarExpr)
		if !isStar {
			continue
		}
		if sel, isSel := star.X.(*ast.SelectorExpr); isSel && identNamed(sel.X, "diag") && sel.Sel.Name == "Recorder" {
			return true
		}
	}
	return false
}

// identNamed reports an expression that is exactly the named identifier.
func identNamed(expr ast.Expr, name string) bool {
	id, isIdent := expr.(*ast.Ident)
	return isIdent && id.Name == name
}

// forEachGoFileUnder parses every non-test .go file under dir and hands it to visit.
func forEachGoFileUnder(t *testing.T, dir string, visit func(path string, file *ast.File)) {
	t.Helper()
	fset := token.NewFileSet()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		parsed, perr := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if perr != nil {
			return perr
		}
		visit(path, parsed)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", dir, err)
	}
}
