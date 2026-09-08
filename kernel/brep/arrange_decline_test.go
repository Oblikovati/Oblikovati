// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"

	"oblikovati.org/math"
)

// Every split that arranges a face must REPORT a non-converging arrangement, not just return
// ok=false. Round 2 of the stage-6 review threaded the recorder into one of the three trimByImprint
// sites and left the other two returning a bare `if err != nil { return nil, false }`, so the same
// defect surfaced there only as the generic ErrUnmodelledBoolean, naming nothing.
//
// A geometric corpus row can only ever cover the ONE path its fixture happens to take (the RING drill
// takes the closed-surface trim), so this is the guard that covers the rest and, more importantly,
// covers the split that has not been written yet: it fails when a new call site appears without a
// decline beside it.
//
// It reads the AST, not the text (review round 4): the round-3 regex matched only `x, _, err :=
// trimByImprint(` and missed `return trimByImprint(...)`, a selector target and a nested call, and
// its "decline within 8 lines" was a substring scan a COMMENT could satisfy. Here a call is any
// *ast.CallExpr whose callee is trimByImprint or splitFace, the decline must be a CallExpr in the
// same function body, and the parser drops comments before either is looked for.
func TestEveryArrangingSplitReportsANonConvergentArrangement(t *testing.T) {
	t.Parallel()
	var found int
	for _, f := range productionGoFiles(t) {
		unguarded, calls := unguardedArrangingCalls(t, f, nil)
		found += calls
		for _, site := range unguarded {
			t.Errorf("%s arranges a face without a recordArrangementDecline call (or a returned "+
				"unconvergedArrangement) in its function body: an unconverged arrangement there would "+
				"surface only as a generic refusal, naming nothing", site)
		}
	}
	if found == 0 {
		t.Fatal("the scanner found no trimByImprint/splitFace call in the package; the guard is vacuous")
	}
}

// unguardedArrangingCalls parses one Go source (from src, or from filename when src is nil) and
// returns the position of every trimByImprint / splitFace call whose enclosing function body carries
// neither a recordArrangementDecline call nor a returned unconvergedArrangement call, plus the total
// number of arranging calls seen so a caller can tell an empty answer from a vacuous scan.
func unguardedArrangingCalls(t *testing.T, filename string, src []byte) ([]string, int) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, parserSource(src), 0)
	if err != nil {
		t.Fatalf("parsing %s: %v", filename, err)
	}
	var unguarded []string
	total := 0
	for _, d := range file.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		calls, declined := arrangingCallsInBody(fn.Body)
		total += len(calls)
		if declined {
			continue
		}
		for _, c := range calls {
			unguarded = append(unguarded, fset.Position(c.Pos()).String())
		}
	}
	return unguarded, total
}

// parserSource keeps a nil slice nil: a nil []byte boxed as `any` is a non-nil interface holding an
// empty source, and the parser would read THAT instead of the file.
func parserSource(src []byte) any {
	if src == nil {
		return nil
	}
	return src
}

// arrangingCallsInBody collects the arranging calls in one function body and reports whether the
// body declines by name: a recordArrangementDecline call, or — for a function that carries no
// recorder, like the public imprint entry — an unconvergedArrangement call returned to its caller.
func arrangingCallsInBody(body *ast.BlockStmt) (calls []*ast.CallExpr, declined bool) {
	ast.Inspect(body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ReturnStmt:
			declined = declined || returnsUnconvergedArrangement(node)
		case *ast.CallExpr:
			switch calleeName(node) {
			case "trimByImprint", "splitFace":
				calls = append(calls, node)
			case "recordArrangementDecline":
				declined = true
			}
		}
		return true
	})
	return calls, declined
}

// returnsUnconvergedArrangement reports whether a return statement hands the named refusal up.
func returnsUnconvergedArrangement(ret *ast.ReturnStmt) bool {
	for _, r := range ret.Results {
		if call, ok := r.(*ast.CallExpr); ok && calleeName(call) == "unconvergedArrangement" {
			return true
		}
	}
	return false
}

// calleeName is the bare identifier a call invokes, or "" for anything that is not a plain
// function call (method calls, function literals, conversions).
func calleeName(call *ast.CallExpr) string {
	if id, ok := call.Fun.(*ast.Ident); ok {
		return id.Name
	}
	return ""
}

// productionGoFiles lists this package's non-test .go files.
func productionGoFiles(t *testing.T) []string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading the package directory: %v", err)
	}
	var out []string
	for _, e := range entries {
		if n := e.Name(); strings.HasSuffix(n, ".go") && !strings.HasSuffix(n, "_test.go") {
			out = append(out, n)
		}
	}
	return out
}

// TestArrangeCheckedReportsConvergenceOnAnOrdinarySet pins the ordinary answer: a well-conditioned
// square converges and yields its one cell, so ok=true is not vacuous (a predicate that never
// answered true would pass the guard above while breaking every boolean in the system).
func TestArrangeCheckedReportsConvergenceOnAnOrdinarySet(t *testing.T) {
	t.Parallel()
	corners := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(0, 4)}
	segs := make([][2]math.Point2, len(corners))
	for i := range corners {
		segs[i] = [2]math.Point2{corners[i], corners[(i+1)%len(corners)]}
	}
	cells, ok := ArrangeChecked(segs)
	if !ok || len(cells) != 1 {
		t.Fatalf("a plain square must converge to one cell; got %d cells ok=%v", len(cells), ok)
	}
}

// TestTheSplitBudgetIsTheEdgePairSetSize pins the bound's ARGUMENT, not a number: it is the size of
// the canonical undirected index-pair set the pass draws its edges from, so a run that spends more
// pair-adding splits than that has re-added a pair it already removed.
func TestTheSplitBudgetIsTheEdgePairSetSize(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 1, 2, 10, 100} {
		if got, want := tjSplitBudget(n), n*(n-1)/2; got != want {
			t.Errorf("tjSplitBudget(%d) = %d, want %d (the number of distinct unordered pairs)", n, got, want)
		}
	}
}

// TestTheRefusalCountsSegmentsNotFaces: booleanOnce reported len(impA)+len(impB) — a FACE count —
// in a message that says "segments" (review round 4). The count is now the segments themselves.
func TestTheRefusalCountsSegmentsNotFaces(t *testing.T) {
	t.Parallel()
	seg := [2]math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0)}
	perFace := [][][2]math.Point3{{seg, seg, seg}, nil, {seg}}
	if got := imprintSegmentCount(perFace); got != 4 {
		t.Errorf("imprintSegmentCount over 3 faces holding 3+0+1 segments = %d, want 4", got)
	}
	if got := imprintSegmentCount(nil); got != 0 {
		t.Errorf("imprintSegmentCount(nil) = %d, want 0", got)
	}
}

// TestAConvergingArrangementArrangesIdenticallyEveryRun: the T-junction pass walks a sorted snapshot
// of the edge set, so an input whose chains END on other segments' interiors — the comb below, nine
// teeth standing on one spine — arranges to the same cells and the same verdict on every run. Walking
// the live map in its random order made the pair-adding count, and so decline-versus-converge near
// the budget, a run-to-run coin toss (final fix wave, finding 7).
func TestAConvergingArrangementArrangesIdenticallyEveryRun(t *testing.T) {
	t.Parallel()
	segs := combSegments()
	first, firstOK := arrangementFingerprint(segs)
	if !firstOK {
		t.Fatal("the comb must converge; it is the CONVERGING row")
	}
	for run := 1; run < 20; run++ {
		if got, ok := arrangementFingerprint(segs); got != first || ok != firstOK {
			t.Fatalf("run %d arranged differently from run 0:\n%s\nvs\n%s", run, got, first)
		}
	}
}

// combSegments is a spine with nine teeth whose feet land on the spine's INTERIOR, closed by a rail
// across their tips — every tooth foot is a T-junction the pass must weld shut.
func combSegments() [][2]math.Point2 {
	segs := [][2]math.Point2{{math.P2(0, 0), math.P2(10, 0)}, {math.P2(0, 1), math.P2(10, 1)},
		{math.P2(0, 0), math.P2(0, 1)}, {math.P2(10, 0), math.P2(10, 1)}}
	for i := 1; i < 10; i++ {
		x := float64(i)
		segs = append(segs, [2]math.Point2{math.P2(x, 0), math.P2(x, 1)})
	}
	return segs
}

// arrangementFingerprint is the arrangement's cells printed in full, and its verdict.
func arrangementFingerprint(segs [][2]math.Point2) (string, bool) {
	cells, ok := ArrangeChecked(segs)
	return fmt.Sprintf("%v", cells), ok
}
