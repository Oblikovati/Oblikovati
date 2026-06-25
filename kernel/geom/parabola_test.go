// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// A parabola point at t=0 is the vertex; the two arms climb symmetrically along the axis.
func TestParabolaVertexAndArms(t *testing.T) {
	p, err := NewParabola(math.P3(1, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), 2)
	if err != nil {
		t.Fatalf("NewParabola: %v", err)
	}
	if v := p.PointAt(0); ptFar(v, math.P3(1, 0, 0)) {
		t.Errorf("vertex = %v, want (1,0,0)", v)
	}
	up, down := p.PointAt(0.8), p.PointAt(-0.8)
	if stdmath.Abs(float64(up.Z-down.Z)) > 1e-9 {
		t.Errorf("arms not symmetric along the axis: %v vs %v", up, down)
	}
	if stdmath.Abs(float64(up.Y+down.Y)) > 1e-9 {
		t.Errorf("arms not antisymmetric across the axis: %v vs %v", up, down)
	}
	// y = x²/(4f): at t (cross) = 0.8, the axial offset is 0.8²/(4·2) = 0.08.
	if got := float64(up.Z); stdmath.Abs(got-0.08) > 1e-9 {
		t.Errorf("axial offset = %g, want 0.08", got)
	}
}

// NewParabola re-orthogonalizes the cross axis against the axis and rejects degenerate input.
func TestNewParabolaValidation(t *testing.T) {
	if _, err := NewParabola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 1), 1); err == nil {
		t.Error("parallel axes should error")
	}
	if _, err := NewParabola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), -1); err == nil {
		t.Error("non-positive focal length should error")
	}
	p, err := NewParabola(math.P3(0, 0, 0), math.V3(0, 0, 2), math.V3(0.5, 1, 0), 1)
	if err != nil {
		t.Fatalf("NewParabola: %v", err)
	}
	if d := p.CrossDir.AsVector().Dot(p.AxisDir.AsVector()); stdmath.Abs(float64(d)) > 1e-12 {
		t.Errorf("cross not orthogonal to axis: dot=%g", d)
	}
}

// TangentAt matches a central finite difference of PointAt.
func TestParabolaTangentFiniteDifference(t *testing.T) {
	p, _ := NewParabola(math.P3(0, 1, 0), math.V3(1, 0, 0), math.V3(0, 1, 0), 1.5)
	const eps = 1e-6
	for _, t0 := range []float64{-1.2, 0, 0.8} {
		fd := p.PointAt(t0 - eps).VectorTo(p.PointAt(t0 + eps)).Scale(math.Scalar(1 / (2 * eps)))
		an := p.TangentAt(t0)
		if d := an.Sub(fd).Length(); float64(d) > 1e-4 {
			t.Errorf("t=%g tangent %v vs finite-diff %v (|Δ|=%g)", t0, an, fd, d)
		}
	}
}

// ParabolicArc reparameterizes the same parabola onto s∈[0,1] over [T0,T1].
func TestParabolicArcReparam(t *testing.T) {
	p, _ := NewParabola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), 2)
	arc := p.Arc(-0.5, 1.5)
	if ptFar(arc.PointAt(0), p.PointAt(-0.5)) || ptFar(arc.PointAt(1), p.PointAt(1.5)) {
		t.Error("arc endpoints must match the parabola at T0/T1")
	}
	if ptFar(arc.PointAt(0.5), p.PointAt(0.5)) {
		t.Error("arc midpoint must match the parabola at the mid t")
	}
	if lo, hi := arc.Domain(); lo != 0 || hi != 1 {
		t.Errorf("arc domain = [%g,%g], want [0,1]", lo, hi)
	}
	const eps = 1e-6
	fd := arc.PointAt(0.5 - eps).VectorTo(arc.PointAt(0.5 + eps)).Scale(math.Scalar(1 / (2 * eps)))
	if d := arc.TangentAt(0.5).Sub(fd).Length(); float64(d) > 1e-4 {
		t.Errorf("arc tangent %v vs finite-diff %v", arc.TangentAt(0.5), fd)
	}
}

// CurveParamAtPoint3 inverts a parabola in closed form: the cross coordinate of a point on the curve
// round-trips to the t (or arc s) that produced it.
func TestParabolaParamRoundTrip(t *testing.T) {
	p, _ := NewParabola(math.P3(1, -2, 3), math.V3(0, 0, 1), math.V3(0, 1, 0), 1.5)
	if lo, hi := p.Domain(); !stdmath.IsInf(lo, -1) || !stdmath.IsInf(hi, 1) {
		t.Errorf("Parabola.Domain = [%g,%g], want unbounded", lo, hi)
	}
	for _, t0 := range []float64{-1.3, 0, 0.9} {
		got, nature := CurveParamAtPoint3(p, p.PointAt(t0))
		if nature != UniqueSolution || stdmath.Abs(got-t0) > 1e-9 {
			t.Errorf("Parabola param at t=%g: got %g (%v)", t0, got, nature)
		}
	}
	arc := p.Arc(-1, 2)
	for _, s := range []float64{0.1, 0.5, 0.95} {
		got, _ := CurveParamAtPoint3(arc, arc.PointAt(s))
		if stdmath.Abs(got-s) > 1e-9 {
			t.Errorf("ParabolicArc param at s=%g: got %g", s, got)
		}
	}
}
