// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"
	"sort"
	"testing"

	"oblikovati.org/math"
)

// TestOpenAnchorArcsIsSortedWithinCutAndDeduped is the ladder-construction invariant every
// downstream station march depends on: openAnchorArcs must return a STRICTLY increasing,
// bounds-respecting sequence — a duplicate or out-of-order arc coordinate would place two
// section planes at (or past) the same station, which geom.MarchCanalEdgeStationsSeeded's
// warm-start continuation cannot recover from.
func TestOpenAnchorArcsIsSortedWithinCutAndDeduped(t *testing.T) {
	t.Parallel()
	length := 100.0
	cut := [2]float64{-10, 110} // both-ends prolong past the edge
	arcs := openAnchorArcs(length, cut, 8)
	if !sort.Float64sAreSorted(arcs) {
		t.Fatalf("openAnchorArcs not sorted: %v", arcs)
	}
	for i, a := range arcs {
		if a < cut[0]-1e-9 || a > cut[1]+1e-9 {
			t.Errorf("arcs[%d] = %v outside cut %v", i, a, cut)
		}
	}
	for i := 1; i < len(arcs); i++ {
		if arcs[i]-arcs[i-1] <= 0 {
			t.Fatalf("arcs[%d]=%v not strictly greater than arcs[%d]=%v (duplicate/non-monotonic)",
				i, arcs[i], i-1, arcs[i-1])
		}
	}
	// The exact cut bounds must be present (prolong runs to exactly the retained window).
	if stdmath.Abs(arcs[0]-cut[0]) > 1e-9 {
		t.Errorf("first arc = %v, want cut[0] = %v", arcs[0], cut[0])
	}
	if stdmath.Abs(arcs[len(arcs)-1]-cut[1]) > 1e-9 {
		t.Errorf("last arc = %v, want cut[1] = %v", arcs[len(arcs)-1], cut[1])
	}
}

// TestOpenAnchorArcsNoCutClampsToEdgeSpan proves the degenerate case (cut == the edge's own
// span, no prolong either side): the ladder must stay within [0, length] exactly — no prolong
// runs leak in when there is nothing to prolong into.
func TestOpenAnchorArcsNoCutClampsToEdgeSpan(t *testing.T) {
	t.Parallel()
	length := 20.0
	cut := [2]float64{0, length}
	arcs := openAnchorArcs(length, cut, 4)
	if arcs[0] != 0 {
		t.Errorf("first arc = %v, want 0", arcs[0])
	}
	if got := arcs[len(arcs)-1]; stdmath.Abs(got-length) > 1e-9 {
		t.Errorf("last arc = %v, want %v", got, length)
	}
}

// TestDedupArcsDropsNearDuplicates is the direct unit proof for dedupArcs: two ladder values
// within tol collapse to one, and the survivor is the FIRST of the pair (deterministic, so the
// station march never depends on map/slice iteration order).
func TestDedupArcsDropsNearDuplicates(t *testing.T) {
	t.Parallel()
	in := []float64{0, 1, 1.0000001, 2, 2.5, 2.5000002, 5}
	out := dedupArcs(in, 1e-4)
	want := []float64{0, 1, 2, 2.5, 5}
	if len(out) != len(want) {
		t.Fatalf("dedupArcs(%v) = %v, want %v", in, out, want)
	}
	for i := range want {
		if stdmath.Abs(out[i]-want[i]) > 1e-9 {
			t.Errorf("dedupArcs[%d] = %v, want %v", i, out[i], want[i])
		}
	}
}

// TestClampArcsToCutDropsOutOfRange is the direct unit proof for clampArcsToCut: values
// strictly outside [cut[0], cut[1]] are dropped, endpoints are kept.
func TestClampArcsToCutDropsOutOfRange(t *testing.T) {
	t.Parallel()
	cut := [2]float64{0, 10}
	in := []float64{-1, 0, 3, 10, 11}
	out := clampArcsToCut(in, cut)
	want := []float64{0, 3, 10}
	if len(out) != len(want) {
		t.Fatalf("clampArcsToCut(%v, %v) = %v, want %v", in, cut, out, want)
	}
	for i := range want {
		if out[i] != want[i] {
			t.Errorf("clampArcsToCut[%d] = %v, want %v", i, out[i], want[i])
		}
	}
}

// TestReverseAnchorPlanIsInvolution proves reverseAnchorPlan applied twice restores the
// original plan exactly (anchors, arcs, and edge indices) — the property the walk-direction
// decision (bsplineHostWalkDirection, decided once from a coarse trial) implicitly relies on:
// reversing must not lose or corrupt information.
func TestReverseAnchorPlanIsInvolution(t *testing.T) {
	t.Parallel()
	c := lineCurve3{p0: math.P3(0, 0, 0), p1: math.P3(9, 0, 0)}
	tab, _ := newEdgeArcTable(c)
	spec := bsplineHostMarchSpec{arcTable: tab}
	arcs := openAnchorArcs(9, [2]float64{0, 9}, 3)
	original := openAnchorPlan(spec, 1, append([]float64{}, arcs...))

	twice := openAnchorPlan(spec, 1, append([]float64{}, arcs...))
	reverseAnchorPlan(&twice)
	reverseAnchorPlan(&twice)

	if len(twice.anchors) != len(original.anchors) {
		t.Fatalf("double-reverse changed anchor count: %d vs %d", len(twice.anchors), len(original.anchors))
	}
	for i := range original.anchors {
		if twice.anchors[i].P.DistanceTo(original.anchors[i].P) > 1e-9 {
			t.Errorf("anchor[%d].P = %v, want %v", i, twice.anchors[i].P, original.anchors[i].P)
		}
		if twice.anchors[i].T.Sub(original.anchors[i].T).Length() > 1e-9 {
			t.Errorf("anchor[%d].T = %v, want %v", i, twice.anchors[i].T, original.anchors[i].T)
		}
	}
	if twice.iEdge0 != original.iEdge0 || twice.iEdge1 != original.iEdge1 {
		t.Errorf("double-reverse edge indices = (%d,%d), want (%d,%d)",
			twice.iEdge0, twice.iEdge1, original.iEdge0, original.iEdge1)
	}
}

// TestReverseAnchorPlanSwapsEndsAndNegatesTangent is the single-reverse proof: the first and
// last anchor swap positions and every tangent flips sign — a single-reverse walk must retrace
// the SAME polyline backward, not a different curve.
func TestReverseAnchorPlanSwapsEndsAndNegatesTangent(t *testing.T) {
	t.Parallel()
	c := lineCurve3{p0: math.P3(0, 0, 0), p1: math.P3(6, 0, 0)}
	tab, _ := newEdgeArcTable(c)
	spec := bsplineHostMarchSpec{arcTable: tab}
	arcs := openAnchorArcs(6, [2]float64{0, 6}, 2)
	plan := openAnchorPlan(spec, 1, arcs)
	firstP, lastP := plan.anchors[0].P, plan.anchors[len(plan.anchors)-1].P
	firstT := plan.anchors[0].T

	reverseAnchorPlan(&plan)

	if plan.anchors[0].P.DistanceTo(lastP) > 1e-9 {
		t.Errorf("after reverse, anchors[0].P = %v, want the old last %v", plan.anchors[0].P, lastP)
	}
	if plan.anchors[len(plan.anchors)-1].P.DistanceTo(firstP) > 1e-9 {
		t.Errorf("after reverse, anchors[last].P = %v, want the old first %v", plan.anchors[len(plan.anchors)-1].P, firstP)
	}
	if got := plan.anchors[len(plan.anchors)-1].T; got.Add(firstT).Length() > 1e-9 {
		t.Errorf("after reverse, tangent = %v, want negated %v", got, firstT)
	}
}
