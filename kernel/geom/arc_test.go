// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestArc2dEvaluation(t *testing.T) {
	// Quarter arc on the unit circle, 0 → π/2.
	a := NewArc2d(math.P2(0, 0), 1, 0, stdmath.Pi/2)
	if got := a.PointAt(0); !got.IsEqualTo(math.P2(1, 0), eqScalar) {
		t.Errorf("PointAt(0) = %v, want {1 0}", got)
	}
	if got := a.PointAt(1); !got.IsEqualTo(math.P2(0, 1), eqScalar) {
		t.Errorf("PointAt(1) = %v, want {0 1}", got)
	}
	approxScalar(t, a.Length(), stdmath.Pi/2, "Length")
}

func TestArc2dByThreePointsReproducesCenterRadius(t *testing.T) {
	// Three points on the unit circle, counter-clockwise.
	a, err := Arc2dByThreePoints(math.P2(1, 0), math.P2(0, 1), math.P2(-1, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Center.IsEqualTo(math.P2(0, 0), 1e-12) {
		t.Errorf("center = %v, want origin", a.Center)
	}
	approxScalar(t, a.Radius, 1, "radius")
	// Endpoints and the through-point are reproduced.
	if !a.PointAt(0).IsEqualTo(math.P2(1, 0), 1e-12) {
		t.Errorf("start = %v, want {1 0}", a.PointAt(0))
	}
	if !a.PointAt(1).IsEqualTo(math.P2(-1, 0), 1e-12) {
		t.Errorf("end = %v, want {-1 0}", a.PointAt(1))
	}
	if !a.PointAt(0.5).IsEqualTo(math.P2(0, 1), 1e-12) {
		t.Errorf("mid = %v, want {0 1}", a.PointAt(0.5))
	}
}

func TestArc2dByThreePointsClockwise(t *testing.T) {
	// Same endpoints but through (0,-1): the arc must go the other way.
	a, err := Arc2dByThreePoints(math.P2(1, 0), math.P2(0, -1), math.P2(-1, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if a.SweepAngle >= 0 {
		t.Errorf("sweep = %v, want negative (clockwise)", a.SweepAngle)
	}
	if !a.PointAt(0.5).IsEqualTo(math.P2(0, -1), 1e-12) {
		t.Errorf("mid = %v, want {0 -1}", a.PointAt(0.5))
	}
}

func TestArc3dByThreePointsReproducesCenterRadius(t *testing.T) {
	// Unit circle lifted to the z=5 plane.
	a, err := Arc3dByThreePoints(math.P3(1, 0, 5), math.P3(0, 1, 5), math.P3(-1, 0, 5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !a.Center.IsEqualTo(math.P3(0, 0, 5), 1e-12) {
		t.Errorf("center = %v, want {0 0 5}", a.Center)
	}
	approxScalar(t, a.Radius, 1, "radius")
	if !a.PointAt(0).IsEqualTo(math.P3(1, 0, 5), 1e-12) {
		t.Errorf("start = %v, want {1 0 5}", a.PointAt(0))
	}
	if !a.PointAt(1).IsEqualTo(math.P3(-1, 0, 5), 1e-12) {
		t.Errorf("end = %v, want {-1 0 5}", a.PointAt(1))
	}
	// The through-point lies on the arc.
	if !a.PointAt(0.5).IsEqualTo(math.P3(0, 1, 5), 1e-9) {
		t.Errorf("mid = %v, want {0 1 5}", a.PointAt(0.5))
	}
}

func TestArc3dByThreePointsCollinearFails(t *testing.T) {
	_, err := Arc3dByThreePoints(math.P3(0, 0, 0), math.P3(1, 1, 1), math.P3(2, 2, 2))
	if err == nil {
		t.Fatal("expected collinear error")
	}
	if _, ok := err.(*CollinearPoints3dError); !ok {
		t.Errorf("error type = %T, want *CollinearPoints3dError", err)
	}
}

func TestArc3dTangentPerpendicularToRadius(t *testing.T) {
	a, _ := NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 3, 0, stdmath.Pi)
	for _, tp := range []float64{0, 0.25, 0.5, 1} {
		radial := a.Center.VectorTo(a.PointAt(tp))
		if !radial.IsPerpendicularTo(a.TangentAt(tp), 1e-9) {
			t.Errorf("tangent not perpendicular to radius at t=%v", tp)
		}
	}
}
