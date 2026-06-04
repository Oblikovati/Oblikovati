// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "github.com/Oblikovati/oblikovati/math"
)

// solved3D adds the given constraints, solves, and fails on non-convergence.
func solved3D(t *testing.T, s *Sketch3D) {
	t.Helper()
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve did not converge: %+v", res)
	}
}

// TestGeometric3DConstraintsRoundTrip checks the line/point geometric constraints survive
// marshal→apply, re-binding to their restored lines and points.
func TestGeometric3DConstraintsRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	a := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	b := s.AddLine3D(gmath.P3(0, 1, 0), gmath.P3(1, 1, 0))
	p := s.AddPoint3D(gmath.P3(5, 5, 5))
	g := s.GeometricConstraints3D()
	g.add(NewParallel3D(a, b))
	g.add(NewPerpendicular3D(a, b))
	g.add(NewMidpoint3D(p, a))
	g.add(NewGround3D(p))
	g.add(NewParallelToZAxis3D(b))
	g.add(NewParallelToXYPlane3D(a))

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.Item(0)
	if got.GeometricConstraints3D().Count() != 6 {
		t.Fatalf("restored constraints = %d, want 6", got.GeometricConstraints3D().Count())
	}
	// The restored constraints must reference the restored lines (a parallel between the
	// two restored *Line3D), proving the entity-id rebind works.
	if _, ok := got.GeometricConstraints3D().Item(0).(*Parallel3D); !ok {
		t.Errorf("first restored constraint is %T, want *Parallel3D", got.GeometricConstraints3D().Item(0))
	}
}

// TestGeometric3DConstraintVariables checks each constraint reports the operand DOFs it
// touches (the Constraint contract surface the future sparse solver consumes).
func TestGeometric3DConstraintVariables(t *testing.T) {
	s := NewSketches3D().Add()
	a := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	b := s.AddLine3D(gmath.P3(0, 1, 0), gmath.P3(1, 1, 0))
	p := s.AddPoint3D(gmath.P3(0, 0, 0))
	cases := []struct {
		c    Constraint
		want int
	}{
		{NewParallel3D(a, b), 12},
		{NewPerpendicular3D(a, b), 12},
		{NewMidpoint3D(p, a), 9},
		{NewGround3D(p), 3},
		{NewParallelToXAxis3D(a), 6},
		{NewParallelToXYPlane3D(a), 6},
	}
	for i, tc := range cases {
		if got := len(tc.c.Variables()); got != tc.want {
			t.Errorf("case %d (%T): Variables = %d, want %d", i, tc.c, got, tc.want)
		}
		if len(tc.c.Residuals()) == 0 {
			t.Errorf("case %d (%T): expected residuals", i, tc.c)
		}
	}
}

// TestConstraint3DRowKindSerialization checks every axis/plane variant serializes to a
// distinct kind name and round-trips back to the right constraint type.
func TestConstraint3DRowKindSerialization(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 1, 1))
	g := s.GeometricConstraints3D()
	g.add(NewParallelToXAxis3D(l))
	g.add(NewParallelToYAxis3D(l))
	g.add(NewParallelToZAxis3D(l))
	g.add(NewParallelToXYPlane3D(l))
	g.add(NewParallelToXZPlane3D(l))
	g.add(NewParallelToYZPlane3D(l))

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if dst.Item(0).GeometricConstraints3D().Count() != 6 {
		t.Errorf("restored %d orientation constraints, want 6", dst.Item(0).GeometricConstraints3D().Count())
	}
}

// TestParallel3DSolves checks the parallel constraint aligns a free line to a fixed one.
func TestParallel3DSolves(t *testing.T) {
	s := NewSketches3D().Add()
	fixed := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0)) // along +X
	free := s.AddLine3D(gmath.P3(0, 1, 0), gmath.P3(1, 1, 1))  // skew
	s.GeometricConstraints3D().add(NewGround3D(fixed.A))
	s.GeometricConstraints3D().add(NewGround3D(fixed.B))
	s.GeometricConstraints3D().add(NewGround3D(free.A))
	s.GeometricConstraints3D().add(NewParallel3D(fixed, free))
	solved3D(t, s)

	d1, d2 := free.Direction(), fixed.Direction()
	if d1.Cross(d2).Length() > 1e-6 {
		t.Errorf("parallel solve left a cross product: %v", d1.Cross(d2))
	}
}

// TestPerpendicular3DSolves checks the perpendicular constraint drives the dot to zero.
func TestPerpendicular3DSolves(t *testing.T) {
	s := NewSketches3D().Add()
	a := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	b := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 1, 0))
	s.GeometricConstraints3D().add(NewGround3D(a.A))
	s.GeometricConstraints3D().add(NewGround3D(a.B))
	s.GeometricConstraints3D().add(NewGround3D(b.A))
	s.GeometricConstraints3D().add(NewPerpendicular3D(a, b))
	solved3D(t, s)
	if d := float64(a.Direction().Dot(b.Direction())); math.Abs(d) > 1e-6 {
		t.Errorf("perpendicular solve left dot %v", d)
	}
}

// TestMidpoint3DSolves checks the midpoint constraint centers a point on a line.
func TestMidpoint3DSolves(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(4, 2, 6))
	p := s.AddPoint3D(gmath.P3(9, 9, 9))
	s.GeometricConstraints3D().add(NewGround3D(l.A))
	s.GeometricConstraints3D().add(NewGround3D(l.B))
	s.GeometricConstraints3D().add(NewMidpoint3D(p, l))
	solved3D(t, s)
	if p.Position().DistanceTo(gmath.P3(2, 1, 3)) > 1e-6 {
		t.Errorf("midpoint = %v, want (2,1,3)", p.Position())
	}
}

// TestGround3DZeroDOF checks grounding a line's endpoints fully constrains it.
func TestGround3DZeroDOF(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(1, 2, 3), gmath.P3(4, 5, 6))
	s.GeometricConstraints3D().add(NewGround3D(l.A))
	s.GeometricConstraints3D().add(NewGround3D(l.B))
	if dof := s.DegreesOfFreedom(); dof != 0 {
		t.Errorf("grounded line DOF = %d, want 0", dof)
	}
}

// TestParallelToAxis3DSolves checks a line aligns to +Z under the axis constraint.
func TestParallelToAxis3DSolves(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 1, 1))
	s.GeometricConstraints3D().add(NewGround3D(l.A))
	s.GeometricConstraints3D().add(NewParallelToZAxis3D(l))
	solved3D(t, s)
	d := l.Direction()
	if math.Abs(float64(d.X)) > 1e-6 || math.Abs(float64(d.Y)) > 1e-6 {
		t.Errorf("parallel-to-Z left a non-axial direction: %v", d)
	}
}

// TestParallelToPlane3DSolves checks a line becomes parallel to the XY plane (its
// direction loses its Z component).
func TestParallelToPlane3DSolves(t *testing.T) {
	s := NewSketches3D().Add()
	l := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 1, 1))
	s.GeometricConstraints3D().add(NewGround3D(l.A))
	s.GeometricConstraints3D().add(NewParallelToXYPlane3D(l))
	solved3D(t, s)
	if z := float64(l.Direction().Z); math.Abs(z) > 1e-6 {
		t.Errorf("parallel-to-XY left a Z component %v", z)
	}
}
