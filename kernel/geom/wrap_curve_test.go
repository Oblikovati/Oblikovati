// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestArcLengthParamVariableSpeed validates the metric-integration inverse on a NON-constant
// speed — the genuinely numerical core of the wrap. With speed(p)=1+p the arc length from 0 to X
// is ∫₀ˣ(1+p)dp = X + X²/2, so reaching arc length T lands at X = √(1+2T) − 1. The midpoint
// integrator must recover that (constant-speed surfaces would never exercise this).
func TestArcLengthParamVariableSpeed(t *testing.T) {
	speed := func(p float64) float64 { return 1 + p }
	for _, target := range []float64{0.5, 2, 5, 12} {
		want := stdmath.Sqrt(1+2*target) - 1
		got := arcLengthParam(0, target, speed)
		if stdmath.Abs(got-want) > 1e-4 {
			t.Errorf("arcLengthParam(0, %v) = %v, want %v (√(1+2T)−1)", target, got, want)
		}
	}
}

// TestArcLengthParamConstantSpeedExact: a constant speed inverts exactly (param0 + target/speed),
// in both directions — the cylinder/plane case must carry no integration error.
func TestArcLengthParamConstantSpeedExact(t *testing.T) {
	speed := func(float64) float64 { return 2 }
	if got := arcLengthParam(1, 6, speed); stdmath.Abs(got-4) > 1e-12 {
		t.Errorf("forward: got %v, want 4 (1 + 6/2)", got)
	}
	if got := arcLengthParam(1, -6, speed); stdmath.Abs(got-(-2)) > 1e-12 {
		t.Errorf("backward: got %v, want -2 (1 − 6/2)", got)
	}
}

// TestWrapCurveOntoCylinderIsArcLengthUnwrap is the headline analytic check: wrapping a planar
// source line onto a cylinder of radius R maps planar x to angle u = x/R (the cylinder unwrap),
// preserving arc length. The source runs along the frame's U axis (circumferential) from the
// anchor, so each point maps onto the cylinder's equator by exactly its planar arc length.
func TestWrapCurveOntoCylinderIsArcLengthUnwrap(t *testing.T) {
	const r = 2.0
	cyl, err := NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r)
	if err != nil {
		t.Fatalf("NewCylinderWithRef: %v", err)
	}
	// Anchor (R,0,0) → (u,v)=(0,0); U circumferential (0,1,0); V axial (0,0,1).
	frame := WrapFrame{Origin: math.P3(r, 0, 0), U: math.V3(0, 1, 0), V: math.V3(0, 0, 1)}
	// Planar source from a=0 to a=π·R/2 (a quarter of the way around), all in the tangent plane.
	arc := stdmath.Pi * r / 2
	src := NewLineSegment(math.P3(r, 0, 0), math.P3(r, arc, 0))

	pts := WrapCurveOntoSurface(cyl, src, frame, 32)
	if len(pts) != 33 {
		t.Fatalf("got %d points, want 33", len(pts))
	}
	for i, p := range pts {
		a := arc * float64(i) / 32 // planar arc length of this sample
		want := math.P3(math.Scalar(r*stdmath.Cos(a/r)), math.Scalar(r*stdmath.Sin(a/r)), 0)
		if !p.IsEqualTo(want, 1e-6) {
			t.Fatalf("sample %d (a=%.4f): got %v, want %v — u must equal a/R", i, a, p, want)
		}
	}
	// End point: a quarter turn lands at (0, R, 0).
	if !pts[len(pts)-1].IsEqualTo(math.P3(0, r, 0), 1e-6) {
		t.Errorf("quarter-turn end = %v, want (0,2,0)", pts[len(pts)-1])
	}
}

// TestWrapCurveOntoCylinderAxialOffsetIsIndependent: a planar V displacement (b) maps to axial
// height v = b, independent of the circumferential wrap — confirms the two parameter directions
// are decoupled on the cylinder (diagonal metric).
func TestWrapCurveOntoCylinderAxialOffsetIsIndependent(t *testing.T) {
	const r = 3.0
	cyl, _ := NewCylinderWithRef(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), r)
	frame := WrapFrame{Origin: math.P3(r, 0, 0), U: math.V3(0, 1, 0), V: math.V3(0, 0, 1)}
	// A single-point-thick source at planar (a=π·R, b=5): half a turn around, 5 up the axis.
	a := stdmath.Pi * r
	src := NewLineSegment(math.P3(r, a, 5), math.P3(r, a, 5))

	pts := WrapCurveOntoSurface(cyl, src, frame, 4)
	// u = a/R = π → (−R, 0, ·); v = b = 5.
	if !pts[0].IsEqualTo(math.P3(-r, 0, 5), 1e-6) {
		t.Errorf("got %v, want (-3,0,5) — half turn at axial height 5", pts[0])
	}
}

// TestWrapCurveOntoPlaneReconstructsSource: wrapping onto a plane whose frame matches the source
// frame is the identity — every planar point maps back to itself (the degenerate flat case).
func TestWrapCurveOntoPlaneReconstructsSource(t *testing.T) {
	plane, err := NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0))
	if err != nil {
		t.Fatalf("NewPlaneFromAxes: %v", err)
	}
	frame := WrapFrame{Origin: math.P3(0, 0, 0), U: math.V3(1, 0, 0), V: math.V3(0, 1, 0)}
	src := NewLineSegment(math.P3(1, 2, 0), math.P3(4, 6, 0))

	pts := WrapCurveOntoSurface(plane, src, frame, 8)
	for i, p := range pts {
		want := src.PointAt(float64(i) / 8)
		if !p.IsEqualTo(want, 1e-9) {
			t.Errorf("sample %d: got %v, want %v (identity on a matching plane)", i, p, want)
		}
	}
}

// TestWrapCurveOntoSurfaceNilInputs guards the degenerate inputs (nil surface/curve, no samples).
func TestWrapCurveOntoSurfaceNilInputs(t *testing.T) {
	plane, _ := NewPlaneFromAxes(math.P3(0, 0, 0), math.V3(1, 0, 0), math.V3(0, 1, 0))
	src := NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	frame := WrapFrame{Origin: math.P3(0, 0, 0), U: math.V3(1, 0, 0), V: math.V3(0, 1, 0)}
	if pts := WrapCurveOntoSurface(nil, src, frame, 4); pts != nil {
		t.Errorf("nil surface: got %v, want nil", pts)
	}
	if pts := WrapCurveOntoSurface(plane, nil, frame, 4); pts != nil {
		t.Errorf("nil source: got %v, want nil", pts)
	}
	if pts := WrapCurveOntoSurface(plane, src, frame, 0); pts != nil {
		t.Errorf("zero samples: got %v, want nil", pts)
	}
}
