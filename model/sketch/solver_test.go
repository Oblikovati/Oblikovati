// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
	"oblikovati/model/param"
)

func TestFullyConstrainedSketchSolvesToUniqueSolution(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(3, 0.4)) // rough start, off-axis
	g := s.GeometricConstraints()
	g.AddFix(p0)                                                // pin p0 (2 eq)
	g.AddHorizontal(p0, p1)                                     // p0.y == p1.y (1 eq)
	_, _ = s.DimensionConstraints().AddDistance(p0, p1, "5 cm") // |p0p1| == 5 (1 eq)
	// 4 variables, 4 independent equations → 0 DOF.

	r := s.Solve()
	if !r.Converged {
		t.Fatalf("did not converge: residual=%v", r.Residual)
	}
	if r.DOF != 0 || r.Status != WellConstrained {
		t.Errorf("DOF=%d status=%v, want 0 / well-constrained", r.DOF, r.Status)
	}
	if !p0.Position().IsEqualTo(math.P2(0, 0), 1e-6) {
		t.Errorf("fixed point moved to %v", p0.Position())
	}
	if !approx(p1.Y, 0) || !approx(stdmath.Abs(p1.X), 5) {
		t.Errorf("p1 = %v, want y=0 and |x|=5", p1.Position())
	}
	if !s.Health().OK() {
		t.Errorf("healthy sketch reports %+v", s.Health())
	}
}

func TestEditingDimensionResolvesStably(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(5, 0))
	g := s.GeometricConstraints()
	g.AddFix(p0)
	g.AddHorizontal(p0, p1)
	dim, _ := s.DimensionConstraints().AddDistance(p0, p1, "5 cm")
	if r := s.Solve(); !r.Converged {
		t.Fatal("initial solve failed")
	}

	// Edit the driving dimension; warm-started re-solve should land exactly.
	if err := dim.Parameter().SetExpression("8 cm"); err != nil {
		t.Fatalf("SetExpression: %v", err)
	}
	r := s.Solve()
	if !r.Converged || !approx(p1.X, 8) || !approx(p1.Y, 0) {
		t.Errorf("after edit p1=%v converged=%v, want (8,0)", p1.Position(), r.Converged)
	}
}

func TestUnderConstrainedReportsFreedom(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(3, 0))
	s.GeometricConstraints().AddFix(p0)                         // 2 eq
	_, _ = s.DimensionConstraints().AddDistance(p0, p1, "5 cm") // 1 eq
	// 4 vars, 3 independent eq → 1 DOF (p1 free on a circle).
	a := s.AnalyzeConstraints()
	if a.DOF != 1 || a.Status != UnderConstrained {
		t.Errorf("DOF=%d status=%v, want 1 / under-constrained", a.DOF, a.Status)
	}
	if s.DegreesOfFreedom() != 1 {
		t.Errorf("DegreesOfFreedom = %d, want 1", s.DegreesOfFreedom())
	}
}

func TestRedundantConstraintIsFlagged(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(5, 0))
	g := s.GeometricConstraints()
	g.AddFix(p0)
	g.AddHorizontal(p0, p1)
	_, _ = s.DimensionConstraints().AddDistance(p0, p1, "5 cm") // fully constrained: 0 DOF
	if a := s.AnalyzeConstraints(); a.Status != WellConstrained {
		t.Fatalf("baseline not well-constrained: %+v", a)
	}
	// A second, redundant distance dimension adds a dependent equation.
	_, _ = s.DimensionConstraints().AddDistance(p0, p1, "5 cm")
	a := s.AnalyzeConstraints()
	if a.Redundant < 1 || a.Status != OverConstrained {
		t.Errorf("redundant constraint not flagged: %+v", a)
	}
	// Solving an over-constrained-but-consistent sketch is a warning, not sick.
	if r := s.Solve(); !r.Converged || s.Health().OK() {
		t.Errorf("expected converged+warning, got converged=%v health=%+v", r.Converged, s.Health())
	}
}

func TestConflictingConstraintsGoSick(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p0 := s.Points().Add(math.P2(0, 0))
	p1 := s.Points().Add(math.P2(5, 0))
	g := s.GeometricConstraints()
	g.AddFix(p0)
	g.AddFix(p1) // pins p1 at (5,0)
	// Demand a distance of 9 between two pinned points — impossible.
	_, _ = s.DimensionConstraints().AddDistance(p0, p1, "9 cm")
	r := s.Solve()
	if r.Converged {
		t.Error("conflicting system reported convergence")
	}
	if s.Health().OK() {
		t.Error("conflicting sketch should be sick")
	}
}

func TestSolve3DViaCore(t *testing.T) {
	// 3D distance between two points, one pinned at origin, target 5 along Z.
	a := NewPoint3D(math.P3(0, 0, 0))
	b := NewPoint3D(math.P3(0, 0, 2))
	ps := param.NewParameters()
	dc := NewDimensionConstraints3D(ps)
	dim, _ := dc.AddDistance(a, b, "5 cm")
	fix := NewCustomConstraint3D(
		func() []float64 { return []float64{a.X, a.Y, a.Z} },
		[]*math.Scalar{&a.X, &a.Y, &a.Z},
	)
	cons := []Constraint{fix, dim}
	vars := []*math.Scalar{&a.X, &a.Y, &a.Z, &b.X, &b.Y, &b.Z}

	r := Solve(cons, vars, Options{})
	if !r.Converged {
		t.Fatalf("3D solve did not converge: residual=%v", r.Residual)
	}
	if !approx(a.Position().DistanceTo(math.P3(0, 0, 0)), 0) {
		t.Error("pinned 3D point moved")
	}
	if !approx(a.Position().DistanceTo(b.Position()), 5) {
		t.Errorf("3D distance = %v, want 5", a.Position().DistanceTo(b.Position()))
	}
}

func TestEmptySketchSolvesTrivially(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	r := s.Solve()
	if !r.Converged || r.DOF != 0 {
		t.Errorf("empty sketch: converged=%v dof=%d", r.Converged, r.DOF)
	}
}

func TestSolveStatusStrings(t *testing.T) {
	cases := map[SolveStatus]string{
		WellConstrained: "well-constrained", UnderConstrained: "under-constrained", OverConstrained: "over-constrained",
	}
	for st, want := range cases {
		if st.String() != want {
			t.Errorf("SolveStatus(%d).String() = %q, want %q", st, st.String(), want)
		}
	}
}

func TestEllipseRadiiAreDegreesOfFreedom(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 4, 2) // center(2) + 2 radii = 4 free vars
	a := s.AnalyzeConstraints()
	if a.Variables != 4 || a.DOF != 4 {
		t.Errorf("ellipse DOF universe = %d vars / %d DOF, want 4 / 4", a.Variables, a.DOF)
	}
}
