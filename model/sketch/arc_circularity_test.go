// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// arcRadii returns an arc's center-to-start and center-to-end distances.
func arcRadii(a *Arc) (start, end float64) {
	c := a.Center.Position()
	return float64(c.DistanceTo(a.Start.Position())), float64(c.DistanceTo(a.End.Position()))
}

// assertCircular fails unless the arc's two endpoints are equidistant from its center.
func assertCircular(t *testing.T, name string, a *Arc) {
	t.Helper()
	s, e := arcRadii(a)
	if stdmath.Abs(s-e) > 1e-6 {
		t.Errorf("%s: arc not circular — |center−start| = %g, |center−end| = %g", name, s, e)
	}
}

// TestArcCarriesCircularityConstraint is acceptance criterion 1 of #1419: an arc's End is no
// longer a free-floating point — it is tied to the arc radius by an internal constraint that
// the solver consumes.
func TestArcCarriesCircularityConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)
	found := false
	for _, c := range s.Constraints() {
		if cc, ok := c.(*arcCircularityConstraint); ok && cc.arc == arc {
			found = true
		}
	}
	if !found {
		t.Fatal("arc has no circularity constraint among the sketch's solver constraints")
	}
}

// TestSolveRestoresArcCircularity is the core regression for #1419: nudging End off the
// radius and solving pulls it back onto the circle — without the internal constraint nothing
// would, leaving a non-circular "arc".
func TestSolveRestoresArcCircularity(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)
	g := s.GeometricConstraints()
	g.AddFix(arc.Center)
	g.AddFix(arc.Start)
	arc.End.SetPosition(math.P2(0, 3.5)) // pushed off the radius-2 circle

	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	assertCircular(t, "after solve", arc)
	if _, e := arcRadii(arc); stdmath.Abs(e-2) > 1e-6 {
		t.Errorf("end radius %g, want 2 (the fixed start radius)", e)
	}
}

// TestDriveArcRadiusKeepsEndOnCircle is the issue's first test case: driving the arc radius
// moves Center+Start, and End must follow to the new radius rather than being left behind.
func TestDriveArcRadiusKeepsEndOnCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(2, 0), math.P2(0, 2), true)
	g := s.GeometricConstraints()
	g.AddFix(arc.Center)
	if _, err := s.DimensionConstraints().AddRadius(arc, "3 cm"); err != nil {
		t.Fatalf("AddRadius: %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	assertCircular(t, "driven radius", arc)
	start, end := arcRadii(arc)
	if stdmath.Abs(start-3) > 1e-6 || stdmath.Abs(end-3) > 1e-6 {
		t.Errorf("radii after driving to 3: start %g, end %g, want both 3", start, end)
	}
}

// TestTangentOnArcKeepsEndOnCircle is the issue's second test case: a tangent constraint
// drives the arc (via its center/start), and its End stays on the circle.
func TestTangentOnArcKeepsEndOnCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1.5, 0), math.P2(0, 1.5), true)
	g := s.GeometricConstraints()
	g.AddFix(arc.Center)
	// A fixed horizontal line at y = 2; tangency drives the arc radius to 2.
	line := s.Lines().AddByTwoPoints(math.P2(-3, 2), math.P2(3, 2))
	g.AddFix(line.A)
	g.AddFix(line.B)
	g.AddTangent(line, arc)

	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	assertCircular(t, "tangent-driven", arc)
}

// TestValidArcRoundTripsThroughSolve is the issue's third test case: a well-formed arc that
// already satisfies its circularity survives an unconstrained solve unchanged (the internal
// constraint adds no spurious motion).
func TestValidArcRoundTripsThroughSolve(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	arc := s.Arcs().AddByCenterStartEnd(math.P2(1, 1), math.P2(4, 1), math.P2(1, 4), true) // radius 3 both ends
	before := arc.End.Position()
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: residual %v", r.Residual)
	}
	assertCircular(t, "round-trip", arc)
	if d := before.DistanceTo(arc.End.Position()); float64(d) > 1e-9 {
		t.Errorf("a valid arc's End moved by %g during an unconstrained solve, want 0", d)
	}
}
