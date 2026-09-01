// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic straight-path sweep (#2164 follow-up): a rigid sweep along a straight run keeps the
// profile's arcs ANALYTIC — a circle → one true cylinder wall, a line+arc profile → real arc cap
// edges — instead of the faceted section skin whose ~48-facet-per-arc rings project onto a sketch as
// chorded arcs. A bent path, or any taper/twist/rail/guide, keeps the faceted skin.

// straightSweep builds a rigid NewBody sweep of profile 0 along the given straight path.
func straightSweep(t *testing.T, sk *sketch.Sketch, path *sketch.Path3D) *topo.Body {
	t.Helper()
	fs := NewPartFeatures(nil)
	def := &SweepDefinition{
		Sketch: sk, ProfileIndex: 0,
		Path:      func() *sketch.Path3D { return path },
		Operation: ops.NewBody,
	}
	return sweepDefRecompute(t, fs, def)
}

// TestSweepCircleStraightMakesCylinder: a circle swept up a straight Z run is ONE analytic cylinder
// wall (plus the two planar caps), not a faceted tube — so a projected face sees a real circle.
func TestSweepCircleStraightMakesCylinder(t *testing.T) {
	t.Parallel()
	const r, l = 2.0, 10.0
	body := straightSweep(t, circleSketchAt(0, 0, r), straightZPath(l, 8))
	if got := cylinderFaceCount(body); got != 1 {
		t.Fatalf("straight circle sweep has %d cylinder faces, want 1 (fell to the faceted skin)", got)
	}
	if got, want := bodyVolume(body), stdmath.Pi*r*r*l; relErr(got, want) > 0.02 {
		t.Errorf("swept cylinder volume = %g, want ≈%g (πr²l)", got, want)
	}
}

// TestSweepStadiumStraightKeepsAnalyticArcs: a stadium (two arcs + two lines) swept along a straight
// run keeps a cap bounded by exactly two arc + two line edges — the extrude analytic prism reused.
func TestSweepStadiumStraightKeepsAnalyticArcs(t *testing.T) {
	t.Parallel()
	const l, r, depth = 10.0, 3.0, 6.0
	body := straightSweep(t, stadiumSketch(l, r), straightZPath(depth, 8))
	cap := topCapPlanarFace(t, body, depth)
	if arcs, lines, other := capEdgeCurveCounts(cap); arcs != 2 || lines != 2 || other != 0 {
		t.Fatalf("straight stadium sweep cap = %d arc + %d line + %d other, want 2 arc + 2 line (arcs faceted, #2164)", arcs, lines, other)
	}
	if got, want := bodyVolume(body), (2*r*l+stdmath.Pi*r*r)*depth; relErr(got, want) > 0.02 {
		t.Errorf("swept stadium volume = %g, want ≈%g", got, want)
	}
}

// TestSweepCircleDiagonalStraightMakesCylinder exercises the frame rotation: a circle on the XY plane
// swept along a straight run in direction (1,1,1) still yields one analytic cylinder — the synthetic
// frame must rotate the profile so its normal follows the tangent, not stay axis-aligned.
func TestSweepCircleDiagonalStraightMakesCylinder(t *testing.T) {
	t.Parallel()
	const r = 1.5
	l := stdmath.Sqrt(3) * 4 // origin → (4,4,4)
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(4, 4, 4)),
	}, false)
	body := straightSweep(t, circleSketchAt(0, 0, r), path)
	if got := cylinderFaceCount(body); got != 1 {
		t.Fatalf("diagonal straight circle sweep has %d cylinder faces, want 1", got)
	}
	if got, want := bodyVolume(body), stdmath.Pi*r*r*l; relErr(got, want) > 0.02 {
		t.Errorf("diagonal swept cylinder volume = %g, want ≈%g (πr²l)", got, want)
	}
}

// circlePath3D samples a full circle of the given radius in the z=0 plane (centre origin) into n
// points, marked closed — the path a torus sweep rides.
func circlePath3D(radius float64, n int) *sketch.Path3D {
	pts := make([]*sketch.Point3D, n)
	for i := range n {
		th := 2 * stdmath.Pi * float64(i) / float64(n)
		pts[i] = sketch.NewPoint3D(math.P3(math.Scalar(radius*stdmath.Cos(th)), math.Scalar(radius*stdmath.Sin(th)), 0))
	}
	return sketch.NewPath3D(pts, true)
}

// TestSweepCircleAroundCircleMakesTorus: a circle swept along a FULL circular path is ONE analytic
// torus face (major = path radius, minor = profile radius), not a faceted ring — so projecting it
// sees a real circle, not a chord polygon.
func TestSweepCircleAroundCircleMakesTorus(t *testing.T) {
	t.Parallel()
	const major, minor = 10.0, 2.0
	body := straightSweep(t, circleSketchAt(0, 0, minor), circlePath3D(major, 24))
	if got := torusFaceCount(body); got != 1 {
		t.Fatalf("circle-around-circle sweep has %d torus faces, want 1 (fell to the faceted ring)", got)
	}
	if got, want := bodyVolume(body), 2*stdmath.Pi*stdmath.Pi*major*minor*minor; relErr(got, want) > 0.02 {
		t.Errorf("swept torus volume = %g, want ≈%g (2π²·R·r²)", got, want)
	}
}

// TestSweepBentPathStaysFaceted guards the collinearity gate: a genuinely bent path is NOT a straight
// run, so it must keep the faceted skin (zero analytic cylinder faces) — never mis-claimed analytic.
func TestSweepBentPathStaysFaceted(t *testing.T) {
	t.Parallel()
	const r = 1.0
	path := sketch.NewPath3D([]*sketch.Point3D{
		sketch.NewPoint3D(math.P3(0, 0, 0)),
		sketch.NewPoint3D(math.P3(0, 0, 5)),
		sketch.NewPoint3D(math.P3(5, 0, 5)),
	}, false)
	body := straightSweep(t, circleSketchAt(0, 0, r), path)
	if got := cylinderFaceCount(body); got != 0 {
		t.Fatalf("bent-path sweep has %d analytic cylinder faces, want 0 (a bend must keep the faceted skin)", got)
	}
}
