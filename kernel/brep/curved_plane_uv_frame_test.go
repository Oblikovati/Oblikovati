// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Exact conic∩polygon numerics for the planeUV operand (#1591, ADR-0049 D-d). Hand-computed fixtures so the
// stable-quadratic crossings, the tangency decline, and the decline-biased gate are pinned independently of
// any assembly.

func unitCircleConic(cx, cy, r float64) planeConic {
	return planeConic{center: math.P2(math.Scalar(cx), math.Scalar(cy)), maj: math.V2(1, 0), A: r, B: r}
}

// TestConicEdgeHitsCircleTwoCrossings: a horizontal chord y=1 through a radius-2 circle at the origin meets
// it at x=±√3, i.e. edge parameters (3∓√3)/6 for the edge (−3,1)→(3,1).
func TestConicEdgeHitsCircleTwoCrossings(t *testing.T) {
	res := geom.ResolutionForSize(1)
	hits, tangent := conicEdgeHits(unitCircleConic(0, 0, 2), math.P2(-3, 1), math.P2(3, 1), res)
	if tangent {
		t.Fatal("a transversal chord must not be flagged tangent")
	}
	if len(hits) != 2 {
		t.Fatalf("got %d crossings, want 2", len(hits))
	}
	root := stdmath.Sqrt(3)
	for _, h := range hits {
		if e := stdmath.Abs(float64(h.p.Y) - 1); e > 1e-12 {
			t.Errorf("crossing off the edge line (y=%.15g)", float64(h.p.Y))
		}
		if e := stdmath.Abs(stdmath.Abs(float64(h.p.X)) - root); e > 1e-9 {
			t.Errorf("crossing x=%.9f, want ±√3=%.9f", float64(h.p.X), root)
		}
	}
}

// TestConicEdgeHitsTangentDeclines: the line y=2 grazes the radius-2 circle at (0,2) — a double root, which
// must be reported tangent (a zero-width sliver the stitch cannot weld) so the gate can decline it.
func TestConicEdgeHitsTangentDeclines(t *testing.T) {
	res := geom.ResolutionForSize(1)
	_, tangent := conicEdgeHits(unitCircleConic(0, 0, 2), math.P2(-3, 2), math.P2(3, 2), res)
	if !tangent {
		t.Error("a grazing (tangent) chord must be flagged tangent")
	}
}

// TestConicEdgeHitsMiss: a chord clear of the circle has no crossings and is not tangent.
func TestConicEdgeHitsMiss(t *testing.T) {
	res := geom.ResolutionForSize(1)
	hits, tangent := conicEdgeHits(unitCircleConic(0, 0, 2), math.P2(-3, 3), math.P2(3, 3), res)
	if len(hits) != 0 || tangent {
		t.Errorf("a clear miss must yield 0 hits and not-tangent, got %d hits tangent=%v", len(hits), tangent)
	}
}

// TestConicEdgeHitsEllipseNormalizes: a radius via the unit-circle normalisation — a 2:1 ellipse (A=2,B=1)
// axis-aligned, chord x=0 (the minor axis) crosses at y=±1; chord y=0 (major axis) at x=±2.
func TestConicEdgeHitsEllipseNormalizes(t *testing.T) {
	res := geom.ResolutionForSize(1)
	ell := planeConic{center: math.P2(0, 0), maj: math.V2(1, 0), A: 2, B: 1}
	minor, _ := conicEdgeHits(ell, math.P2(0, -3), math.P2(0, 3), res)
	if len(minor) != 2 {
		t.Fatalf("minor-axis chord: %d hits, want 2", len(minor))
	}
	for _, h := range minor {
		if e := stdmath.Abs(stdmath.Abs(float64(h.p.Y)) - 1); e > 1e-9 {
			t.Errorf("minor crossing y=%.9f, want ±1", float64(h.p.Y))
		}
	}
	major, _ := conicEdgeHits(ell, math.P2(-3, 0), math.P2(3, 0), res)
	for _, h := range major {
		if e := stdmath.Abs(stdmath.Abs(float64(h.p.X)) - 2); e > 1e-9 {
			t.Errorf("major crossing x=%.9f, want ±2", float64(h.p.X))
		}
	}
}

func squareUV(half float64) [][]math.Point2 {
	h := math.Scalar(half)
	return [][]math.Point2{{math.P2(-h, -h), math.P2(h, -h), math.P2(h, h), math.P2(-h, h)}}
}

// TestPlaneUVContactOK: a circle clipping one square edge is a valid partial contact (2 crossings); one
// wholly inside is not partial (0 crossings); an over-eccentric ellipse declines.
func TestPlaneUVContactOK(t *testing.T) {
	res := geom.ResolutionForSize(1)
	if !planeUVContactOK(unitCircleConic(3, 0, 2), squareUV(3), res) {
		t.Error("a circle clipping the right edge (2 crossings) is a valid partial contact")
	}
	if planeUVContactOK(unitCircleConic(0, 0, 1), squareUV(3), res) {
		t.Error("a circle wholly inside (0 crossings) is not a partial contact — declines to the fast path")
	}
	if planeUVContactOK(planeConic{center: math.P2(3, 0), maj: math.V2(1, 0), A: 2, B: 0.0005}, squareUV(3), res) {
		t.Error("an over-eccentric ellipse (B/A<1e-3) must decline to CSG")
	}
}

// TestToPlaneConicCircle: a circle lying in the XY plane projects to a chart circle of the same radius,
// centred at the plane-projected centre (the chart is an isometry).
func TestToPlaneConicCircle(t *testing.T) {
	pl, err := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	circ, _ := geom.NewCircle(math.P3(1, 2, 0), math.V3(0, 0, 1), 2.5)
	C, ok := toPlaneConic(circ, pl)
	if !ok {
		t.Fatal("a circle imprint must project")
	}
	if stdmath.Abs(C.A-2.5) > 1e-12 || stdmath.Abs(C.B-2.5) > 1e-12 {
		t.Errorf("chart conic radii (%.3f,%.3f), want 2.5,2.5", C.A, C.B)
	}
}
