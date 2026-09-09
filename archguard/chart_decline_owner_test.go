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

// TestOnlyTheCurvedFaceRouterOwnsAChartDeclineLog fails BOTH ways: on a second production file
// constructing a log (the silent-caller regression), and on the owner no longer constructing one
// (which would mean the router stopped stamping, and the guard would otherwise pass vacuously).
func TestOnlyTheCurvedFaceRouterOwnsAChartDeclineLog(t *testing.T) {
	t.Parallel()
	owners := chartDeclineLogConstructors(t)
	sort.Strings(owners)
	if len(owners) == 1 && owners[0] == chartDeclineLogOwner {
		return
	}
	if len(owners) == 0 {
		t.Fatalf("no production file constructs a chartDeclineLog; %s is meant to own the only one, "+
			"so either the router stopped stamping the decline or the type was renamed and this guard "+
			"now checks nothing", chartDeclineLogOwner)
	}
	t.Errorf("%d production files construct a chartDeclineLog (%s); exactly one may — %s, the "+
		"curved-face router. A second log is a caller reporting into a slot nobody stamps, which is "+
		"the silent fallback #3520 removed: take the router's log as a parameter instead",
		len(owners), strings.Join(owners, ", "), chartDeclineLogOwner)
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

// buildsAChartDeclineLog reports whether the file holds a `chartDeclineLog{...}` composite literal.
// It reads the AST rather than the text so a mention in a comment — this guard's own subject is
// discussed at length in chart_decline.go — is not a construction.
func buildsAChartDeclineLog(t *testing.T, path string) bool {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	found := false
	ast.Inspect(file, func(n ast.Node) bool {
		lit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if id, isIdent := lit.Type.(*ast.Ident); isIdent && id.Name == "chartDeclineLog" {
			found = true
		}
		return !found
	})
	return found
}
