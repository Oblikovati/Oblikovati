// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// lineCurve3 is a minimal geom.Curve3 test double — a straight segment from P0 to P1 over
// domain [0, 1] — used to exercise edgeArcTable against ground truth the test can compute by
// hand (a straight line's arc length is exactly the chord, so every sample is checkable to
// float precision instead of only "close").
type lineCurve3 struct{ p0, p1 math.Point3 }

func (l lineCurve3) PointAt(t float64) math.Point3 {
	return l.p0.TranslateBy(l.p0.VectorTo(l.p1).Scale(math.Scalar(t)))
}
func (l lineCurve3) TangentAt(float64) math.Vector3 { return l.p0.VectorTo(l.p1) }
func (l lineCurve3) Domain() (float64, float64)     { return 0, 1 }

// TestNewEdgeArcTableLengthMatchesChord is the ground-truth proof: a straight line's arc
// length must equal its Euclidean chord length, not merely approximate it (the polyline
// sampling is exact on a line, so there is no discretization slack to hide behind).
func TestNewEdgeArcTableLengthMatchesChord(t *testing.T) {
	t.Parallel()
	c := lineCurve3{p0: math.P3(0, 0, 0), p1: math.P3(30, 40, 0)} // 3-4-5 triangle, chord=50
	tab, ok := newEdgeArcTable(c)
	if !ok {
		t.Fatal("newEdgeArcTable declined a well-formed line")
	}
	if got, want := tab.length, 50.0; stdmath.Abs(got-want) > 1e-9 {
		t.Fatalf("table length = %v, want %v", got, want)
	}
}

// TestEdgeArcTableAtEndpointsAndMidpoint pins .at() against the closed-form line position at
// s=0, s=L/2 and s=L, and checks the tangent direction is the line's own direction throughout
// (a straight line has no curvature, so the tangent must never wobble).
func TestEdgeArcTableAtEndpointsAndMidpoint(t *testing.T) {
	t.Parallel()
	c := lineCurve3{p0: math.P3(0, 0, 0), p1: math.P3(10, 0, 0)}
	tab, ok := newEdgeArcTable(c)
	if !ok {
		t.Fatal("newEdgeArcTable declined a well-formed line")
	}
	cases := []struct {
		s    float64
		want math.Point3
	}{
		{0, math.P3(0, 0, 0)},
		{5, math.P3(5, 0, 0)},
		{10, math.P3(10, 0, 0)},
	}
	for _, tc := range cases {
		p, tan := tab.at(tc.s)
		if p.DistanceTo(tc.want) > 1e-6 {
			t.Errorf("at(%v) = %v, want %v", tc.s, p, tc.want)
		}
		if tan.X <= 0 || stdmath.Abs(tan.Y) > 1e-12 || stdmath.Abs(tan.Z) > 1e-12 {
			t.Errorf("at(%v) tangent = %v, want +X direction", tc.s, tan)
		}
	}
}

// TestNewEdgeArcTableDeclinesDegenerateCurve proves the zero-length guard: a curve whose two
// endpoints coincide must decline (ok=false), never build a table with length 0 that a caller
// could divide by.
func TestNewEdgeArcTableDeclinesDegenerateCurve(t *testing.T) {
	t.Parallel()
	c := lineCurve3{p0: math.P3(1, 1, 1), p1: math.P3(1, 1, 1)}
	if _, ok := newEdgeArcTable(c); ok {
		t.Fatal("newEdgeArcTable accepted a zero-length curve")
	}
}

// TestUniformAnchorsSpacingAndEndpoints checks uniformAnchors places anchors evenly (equal
// arc-length steps) and that dir=+1 starts at P0 while dir=-1 starts at P1 (the walk-direction
// contract bsplineHostWalkDirection and the closed-loop march both depend on).
func TestUniformAnchorsSpacingAndEndpoints(t *testing.T) {
	t.Parallel()
	c := lineCurve3{p0: math.P3(0, 0, 0), p1: math.P3(12, 0, 0)}
	tab, _ := newEdgeArcTable(c)
	fwd := tab.uniformAnchors(4, 1)
	if fwd[0].P.DistanceTo(math.P3(0, 0, 0)) > 1e-9 || fwd[3].P.DistanceTo(math.P3(12, 0, 0)) > 1e-9 {
		t.Fatalf("dir=+1 endpoints = %v .. %v, want (0,0,0)..(12,0,0)", fwd[0].P, fwd[3].P)
	}
	for i := range 3 {
		step := fwd[i].P.DistanceTo(fwd[i+1].P)
		if stdmath.Abs(step-4) > 1e-9 {
			t.Errorf("fwd step %d = %v, want 4", i, step)
		}
	}
	rev := tab.uniformAnchors(4, -1)
	if rev[0].P.DistanceTo(math.P3(12, 0, 0)) > 1e-9 || rev[3].P.DistanceTo(math.P3(0, 0, 0)) > 1e-9 {
		t.Fatalf("dir=-1 endpoints = %v .. %v, want (12,0,0)..(0,0,0)", rev[0].P, rev[3].P)
	}
	if rev[0].T.X >= 0 {
		t.Fatalf("dir=-1 tangent = %v, want negative X (walking backward)", rev[0].T)
	}
}
