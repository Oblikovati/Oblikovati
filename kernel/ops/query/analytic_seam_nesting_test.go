// SPDX-License-Identifier: GPL-2.0-only

package query

import (
	"math"
	"testing"
)

// uvRing is a rectangular closed uv polyline, the shape both a band's trim and a small hole take in
// these tests.
func uvRing(uLo, uHi, vLo, vHi float64) []arcSample {
	return []arcSample{{u: uLo, v: vLo}, {u: uHi, v: vLo}, {u: uHi, v: vHi}, {u: uLo, v: vHi}}
}

// TestHoleStraddlingTheSeamIsInsideItsFace pins the nesting bug that made a wrapped emboss ADD its
// footprint to the host cone (Oblikovati/Oblikovati#3505, the straddling sibling of the #3489 drilled-wall
// case). The face occupies a whole period, u ∈ [−2π, 0]; the hole hangs off BOTH ends of that
// interval, at u ∈ [−0.03, 0.03], so no whole-period shift can bring it inside and a one-branch
// even-odd test reads it as a top-level loop.
func TestHoleStraddlingTheSeamIsInsideItsFace(t *testing.T) {
	t.Parallel()
	outer := uvRing(-2*math.Pi, 0, 10, 20)
	hole := uvRing(-0.03, 0.03, 14, 16)
	per := uvPeriod{u: 2 * math.Pi}

	if !pointInLoopPolygon(outer, hole[0].u, hole[0].v, per) {
		t.Errorf("the hole's first sample (%g, %g) reads as OUTSIDE the face loop spanning u ∈ [−2π, 0]; "+
			"on a periodic parameter it is the same point as u = %g, which is inside",
			hole[0].u, hole[0].v, hole[0].u-2*math.Pi)
	}
	if loopDepthIsEven([][]arcSample{outer, hole}, 1, per) {
		t.Error("the seam-straddling hole came back at EVEN nesting depth, so its measure would be " +
			"ADDED to the face instead of subtracted")
	}
}

// TestNonPeriodicNestingIsUnchanged holds the planar case fixed: with no period there is only the
// untranslated probe, so a loop genuinely outside another stays outside.
func TestNonPeriodicNestingIsUnchanged(t *testing.T) {
	t.Parallel()
	left, right := uvRing(0, 1, 0, 1), uvRing(2, 3, 0, 1)
	if pointInLoopPolygon(left, right[0].u, right[0].v, uvPeriod{}) {
		t.Error("a disjoint loop reads as contained on a non-periodic surface")
	}
	if !loopDepthIsEven([][]arcSample{left, right}, 1, uvPeriod{}) {
		t.Error("two disjoint top-level loops must both sit at even depth and both ADD")
	}
}

// TestBranchOffsetsOnlyListsTranslatesThatCanHit keeps the translate search bounded and exact: a
// non-periodic parameter offers the identity alone, and a periodic one offers only the offsets that
// land the probe inside the polygon's own extent.
func TestBranchOffsetsOnlyListsTranslatesThatCanHit(t *testing.T) {
	t.Parallel()
	if got := branchOffsets(0.5, paramRange{0, 10}, 0); len(got) != 1 || got[0] != 0 {
		t.Errorf("branchOffsets with period 0 = %v, want the single identity offset", got)
	}
	got := branchOffsets(0.03, paramRange{-2 * math.Pi, 0}, 2*math.Pi)
	if len(got) != 1 || math.Abs(got[0]+2*math.Pi) > 1e-12 {
		t.Errorf("branchOffsets(0.03, [−2π, 0], 2π) = %v, want the single offset −2π", got)
	}
	if got := branchOffsets(0.5, paramRange{-1000, 1000}, 2*math.Pi); len(got) != branchOffsetCap {
		t.Errorf("branchOffsets over a 1000-period extent returned %d offsets, want the cap %d — the "+
			"predicate must stay O(1) whatever a malformed loop's parametric extent is", len(got), branchOffsetCap)
	}
}
