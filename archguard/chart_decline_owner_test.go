// SPDX-License-Identifier: GPL-2.0-only

package archguard

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// "Never degrade silently. A fallback, approximation, or dropped element is a diag.Defect that
// reaches feature health, the API, and the UI." (kernel ground rules.)
//
// Oblikovati/Oblikovati#3520 moved the chart-driven mesher's decline report off its three callers
// and onto ONE recorder: chartFaceMesh records into a *chartDeclineLog, and the curved-face router
// owns the log and stamps it onto whatever mesh the face ships instead.
//
// That shape only holds while the ROUTER owns the only log. Go cannot enforce it — chartDeclineLog
// is an ordinary package-level struct with a usable zero value, so any file in kernel/ops/tessellate
// can write chartFaceMesh(f, s, q, &chartDeclineLog{}), drop the answer, and fall back in silence.
// A reviewer built exactly that fourth caller against the first cut of #3520: it compiled, and the
// whole archguard suite stayed green. The defect the issue named would have come straight back in a
// new shape that the compiler, the linter and this package all accepted.
//
// So the convention is made checkable here instead. It is the same bargain the other guards in this
// package strike: the language cannot express "only this file", so the guard says it and a probe
// proves the guard bites.

// chartDeclineLogOwner is the ONE non-test file allowed to construct a chartDeclineLog. Everything
// else must take one as a parameter, which is what puts the decline on the router's log rather than
// on a private one nobody reads.
const chartDeclineLogOwner = "kernel/ops/tessellate/tessellate_trim.go"

// chartDeclineLogPackage is the package the type lives in; a construction outside it cannot compile,
// so the guard only has to walk this directory.
const chartDeclineLogPackage = "../kernel/ops/tessellate"

// TestOnlyTheCurvedFaceRouterOwnsAChartDeclineLog fails THREE ways, and each one is planted by a probe
// (TestTheOwnershipGuardBitesEveryConstructionForm): a second production file constructing a log, the
// owner no longer constructing one, and the owner no longer STAMPING it.
//
// The third is not decoration. Construction and stamping are different facts, and the first cut of this
// guard claimed the second while checking only the first — a reviewer deleted the recordOn call and the
// guard stayed green. A router that builds a log and never stamps it reports nothing, which is the exact
// defect #3520 removed.
func TestOnlyTheCurvedFaceRouterOwnsAChartDeclineLog(t *testing.T) {
	t.Parallel()
	owners := chartDeclineLogConstructors(t)
	sort.Strings(owners)
	if len(owners) == 0 {
		t.Fatalf("no production file constructs a chartDeclineLog; %s is meant to own the only one, so "+
			"either the router stopped taking one or the type was renamed and this guard now checks "+
			"nothing", chartDeclineLogOwner)
	}
	if len(owners) > 1 || owners[0] != chartDeclineLogOwner {
		t.Errorf("%d production file(s) construct a chartDeclineLog (%s); exactly one may — %s, the "+
			"curved-face router. A second log is a caller reporting into a slot nobody stamps, which is "+
			"the silent fallback #3520 removed: take the router's log as a parameter instead",
			len(owners), strings.Join(owners, ", "), chartDeclineLogOwner)
		return
	}
	if !stampsTheDecline(t, filepath.Join(chartDeclineLogPackage, filepath.Base(chartDeclineLogOwner))) {
		t.Errorf("%s constructs a chartDeclineLog and never calls recordOn on it; a log nobody stamps "+
			"reports nothing, which is the silent fallback #3520 removed", chartDeclineLogOwner)
	}
}

// TestTheOwnershipGuardBitesEveryConstructionForm plants each shape the guard must catch, against the
// guard's own matcher, so "it cannot be evaded" is a measured claim and not a hope. The first cut
// matched only the brace form; `new(chartDeclineLog)` and a `var` declaration both compiled and left it
// green, which is a guard calibrated to the last reviewer rather than to the language.
func TestTheOwnershipGuardBitesEveryConstructionForm(t *testing.T) {
	t.Parallel()
	for _, row := range []struct{ name, src string }{
		{"composite literal", "package p\nfunc f() { g(&chartDeclineLog{}) }"},
		{"new()", "package p\nfunc f() { g(new(chartDeclineLog)) }"},
		{"var declaration", "package p\nfunc f() { var l chartDeclineLog; g(&l) }"},
		{"package-level var", "package p\nvar l chartDeclineLog"},
	} {
		if !sourceBuildsAChartDeclineLog(t, row.src) {
			t.Errorf("the guard does not see a log built as a %s; that form evades it", row.name)
		}
	}
	// A PARAMETER is how every legitimate caller takes the router's log, so it must stay legal — a guard
	// that flagged it would forbid the very shape the design depends on.
	if sourceBuildsAChartDeclineLog(t, "package p\nfunc f(log *chartDeclineLog) { _ = log }") {
		t.Error("the guard flags a *chartDeclineLog PARAMETER; taking the router's log is the whole design")
	}
	if sourceBuildsAChartDeclineLog(t, "package p\n// chartDeclineLog is discussed at length here.\nfunc f() {}") {
		t.Error("the guard flags a mention in a COMMENT; chart_decline.go discusses the type at length")
	}
}

// chartDeclineLogConstructors is every non-test file under the tessellate package that builds a
// chartDeclineLog composite literal, as repo-relative paths.
func chartDeclineLogConstructors(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(chartDeclineLogPackage)
	if err != nil {
		t.Fatalf("read %s: %v", chartDeclineLogPackage, err)
	}
	var out []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if buildsAChartDeclineLog(t, filepath.Join(chartDeclineLogPackage, name)) {
			out = append(out, "kernel/ops/tessellate/"+name)
		}
	}
	return out
}

// buildsAChartDeclineLog reports whether the file constructs a chartDeclineLog, in ANY of the three
// shapes Go offers. It reads the AST rather than the text so a mention in a comment — this guard's own
// subject is discussed at length in chart_decline.go — is not a construction.
func buildsAChartDeclineLog(t *testing.T, path string) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return fileBuildsAChartDeclineLog(file)
}

// sourceBuildsAChartDeclineLog runs the matcher over a source string, so the probe rows can plant each
// shape without writing files into the package under guard.
func sourceBuildsAChartDeclineLog(t *testing.T, src string) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), "probe.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse probe: %v", err)
	}
	return fileBuildsAChartDeclineLog(file)
}

// fileBuildsAChartDeclineLog is the matcher itself: a composite literal of the type, a new() of it, or a
// var declaration of it — the three ways Go makes a value. A *chartDeclineLog PARAMETER is deliberately
// NOT a construction (it is an *ast.Field, never a ValueSpec), because taking the router's log is how
// every legitimate caller works.
func fileBuildsAChartDeclineLog(file *ast.File) bool {
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.CompositeLit:
			found = found || isChartDeclineLogIdent(v.Type)
		case *ast.ValueSpec:
			found = found || isChartDeclineLogIdent(v.Type)
		case *ast.CallExpr:
			found = found || isNewOfChartDeclineLog(v)
		}
		return !found
	})
	return found
}

// isNewOfChartDeclineLog matches `new(chartDeclineLog)` — a CallExpr, which no composite-literal matcher
// ever sees.
func isNewOfChartDeclineLog(call *ast.CallExpr) bool {
	fn, isIdent := call.Fun.(*ast.Ident)
	return isIdent && fn.Name == "new" && len(call.Args) == 1 && isChartDeclineLogIdent(call.Args[0])
}

// isChartDeclineLogIdent reports whether an expression is the bare type name.
func isChartDeclineLogIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "chartDeclineLog"
}

// stampsTheDecline reports whether the owner file calls recordOn — the step that puts a pending decline
// onto the mesh the face ships. Constructing a log and never stamping it reports nothing.
func stampsTheDecline(t *testing.T, path string) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		call, isCall := n.(*ast.CallExpr)
		if !isCall {
			return true
		}
		if sel, isSel := call.Fun.(*ast.SelectorExpr); isSel && sel.Sel.Name == "recordOn" {
			found = true
		}
		return !found
	})
	return found
}
