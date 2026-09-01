// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// A hyperbola point at θ=0 is the vertex Center + A·TransverseAxis; the arms climb symmetrically.
func TestHyperbolaVertexAndArms(t *testing.T) {
	t.Parallel()
	h, err := NewHyperbola(math.P3(1, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), 2, 3)
	if err != nil {
		t.Fatalf("NewHyperbola: %v", err)
	}
	vertex := h.PointAt(0)
	if got := vertex; ptFar(got, math.P3(1, 0, 2)) {
		t.Errorf("vertex = %v, want (1,0,2)", got)
	}
	up, down := h.PointAt(0.7), h.PointAt(-0.7)
	if stdmath.Abs(float64(up.Z-down.Z)) > 1e-9 {
		t.Errorf("arms not symmetric in transverse: %v vs %v", up, down)
	}
	if stdmath.Abs(float64(up.Y+down.Y)) > 1e-9 {
		t.Errorf("arms not antisymmetric in conjugate: %v vs %v", up, down)
	}
}

// NewHyperbola re-orthogonalizes the conjugate axis against the transverse and rejects degenerate input.
func TestNewHyperbolaValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewHyperbola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 0, 1), 1, 1); err == nil {
		t.Error("parallel axes should error")
	}
	if _, err := NewHyperbola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), -1, 1); err == nil {
		t.Error("non-positive semi-axis should error")
	}
	h, err := NewHyperbola(math.P3(0, 0, 0), math.V3(0, 0, 2), math.V3(0.5, 1, 0), 1, 1)
	if err != nil {
		t.Fatalf("NewHyperbola: %v", err)
	}
	if d := h.ConjugateAxis.AsVector().Dot(h.TransverseAxis.AsVector()); stdmath.Abs(float64(d)) > 1e-12 {
		t.Errorf("conjugate not orthogonal to transverse: dot=%g", d)
	}
}

// TangentAt matches a central finite difference of PointAt.
func TestHyperbolaTangentFiniteDifference(t *testing.T) {
	t.Parallel()
	h, _ := NewHyperbola(math.P3(0, 1, 0), math.V3(1, 0, 0), math.V3(0, 1, 0), 1.5, 2.5)
	const eps = 1e-6
	for _, theta := range []float64{-1.2, 0, 0.8} {
		fd := h.PointAt(theta - eps).VectorTo(h.PointAt(theta + eps)).Scale(math.Scalar(1 / (2 * eps)))
		an := h.TangentAt(theta)
		if d := an.Sub(fd).Length(); float64(d) > 1e-4 {
			t.Errorf("θ=%g tangent %v vs finite-diff %v (|Δ|=%g)", theta, an, fd, d)
		}
	}
}

// HyperbolicArc reparameterizes the same branch onto t∈[0,1] over [Theta0,Theta1].
func TestHyperbolicArcReparam(t *testing.T) {
	t.Parallel()
	h, _ := NewHyperbola(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(0, 1, 0), 2, 1)
	arc := h.Arc(-0.5, 1.5)
	if ptFar(arc.PointAt(0), h.PointAt(-0.5)) || ptFar(arc.PointAt(1), h.PointAt(1.5)) {
		t.Error("arc endpoints must match the branch at Theta0/Theta1")
	}
	if ptFar(arc.PointAt(0.5), h.PointAt(0.5)) {
		t.Error("arc midpoint must match the branch at the mid θ")
	}
	lo, hi := arc.Domain()
	if lo != 0 || hi != 1 {
		t.Errorf("arc domain = [%g,%g], want [0,1]", lo, hi)
	}
	const eps = 1e-6
	fd := arc.PointAt(0.5 - eps).VectorTo(arc.PointAt(0.5 + eps)).Scale(math.Scalar(1 / (2 * eps)))
	if d := arc.TangentAt(0.5).Sub(fd).Length(); float64(d) > 1e-4 {
		t.Errorf("arc tangent %v vs finite-diff %v (|Δ|=%g)", arc.TangentAt(0.5), fd, d)
	}
}

// CurveParamAtPoint3 inverts a hyperbola branch in closed form: the parameter of a point on the
// branch round-trips to the θ (or arc t) that produced it.
func TestHyperbolaParamRoundTrip(t *testing.T) {
	t.Parallel()
	h, _ := NewHyperbola(math.P3(1, -2, 3), math.V3(0, 0, 1), math.V3(0, 1, 0), 2, 1.5)
	lo, hi := h.Domain()
	if !stdmath.IsInf(lo, -1) || !stdmath.IsInf(hi, 1) {
		t.Errorf("Hyperbola.Domain = [%g,%g], want unbounded", lo, hi)
	}
	for _, theta := range []float64{-1.3, 0, 0.9} {
		got, nature := CurveParamAtPoint3(h, h.PointAt(theta))
		if nature != UniqueSolution || stdmath.Abs(got-theta) > 1e-9 {
			t.Errorf("Hyperbola param at θ=%g: got %g (%v)", theta, got, nature)
		}
	}
	arc := h.Arc(-1, 2)
	for _, ts := range []float64{0.1, 0.5, 0.95} {
		got, _ := CurveParamAtPoint3(arc, arc.PointAt(ts))
		if stdmath.Abs(got-ts) > 1e-9 {
			t.Errorf("HyperbolicArc param at t=%g: got %g", ts, got)
		}
	}
}

func ptFar(a, b math.Point3) bool { return float64(a.DistanceTo(b)) > 1e-9 }
