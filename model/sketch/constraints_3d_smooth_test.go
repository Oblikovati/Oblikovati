// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// residualsNear asserts every residual component is within tol of zero.
func residualsNear(t *testing.T, c Constraint, tol float64) {
	t.Helper()
	for i, r := range c.Residuals() {
		if stdmath.Abs(r) > tol {
			t.Errorf("residual[%d] = %g, want |r| <= %g", i, r, tol)
		}
	}
}

// TestTangent3DLineArcResidualZeroWhenSatisfied builds a line meeting an arc
// tangentially in the XY plane: line along +X ending at (1,0,0), arc centered at
// (1,1,0) through (1,0,0) — the radius is ⟂ the line there, so they are tangent.
func TestTangent3DLineArcResidualZeroWhenSatisfied(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	a := s.AddArc3D(gmath.P3(1, 1, 0), gmath.P3(1, 0, 0), gmath.P3(2, 1, 0), false)
	c := NewTangent3D(l, a, l.B, a.Start)
	residualsNear(t, c, 1e-12)
}

// TestTangent3DSolvesPerturbedJoin perturbs the arc start away from the line end and
// checks the solver restores a coincident, tangent join.
func TestTangent3DSolvesPerturbedJoin(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	a := s.AddArc3D(gmath.P3(1, 1, 0), gmath.P3(1.2, 0.1, 0.05), gmath.P3(2, 1, 0), false)
	c := NewTangent3D(l, a, l.B, a.Start)
	s.GeometricConstraints3D().add(c)
	solved3D(t, s)
	residualsNear(t, c, 1e-6)
	if d := float64(l.B.Position().DistanceTo(a.Start.Position())); d > 1e-6 {
		t.Errorf("join endpoints %g apart after solve, want coincident", d)
	}
}

// TestTangent3DRejectsNonEndpoint reports unsatisfiable (constant penalty) when the
// given point is not an endpoint of the curve.
func TestTangent3DRejectsNonEndpoint(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	l2 := s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(2, 1, 0))
	stray := s.AddPoint3D(gmath.P3(9, 9, 9))
	c := NewTangent3D(l1, l2, stray, l2.A)
	for i, r := range c.Residuals() {
		if r != 1 {
			t.Fatalf("residual[%d] = %g, want the constant penalty 1", i, r)
		}
	}
}

// TestSmooth3DSplineLineSolvesG2 joins a line to a spline with G2 continuity: after
// solving, the spline's terminal points must be collinear with the line direction
// (zero curvature matches the straight line).
func TestSmooth3DSplineLineSolvesG2(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(-2, 0, 0), gmath.P3(0, 0, 0))
	sp := s.AddSpline3D([]gmath.Point3{
		{X: 0.2, Y: 0.3, Z: 0}, {X: 1, Y: 0.8, Z: 0.4}, {X: 2, Y: 0.2, Z: 0.9},
	}, false, true)
	c := NewSmooth3D(l, sp, l.B, sp.Points[0])
	s.GeometricConstraints3D().add(c)
	solved3D(t, s)
	residualsNear(t, c, 1e-6)
	if d := float64(l.B.Position().DistanceTo(sp.Points[0].Position())); d > 1e-6 {
		t.Errorf("join endpoints %g apart after solve, want coincident", d)
	}
}

// TestSmooth3DArcArcResidualZeroWhenCocircular checks two arcs of one circle meeting
// at a point are already G2 (equal curvature vectors, collinear tangents).
func TestSmooth3DArcArcResidualZeroWhenCocircular(t *testing.T) {
	s := NewSketches3D().Add()
	// Unit circle around the origin in XY: arc1 from (1,0) to (0,1), arc2 from (0,1)
	// to (-1,0); they meet at (0,1) sharing center and radius.
	a1 := s.AddArc3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0), gmath.P3(0, 1, 0), true)
	a2 := s.AddArc3D(gmath.P3(0, 0, 0), gmath.P3(0, 1, 0), gmath.P3(-1, 0, 0), true)
	c := NewSmooth3D(a1, a2, a1.End, a2.Start)
	residualsNear(t, c, 1e-9)
}

// TestNearestEndpointPair3D picks the closest endpoints of two separated lines and
// reports no pair for a closed spline (no free endpoints).
func TestNearestEndpointPair3D(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	l2 := s.AddLine3D(gmath.P3(1.1, 0, 0), gmath.P3(5, 5, 5))
	p1, p2, ok := NearestEndpointPair3D(l1, l2)
	if !ok || p1 != l1.B || p2 != l2.A {
		t.Errorf("nearest pair = (%v,%v,%v), want l1.B/l2.A", p1, p2, ok)
	}
	closed := s.AddSpline3D([]gmath.Point3{{X: 0, Y: 0, Z: 0}, {X: 1, Y: 0, Z: 0}, {X: 0, Y: 1, Z: 0}}, true, true)
	if _, _, ok := NearestEndpointPair3D(l1, closed); ok {
		t.Error("a closed spline must report no free endpoints")
	}
}

// TestSmoothJoin3DRoundTrip checks tangent + smooth survive marshal→apply, re-binding
// to the restored curves and endpoints.
func TestSmoothJoin3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	a := s.AddArc3D(gmath.P3(1, 1, 0), gmath.P3(1, 0, 0), gmath.P3(2, 1, 0), false)
	sp := s.AddSpline3D([]gmath.Point3{{X: 2, Y: 1, Z: 0}, {X: 3, Y: 2, Z: 1}, {X: 4, Y: 1, Z: 0}}, false, true)
	s.GeometricConstraints3D().add(NewTangent3D(l, a, l.B, a.Start))
	s.GeometricConstraints3D().add(NewSmooth3D(a, sp, a.End, sp.Points[0]))

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.Item(0).GeometricConstraints3D()
	if got.Count() != 2 {
		t.Fatalf("restored constraints = %d, want 2", got.Count())
	}
	tan, ok := got.Item(0).(*Tangent3D)
	if !ok {
		t.Fatalf("restored[0] = %T, want *Tangent3D", got.Item(0))
	}
	residualsNear(t, tan, 1e-9) // satisfied layout must restore satisfied
	if _, ok := got.Item(1).(*Smooth3D); !ok {
		t.Fatalf("restored[1] = %T, want *Smooth3D", got.Item(1))
	}
}
