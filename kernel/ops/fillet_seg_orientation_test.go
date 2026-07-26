// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// THE INVARIANT THESE THREE TESTS PIN: a loop segment's carried curve must run from the segment's OWN
// `from` point to its OWN `to` point. It is not cosmetic. discretizeEdge samples an edge's curve over its
// domain and then FORCES the polyline's two ends onto the edge's start/end vertices (edge_discretize.go),
// so a curve pointing the other way yields a polyline that leaps to the far end, walks the curve backwards,
// and leaps again — a doubled-back boundary whose developed loop self-crosses, with no area, no inside and
// no correct triangulation. Two retrim paths violated it; five shipped edges across four corpus cases
// carried it (planar-retrim-selfcross-report.md).

// curveRunsFromTo reports whether c runs p0→p1 over its own domain, within tol.
func curveRunsFromTo(c geom.Curve3, p0, p1 math.Point3, tol float64) bool {
	lo, hi := c.Domain()
	return float64(c.PointAt(lo).DistanceTo(p0)) <= tol && float64(c.PointAt(hi).DistanceTo(p1)) <= tol
}

// quarterArcSeg is a radius-10 quarter arc in z=0 from (10,0,0) to (0,10,0) through its own midpoint,
// packaged as the endSeg a far cross-section trim hands to the cap retrim.
func quarterArcSeg(t *testing.T) endSeg {
	t.Helper()
	from, to := math.P3(10, 0, 0), math.P3(0, 10, 0)
	mid := math.P3(10/stdmath.Sqrt2, 10/stdmath.Sqrt2, 0)
	arc, err := geom.Arc3dByThreePoints(from, mid, to)
	if err != nil {
		t.Fatalf("quarter arc through (10,0,0),(%v),(0,10,0): %v", mid, err)
	}
	return endSeg{from: from, to: to, curve: arc, mid: mid, arc: true}
}

// TestMatchArcFeetReversesTheSwappedPairing is the regression guard for the M4/N3/N9 root: when the
// cross-section arc's `to` foot is the one lying on the PREVIOUS flank, matchArcFeet must return the arc
// REVERSED, not merely its endpoints exchanged. The bug returned the two points while the caller re-wrapped
// the original curve, shipping a segment whose curve ran to→from.
func TestMatchArcFeetReversesTheSwappedPairing(t *testing.T) {
	arc := quarterArcSeg(t)
	// prev's supporting line carries the arc's TO foot (0,10,0); next's carries its FROM foot (10,0,0).
	prev := endSeg{from: math.P3(0, 30, 0), to: math.P3(0, 10, 0)}
	next := endSeg{from: math.P3(10, 0, 0), to: math.P3(30, 0, 0)}
	got, ok := matchArcFeet(prev, next, arc, 1e-9)
	if !ok {
		t.Fatal("the swapped pairing must match: prev carries the arc's to foot, next its from foot")
	}
	if got.from != arc.to || got.to != arc.from {
		t.Errorf("swapped pairing must run to→from, got %v→%v (arc %v→%v)", got.from, got.to, arc.from, arc.to)
	}
	if !curveRunsFromTo(got.curve, got.from, got.to, 1e-9) {
		lo, hi := got.curve.Domain()
		t.Errorf("swapped pairing's curve runs %v→%v but its segment runs %v→%v — a doubled-back boundary",
			got.curve.PointAt(lo), got.curve.PointAt(hi), got.from, got.to)
	}
}

// TestMatchArcFeetKeepsTheAlignedPairing pins the other side: when the arc already runs prev-side →
// next-side it is returned UNTOUCHED (identical curve object), so the fix cannot pass by reversing
// everything and every aligned corpus case stays byte-identical.
func TestMatchArcFeetKeepsTheAlignedPairing(t *testing.T) {
	arc := quarterArcSeg(t)
	prev := endSeg{from: math.P3(30, 0, 0), to: math.P3(10, 0, 0)}
	next := endSeg{from: math.P3(0, 10, 0), to: math.P3(0, 30, 0)}
	got, ok := matchArcFeet(prev, next, arc, 1e-9)
	if !ok {
		t.Fatal("the aligned pairing must match")
	}
	if got.from != arc.from || got.to != arc.to || got.curve != arc.curve {
		t.Errorf("aligned pairing must be returned untouched, got %v→%v", got.from, got.to)
	}
}

// TestSurvivorCurveReversesAReversedOpenBSpline is the regression guard for the complex/F2 root: a
// REVERSED use of an OPEN non-arc curved survivor must come back traversed backwards. The Arc3d arm always
// flipped its sweep; this arm returned the curve as-is, so F2 shipped two b-spline retrim edges whose
// curve ran end→start (28.52 / 28.18 of endpoint mismatch) and three of its walls self-crossed.
func TestSurvivorCurveReversesAReversedOpenBSpline(t *testing.T) {
	p0, p1 := math.P3(0, 0, 0), math.P3(10, 0, 0)
	spline, err := geom.NewBSplineCurveUniformWeights(3,
		[]math.Point3{p0, math.P3(4, 3, 0), math.P3(7, -3, 0), p1}, []float64{0, 0, 0, 0, 1, 1, 1, 1})
	if err != nil {
		t.Fatalf("cubic bezier b-spline through 4 control points: %v", err)
	}
	fwd, rev := survivorUsesOf(t, spline, p0, p1)
	if !curveRunsFromTo(survivorCurve(fwd), p0, p1, 1e-9) {
		t.Error("a FORWARD use must carry the curve unchanged (p0→p1)")
	}
	if !curveRunsFromTo(survivorCurve(rev), p1, p0, 1e-9) {
		lo, hi := survivorCurve(rev).Domain()
		t.Errorf("a REVERSED use of an open b-spline must carry it p1→p0, got %v→%v",
			survivorCurve(rev).PointAt(lo), survivorCurve(rev).PointAt(hi))
	}
}

// survivorUsesOf builds a two-face body sharing one edge with curve c, and returns that edge's FORWARD
// use (on the first face) and its REVERSED use (on the second) — the pair transformLoop walks.
func survivorUsesOf(t *testing.T, c geom.Curve3, p0, p1 math.Point3) (fwd, rev *topo.EdgeUse) {
	t.Helper()
	lin := topo.NewLineage(topo.Tok("test", "survivor-orient", 0))
	bld := topo.NewBuilder(true, lin)
	v0, v1 := bld.AddVertex(p0, lin), bld.AddVertex(p1, lin)
	e := bld.AddEdge(c, v0, v1, lin)
	f0 := bld.AddFace(planeWithNormal(0, 0, 1), lin, topo.OuterLoop(topo.Fwd(e), topo.Rev(e)))
	f1 := bld.AddFace(planeWithNormal(0, 0, -1), lin, topo.OuterLoop(topo.Rev(e), topo.Fwd(e)))
	bld.Build()
	return f0.Loops()[0].EdgeUses()[0], f1.Loops()[0].EdgeUses()[0]
}
