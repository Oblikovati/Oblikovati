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
//
// SCOPE, precisely: the CLOSED-FORM section only. The marched route (geom.TraceSurfaceIntersection,
// curved_crossing_imprint.go) is a second way this package asks geom for a section and is not matched
// here — it already takes a recorder and reports its own degradations, so it carries no silence to
// close (review round 2, N7).
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

// TestEveryMixedImprintDeclineReachesTheRecorder: every path that RETURNS a decline from the mixed
// imprint must reach a recorder — not merely be able to, which is what an earlier version of this rule
// checked and which left the defect one frame up (#3525, review round 2, N2). The signature check was
// green against a bare `return nil, false` inside a pairing and against one inside
// mixedCurvedImprints itself.
//
// The rule is DOMINANCE, the same "name it or forward it" shape kernel/geom's decline guard uses: a
// return whose ok position is the literal `false` must be preceded, in its own block or an enclosing
// one, by either a record*(…) call or a call that hands the recorder on to a callee in this set. The
// set is the transitive closure of mixedCurvedImprints over calls that pass the recorder, so a helper
// a pairing forwards to is held to the same rule and the forward is not a permanent free pass.
//
// WHAT IT CANNOT PROVE, precisely: that the record which dominates a return is the record FOR that
// return. A `return nil, false` appended AFTER an existing forward in the same block passes. Deciding
// that needs the ok value's definition traced to the call that produced it — data flow, not syntax —
// and Go's AST alone does not carry it (it would take go/types plus an SSA pass over each function).
// The dominance form catches every shape a new gate actually takes, because a new decision is written
// where the decision is made.
func TestEveryMixedImprintDeclineReachesTheRecorder(t *testing.T) {
	t.Parallel()
	funcs := declineSubtreeOf(t, brepPackageDir, "mixedCurvedImprints")
	if len(funcs) < 9 {
		t.Fatalf("the decline subtree of mixedCurvedImprints holds %d functions (%v); the guard is not "+
			"reading the code it names and would pass vacuously", len(funcs), sortedNames(funcs))
	}
	checked := 0
	forEachGoFileUnder(t, brepPackageDir, func(path string, file *ast.File) {
		for _, decl := range file.Decls {
			if fn, isFunc := decl.(*ast.FuncDecl); isFunc && fn.Body != nil && funcs[fn.Name.Name] {
				checked++
				reportUndominatedDeclines(t, relToRepo(path), fn)
			}
		}
	})
	t.Logf("checked %d functions in the mixed imprint's decline subtree: %v", checked, sortedNames(funcs))
}

// reportUndominatedDeclines fails once per decline return the function leaves unreported.
func reportUndominatedDeclines(t *testing.T, path string, fn *ast.FuncDecl) {
	t.Helper()
	for range undominatedDeclineReturns(fn, recorderParamName(fn)) {
		t.Errorf("%s: %s returns a decline that no record*() call and no recorder-carrying call "+
			"dominates — the decline cannot reach feature health, the API or the UI (#3525)",
			path, fn.Name.Name)
	}
}

// declineSubtreeOf is mixedCurvedImprints plus every same-package function reachable from it by a call
// that PASSES THE RECORDER — the functions a decline can travel through. Derived, so a pairing or a
// helper added tomorrow joins the set the day it lands.
func declineSubtreeOf(t *testing.T, pkg, root string) map[string]bool {
	t.Helper()
	bodies := funcBodiesIn(t, pkg)
	in := map[string]bool{root: true}
	for grew := true; grew; {
		grew = false
		for name := range in {
			fn, known := bodies[name]
			if !known {
				continue
			}
			for _, callee := range calleesGivenTheRecorder(fn, recorderParamName(fn)) {
				if _, isLocal := bodies[callee]; isLocal && !in[callee] {
					in[callee], grew = true, true
				}
			}
		}
	}
	return in
}

// calleesGivenTheRecorder names the same-package functions this one hands its recorder to.
func calleesGivenTheRecorder(fn *ast.FuncDecl, rec string) []string {
	var out []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall || !callTakesTheRecorder(call, rec) {
			return true
		}
		if id, isIdent := call.Fun.(*ast.Ident); isIdent {
			out = append(out, id.Name)
		}
		return true
	})
	return out
}

// callTakesTheRecorder reports a call passing the recorder identifier (or an explicit nil in its place
// is NOT accepted — a nil recorder is a discard, and a decline routed into one is silent).
func callTakesTheRecorder(call *ast.CallExpr, rec string) bool {
	for _, arg := range call.Args {
		if rec != "" && identNamed(arg, rec) {
			return true
		}
	}
	return false
}

// recorderParamName is the name the function gave its *diag.Recorder parameter, or "" when it has none.
func recorderParamName(fn *ast.FuncDecl) string {
	for _, field := range fn.Type.Params.List {
		star, isStar := field.Type.(*ast.StarExpr)
		if !isStar {
			continue
		}
		sel, isSel := star.X.(*ast.SelectorExpr)
		if isSel && identNamed(sel.X, "diag") && sel.Sel.Name == "Recorder" && len(field.Names) > 0 {
			return field.Names[0].Name
		}
	}
	return ""
}

// undominatedDeclineReturns walks the function's blocks in order and reports every `false` return that
// no record*() call and no recorder-carrying call precedes in scope.
func undominatedDeclineReturns(fn *ast.FuncDecl, rec string) []string {
	ok, found := okResultIndex(fn.Type.Results)
	if !found {
		return nil
	}
	var offenders []string
	walkBlockForDeclines(fn.Body, rec, ok, false, &offenders)
	return offenders
}

// okResultIndex is the position of the bool result a decline is returned in.
func okResultIndex(results *ast.FieldList) (int, bool) {
	if results == nil {
		return 0, false
	}
	for i, name := range flatResultNames(results) {
		if name == "bool" {
			return i, true
		}
	}
	return 0, false
}

// flatResultNames names each result position's type, expanding a grouped field.
func flatResultNames(results *ast.FieldList) []string {
	var out []string
	for _, field := range results.List {
		name := ""
		if id, isIdent := field.Type.(*ast.Ident); isIdent {
			name = id.Name
		}
		for range maxInt(len(field.Names), 1) {
			out = append(out, name)
		}
	}
	return out
}

// walkBlockForDeclines carries "a record or a forward has happened in scope" down the block tree, in
// statement order, and flags a decline return that nothing dominates.
func walkBlockForDeclines(block *ast.BlockStmt, rec string, ok int, covered bool, offenders *[]string) {
	for _, stmt := range block.List {
		if ret, isReturn := stmt.(*ast.ReturnStmt); isReturn {
			if !covered && returnsDecline(ret, ok) {
				*offenders = append(*offenders, "")
			}
			continue
		}
		inner := covered || statementReportsOrForwards(stmt, rec)
		walkNestedBlocks(stmt, rec, ok, inner, offenders)
		covered = inner
	}
}

// walkNestedBlocks recurses into whatever blocks a statement owns, carrying the coverage its own
// condition or initialiser established.
func walkNestedBlocks(stmt ast.Stmt, rec string, ok int, covered bool, offenders *[]string) {
	ast.Inspect(stmt, func(n ast.Node) bool {
		block, isBlock := n.(*ast.BlockStmt)
		if !isBlock || block == stmt {
			return true
		}
		walkBlockForDeclines(block, rec, ok, covered, offenders)
		return false
	})
}

// statementReportsOrForwards reports a statement that records a decline itself, or hands the recorder
// to a callee that can. Either way a decline returned after it is on the record.
func statementReportsOrForwards(stmt ast.Stmt, rec string) bool {
	reports := false
	ast.Inspect(stmt, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if id, isIdent := call.Fun.(*ast.Ident); isIdent && strings.HasPrefix(id.Name, "record") {
			reports = true
		}
		if callTakesTheRecorder(call, rec) {
			reports = true
		}
		return true
	})
	return reports
}

// returnsDecline reports a return whose ok position is the literal false.
func returnsDecline(ret *ast.ReturnStmt, ok int) bool {
	return len(ret.Results) > ok && isFalseLiteral(ret.Results[ok])
}

// isFalseLiteral reports the untyped `false` written straight into a return.
func isFalseLiteral(expr ast.Expr) bool {
	id, isIdent := expr.(*ast.Ident)
	return isIdent && id.Name == "false"
}

// funcBodiesIn indexes the package's function declarations by name.
func funcBodiesIn(t *testing.T, pkg string) map[string]*ast.FuncDecl {
	t.Helper()
	out := map[string]*ast.FuncDecl{}
	forEachGoFileUnder(t, pkg, func(_ string, file *ast.File) {
		for _, decl := range file.Decls {
			if fn, isFunc := decl.(*ast.FuncDecl); isFunc && fn.Body != nil {
				out[fn.Name.Name] = fn
			}
		}
	})
	return out
}

// sortedNames renders a name set in one explicit order, for a stable log line.
func sortedNames(set map[string]bool) []string {
	out := make([]string, 0, len(set))
	for name := range set {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// relToRepo trims the archguard-relative prefix so a message names a repo path.
func relToRepo(path string) string { return strings.TrimPrefix(filepath.ToSlash(path), "../") }

// maxInt is max for ints, spelled out because the builtin needs a type parameter here.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
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
