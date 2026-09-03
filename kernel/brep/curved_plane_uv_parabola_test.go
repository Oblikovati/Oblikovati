// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane cuts a cone in any of the three conics depending only on how it is tilted, and an imprint
// must not care which it got. planeConic carried two of them; the parabola is the third, and its
// equation in the conic's own frame is ξ² = 4f·η rather than the signed ξ²±η²=1 the other two share.
// One quadratic still covers all three, so there is one solver (ADR-0062).

// parabolaChart is the (u,v) form of z = x²/8 in the y=0 plane (focal 2), placed at the origin.
func parabolaChart(t *testing.T) (planeConic, geom.Plane) {
	t.Helper()
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	par, err := geom.NewParabola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2)
	if err != nil {
		t.Fatalf("NewParabola: %v", err)
	}
	pc, ok := toPlaneConic(par, pl)
	if !ok {
		t.Fatal("toPlaneConic refused a parabola")
	}
	return pc, pl
}

// TestPlaneConicSolvesAParabolaAgainstAnEdge: a segment crossing both arms gives two exact crossings,
// each ON the parabola.
func TestPlaneConicSolvesAParabolaAgainstAnEdge(t *testing.T) {
	t.Parallel()
	pc, pl := parabolaChart(t)
	// The horizontal line z = 2 meets z = x²/8 at x = ±4.
	a, b := to2D(pl, math.P3(-9, 0, 2)), to2D(pl, math.P3(9, 0, 2))
	hits, tangent := conicFrameHits(pc, a, b, geom.ResolutionForSize(20))
	if tangent {
		t.Error("a transversal crossing was reported tangent")
	}
	if len(hits) != 2 {
		t.Fatalf("got %d crossings, want 2 (one per arm)", len(hits))
	}
	for _, h := range hits {
		p := to3D(pl, h.p)
		if d := stdmath.Abs(float64(p.Z) - float64(p.X)*float64(p.X)/8); d > 1e-9 {
			t.Errorf("crossing %v is %g off the parabola z = x²/8", p, d)
		}
		if x := stdmath.Abs(float64(p.X)); stdmath.Abs(x-4) > 1e-9 {
			t.Errorf("crossing at |x|=%g, want 4", x)
		}
	}
}

// TestPlaneConicParabolaAxisParallelEdgeHasOneRoot: an edge parallel to the parabola's axis of symmetry
// collapses the quadratic to a linear equation — one root, not a degeneracy. It is the same shape a
// hyperbola's asymptote-parallel edge has, and it goes through the same branch.
func TestPlaneConicParabolaAxisParallelEdgeHasOneRoot(t *testing.T) {
	t.Parallel()
	pc, pl := parabolaChart(t)
	// x = 4 runs along the axis direction (z); it meets z = x²/8 once, at z = 2.
	a, b := to2D(pl, math.P3(4, 0, -5)), to2D(pl, math.P3(4, 0, 9))
	hits, _ := conicFrameHits(pc, a, b, geom.ResolutionForSize(20))
	if len(hits) != 1 {
		t.Fatalf("got %d crossings, want exactly 1", len(hits))
	}
	p := to3D(pl, hits[0].p)
	if stdmath.Abs(float64(p.Z)-2) > 1e-9 {
		t.Errorf("the crossing is at z=%v, want 2", p.Z)
	}
}

// TestPlaneConicParabolaMissIsNoCrossing: an edge wholly inside the parabola's opening, or wholly
// outside it, crosses nothing.
func TestPlaneConicParabolaMissIsNoCrossing(t *testing.T) {
	t.Parallel()
	pc, pl := parabolaChart(t)
	below := []math.Point3{math.P3(-1, 0, -3), math.P3(1, 0, -3)} // under the vertex: outside
	if hits, _ := conicFrameHits(pc, to2D(pl, below[0]), to2D(pl, below[1]), geom.ResolutionForSize(20)); len(hits) != 0 {
		t.Errorf("an edge clear of the parabola gave %d crossings", len(hits))
	}
}
