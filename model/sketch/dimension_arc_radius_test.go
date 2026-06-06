// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

// TestAddRadiusOnArc verifies a radius dimension drives an arc (not just a circle),
// matching Inventor. The arc starts at radius 3; dimensioning it to 5 must, after a
// solve, leave the arc's center-to-start distance at 5.
func TestAddRadiusOnArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.GeometricConstraints().AddGround(s.Points().Add(math.P2(0, 0)))
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(3, 0), math.P2(0, 3), true)

	dim, err := s.DimensionConstraints().AddRadius(arc, "5 cm")
	if err != nil {
		t.Fatalf("AddRadius on arc: %v", err)
	}
	if dim.Kind() != RadiusDim {
		t.Errorf("dim kind = %v, want RadiusDim", dim.Kind())
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if got := float64(arc.Radius()); stdmath.Abs(got-5) > 1e-6 {
		t.Errorf("arc radius after dimension = %.6f, want 5", got)
	}
}

// TestAddDiameterOnArc checks the diameter dimension also accepts an arc.
func TestAddDiameterOnArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.GeometricConstraints().AddGround(s.Points().Add(math.P2(0, 0)))
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)
	if _, err := s.DimensionConstraints().AddDiameter(arc, "10 cm"); err != nil {
		t.Fatalf("AddDiameter on arc: %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if got := float64(arc.Radius()); stdmath.Abs(got-5) > 1e-6 {
		t.Errorf("arc radius after diameter=10 = %.6f, want 5", got)
	}
}
