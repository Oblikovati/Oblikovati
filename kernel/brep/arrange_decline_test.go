// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"os"
	"regexp"
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
func TestEveryArrangingSplitReportsANonConvergentArrangement(t *testing.T) {
	t.Parallel()
	calls := regexp.MustCompile(`(?m)^\s*(?:[\w, ]+:?=\s*)?trimByImprint\(|^\s*(?:[\w, ]+:?=\s*)?splitFace\(`)
	for _, f := range productionGoFiles(t) {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		lines := strings.Split(string(src), "\n")
		for i, l := range lines {
			if !calls.MatchString(l) {
				continue
			}
			if !declineWithin(lines, i, 8) {
				t.Errorf("%s:%d arranges a face (%s) without a recordArrangementDecline within 8 lines: "+
					"an unconverged arrangement there would surface only as a generic refusal, naming nothing",
					f, i+1, strings.TrimSpace(l))
			}
		}
	}
}

// declineWithin reports whether a recordArrangementDecline (or an explicit propagation of the named
// error) appears within n lines after the call — the shape every site uses.
func declineWithin(lines []string, at, n int) bool {
	for i := at; i < len(lines) && i <= at+n; i++ {
		if strings.Contains(lines[i], "recordArrangementDecline") || strings.Contains(lines[i], "unconvergedArrangement(") {
			return true
		}
	}
	return false
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
// the canonical undirected index-pair set the pass grows, so it is what a run can spend and still be
// making progress.
func TestTheSplitBudgetIsTheEdgePairSetSize(t *testing.T) {
	t.Parallel()
	for _, n := range []int{0, 1, 2, 10, 100} {
		if got, want := tjSplitBudget(n), n*(n-1)/2; got != want {
			t.Errorf("tjSplitBudget(%d) = %d, want %d (the number of distinct unordered pairs)", n, got, want)
		}
	}
}
