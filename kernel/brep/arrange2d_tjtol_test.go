// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
	stdmath "math"
	"strings"
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The T-junction pass reads its tolerance in TWO classes — a perpendicular distance to the edge and
// a bound on the parameter along it — and #3513 (ADR-0061 G9) is that it used one absolute for both.
// These rows pin each class separately, so the two cannot silently merge again.

// TestTheEndExclusionIsALengthOnAShortEdge is the regression row for #3513. The vertex sits 5e-8 from
// the edge's START — inside the 1e-7 on-edge tolerance, so it is that endpoint, not an interior
// T-junction — but the edge is only 1e-5 long, so in PARAMETER it sits at t = 5e-3. Compared against
// a bare 1e-7 t-pad it looked interior and the pass split there, then split the shorter half again,
// which is the churn tjSplitBudget had to bound. Converted through |dP/dt| = |ab| the pad is 1e-2 and
// the vertex is correctly refused.
func TestTheEndExclusionIsALengthOnAShortEdge(t *testing.T) {
	t.Parallel()
	pts := []math.Point2{math.P2(0, 0), math.P2(1e-5, 0), math.P2(5e-8, 0)}
	if got := onEdgeInterior(pts, 0, 1); got != -1 {
		t.Errorf("a vertex 5e-8 along a 1e-5 edge is within the on-edge tolerance of its START, so it "+
			"is that endpoint, not an interior T-junction; vertexOnEdgeInterior returned %d", got)
	}
}

// TestARealTJunctionOnAShortEdgeStillSplits is the other half of the row above: the end exclusion may
// not swallow the edge's interior. The same 1e-5 edge, with the vertex at its MIDPOINT and 5e-8 off
// the line, is a T-junction and must be found.
func TestARealTJunctionOnAShortEdgeStillSplits(t *testing.T) {
	t.Parallel()
	pts := []math.Point2{math.P2(0, 0), math.P2(1e-5, 0), math.P2(5e-6, 5e-8)}
	if got := onEdgeInterior(pts, 0, 1); got != 2 {
		t.Errorf("a vertex at the midpoint of a 1e-5 edge, 5e-8 off the line, is a T-junction; "+
			"vertexOnEdgeInterior returned %d, want 2", got)
	}
}

// TestANearMissIsNotATJunction pins the distance class: 5e-6 off a unit edge is 50x the on-edge
// tolerance and must never split it.
func TestANearMissIsNotATJunction(t *testing.T) {
	t.Parallel()
	pts := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(0.5, 5e-6)}
	if got := onEdgeInterior(pts, 0, 1); got != -1 {
		t.Errorf("a vertex 5e-6 off a unit edge is a near miss, not a T-junction; got %d", got)
	}
}

// onEdgeInterior runs vertexOnEdgeInterior the way splitTJunctions does: over the vertex grid, with
// the on-edge tolerance taken from the point set's own 2D extent.
func onEdgeInterior(pts []math.Point2, a, b int) int {
	return vertexOnEdgeInterior(pts, a, b, newVertexCullGrid(pts), tjOnEdgeTol(geom.ResolutionForPoints2D(pts)))
}

// TestANaNVertexIsNotOnAnEdge pins the one input on which the extracted onEdgeInteriorAt diverges from
// the shape it replaced. The inlined original rejected on `dist > tol`, so a NaN distance fell through
// and the vertex was ACCEPTED; the helper asks `dist <= tol`, so it declines. A point whose distance to
// the edge is not a number cannot be shown to lie on it, and splitting there would cut an edge at a
// place the pass knows nothing about. Every finite input is unaffected (review round 2, N3).
func TestANaNVertexIsNotOnAnEdge(t *testing.T) {
	t.Parallel()
	pa, pb := math.P2(0, 0), math.P2(1, 0)
	ab := pa.VectorTo(pb)
	lenSq := ab.LengthSquared()
	nan := stdmath.NaN()
	if onEdgeInteriorAt(pa, ab, lenSq, math.P2(0.5, math.Scalar(nan)), 1e-7, 1e-7) {
		t.Error("a vertex at a NaN offset must not count as lying on the edge")
	}
	// The finite control at the same place, so the row cannot pass by rejecting everything.
	if !onEdgeInteriorAt(pa, ab, lenSq, math.P2(0.5, 5e-8), 1e-7, 1e-7) {
		t.Error("the finite control at 5e-8 off a unit edge must still qualify")
	}
}

// TestTheOnEdgeToleranceScalesUpAndIsFloored: the distance reads geom.Resolution.Plane — the member
// whose whole definition is "how far a point may sit from a segment and still count as on it" — so it
// grows with the arrangement, and it is floored at tjTol because the welder grid it must absorb is an
// absolute under #1399 and would otherwise overtake it on a sub-unit arrangement.
func TestTheOnEdgeToleranceScalesUpAndIsFloored(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		size    float64
		floored bool
	}{
		{1e-6, true}, {0.01, true}, {1, true}, {100, false}, {1e4, false},
	} {
		res := geom.ResolutionForSize(tc.size)
		want := res.Plane()
		if tc.floored {
			want = tjTol
		}
		if got := tjOnEdgeTol(res); got != want {
			t.Errorf("tjOnEdgeTol at size %g = %g, want %g (floored=%v)", tc.size, got, want, tc.floored)
		}
	}
	// At size 1 the two coincide EXACTLY: tjTol is 100 x arrTol and Plane is 100 x Weld, which is the
	// same ratio. That is why the floor reproduces the pre-#3513 behaviour on a unit-scale part.
	if geom.ResolutionForSize(1).Plane() != tjTol {
		t.Errorf("Plane at size 1 (%g) must equal the tjTol floor (%g)", geom.ResolutionForSize(1).Plane(), tjTol)
	}
	if tjTol <= arrTol {
		t.Errorf("the on-edge floor %g must stay above the welder grid %g, or a welded vertex could "+
			"sit off its host edge and never count as on it", tjTol, arrTol)
	}
}

// TestOnlyAPairAddingSplitIsChargedToTheBudget pins the bound's MECHANISM, which no live input
// reaches any more (#3513 took the kernel corpus from 42 budget exhaustions in 24694 arrangements to
// 0). The budget is the pass's termination argument, so it has to stay tested on its own terms.
func TestOnlyAPairAddingSplitIsChargedToTheBudget(t *testing.T) {
	t.Parallel()
	adding := map[[2]int]bool{{0, 1}: true}
	budget := 5
	if !splitEdgeAt(adding, [2]int{0, 1}, 2, &budget) || budget != 4 {
		t.Errorf("a split whose halves are both new must be charged once; budget %d, want 4", budget)
	}
	if adding[[2]int{0, 1}] || !adding[[2]int{0, 2}] || !adding[[2]int{1, 2}] {
		t.Errorf("the split must replace the edge with its two halves; got %v", adding)
	}
	free := map[[2]int]bool{{0, 1}: true, {0, 2}: true, {1, 2}: true}
	budget = 5
	if !splitEdgeAt(free, [2]int{0, 1}, 2, &budget) || budget != 5 {
		t.Errorf("a split that adds neither half strictly shrinks the set and is free; budget %d, want 5", budget)
	}
}

// TestTheSplitBudgetRefusesWhenExhausted: the charge is what stops the loop, so the exhausted case
// must return false rather than run on.
func TestTheSplitBudgetRefusesWhenExhausted(t *testing.T) {
	t.Parallel()
	edges := map[[2]int]bool{{0, 1}: true}
	budget := 0
	if splitEdgeAt(edges, [2]int{0, 1}, 2, &budget) {
		t.Error("a pair-adding split with no budget left must refuse, not keep splitting")
	}
	if !edges[[2]int{0, 1}] {
		t.Error("a refused split must leave the edge set untouched for the caller to abandon")
	}
}

// TestTheDeclineNamesTheSplitThatMadeIt: there are four sites that arrange a face and they fail for
// the same reason, so a diagnostic that did not distinguish them would send the reader to the wrong
// one (ADR-0061 stage 6, review round 3). The RING drill used to assert this end to end; it converges
// now (#3513), so the property is pinned here instead.
func TestTheDeclineNamesTheSplitThatMadeIt(t *testing.T) {
	t.Parallel()
	rec := &diag.Recorder{}
	recordArrangementDecline(rec, siteClosedSurfaceTrim, unconvergedArrangement(7))
	if !rec.Has(CodeArrangementUnconverged) {
		t.Fatalf("the decline must reach the diagnostic channel; got %v", rec.Records())
	}
	detail := rec.Records()[0].Detail
	if !strings.Contains(detail, "closed-surface") || !strings.Contains(detail, "7 segments") {
		t.Errorf("the decline must name the split and the size of the input that failed; got %q", detail)
	}
}

// TestAnUnrelatedErrorIsNotRecordedAsAnUnconvergedArrangement keeps the report specific: the sites
// call recordArrangementDecline with whatever trimByImprint returned, and every other refusal has its
// own name.
func TestAnUnrelatedErrorIsNotRecordedAsAnUnconvergedArrangement(t *testing.T) {
	t.Parallel()
	rec := &diag.Recorder{}
	recordArrangementDecline(rec, siteWallTrim, errors.New("brep: some other refusal"))
	if len(rec.Records()) != 0 {
		t.Errorf("only an unconverged arrangement may be recorded as one; got %v", rec.Records())
	}
}
