// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestRegularPolygonFullyConstrained verifies the auto-regularity constraints make a
// polygon a rigid regular n-gon: grounding its centre, dimensioning the circumradius
// (centre→a vertex) and locking one vertex's rotation drives DOF to 0, and the solved
// shape is regular (equal edges, all vertices on the circumradius).
func TestRegularPolygonFullyConstrained(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	lines, center, err := s.AddPolygon(math.P2(0, 0), math.P2(2, 0), 6, true)
	if err != nil {
		t.Fatalf("AddPolygon: %v", err)
	}
	v0 := lines[0].(*Line).StartPoint()

	g := s.GeometricConstraints()
	g.AddGround(center)
	g.AddHorizontal(center, v0) // lock rotation: first vertex due-east of centre
	if _, err := s.DimensionConstraints().AddDistance(center, v0, "2.5 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}

	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if dof := s.DegreesOfFreedom(); dof != 0 {
		t.Fatalf("regular hexagon DOF = %d, want 0 (fully constrained)", dof)
	}

	// Regular: every vertex at the dimensioned circumradius 2.5, all edges equal.
	for _, l := range lines {
		ln := l.(*Line)
		for _, p := range []*Point{ln.StartPoint(), ln.EndPoint()} {
			if d := center.Position().DistanceTo(p.Position()); stdmath.Abs(float64(d)-2.5) > 1e-6 {
				t.Errorf("vertex at radius %v, want 2.5", d)
			}
		}
	}
	want := float64(lines[0].(*Line).Length())
	for _, l := range lines {
		if got := float64(l.(*Line).Length()); stdmath.Abs(got-want) > 1e-6 {
			t.Errorf("edge length %v, want %v (not regular)", got, want)
		}
	}
}
