// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestCircleOnPlaneAndRadius(t *testing.T) {
	c, err := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Every sampled point is exactly Radius from Center and in the z=0 plane.
	for i := 0; i <= 8; i++ {
		p := c.PointAt(float64(i) / 8)
		approxScalar(t, p.DistanceTo(c.Center), 2, "radius")
		approxScalar(t, p.Z, 0, "planarity")
	}
	// Closed: start meets end.
	if !c.PointAt(0).IsEqualTo(c.PointAt(1), 1e-9) {
		t.Error("circle should be closed")
	}
}

func TestCircleTangentPerpendicularToRadius(t *testing.T) {
	c, _ := NewCircle(math.P3(1, 2, 3), math.V3(0, 0, 1), 5)
	for _, tp := range []float64{0, 0.1, 0.37, 0.8} {
		radial := c.Center.VectorTo(c.PointAt(tp))
		if !radial.IsPerpendicularTo(c.TangentAt(tp), 1e-9) {
			t.Errorf("tangent not perpendicular to radius at t=%v", tp)
		}
	}
}

func TestCircle2dByThreePoints(t *testing.T) {
	// Unit circle sampled at three points → center origin, radius 1.
	c, err := Circle2dByThreePoints(math.P2(1, 0), math.P2(0, 1), math.P2(-1, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !c.Center.IsEqualTo(math.P2(0, 0), 1e-12) {
		t.Errorf("center = %v, want origin", c.Center)
	}
	approxScalar(t, c.Radius, 1, "radius")
}

func TestCircle2dByThreePointsCollinearFails(t *testing.T) {
	_, err := Circle2dByThreePoints(math.P2(0, 0), math.P2(1, 0), math.P2(2, 0))
	if err == nil {
		t.Fatal("expected collinear error")
	}
	if _, ok := err.(*CollinearPointsError); !ok {
		t.Errorf("error type = %T, want *CollinearPointsError", err)
	}
}
