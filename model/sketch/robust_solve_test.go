// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// trilaterationFixture pins a point by two distance dimensions to two fixed anchors, the
// whole thing scaled by k (geometry AND dimension targets) — a uniformly scaled copy of one
// sketch. It is well-constrained (DOF 0) and needs a solve to place the free point.
func trilaterationFixture(k float64) *Sketch {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(math.Scalar(4*k), 0))
	g.AddFix(a)
	g.AddFix(b)
	p := s.Points().Add(math.P2(math.Scalar(1.5*k), math.Scalar(0.5*k))) // off the solution
	_, _ = s.DimensionConstraints().AddDistance(a, p, fmt.Sprintf("%g cm", 3*k))
	_, _ = s.DimensionConstraints().AddDistance(b, p, fmt.Sprintf("%g cm", 3*k))
	return s
}

// TestScaleSweepConvergenceAndDOFParity is acceptance criterion 2 of #1420: µm-scale and
// m-scale copies of one sketch converge identically and classify identically, because the
// convergence tolerance and the rank/DOF tolerance are relative to the model scale.
func TestScaleSweepConvergenceAndDOFParity(t *testing.T) {
	type outcome struct {
		converged bool
		dof       int
		status    SolveStatus
	}
	run := func(k float64) outcome {
		s := trilaterationFixture(k)
		r := s.Solve()
		return outcome{r.Converged, s.DegreesOfFreedom(), s.AnalyzeConstraints().Status}
	}
	base := run(1)
	if !base.converged || base.dof != 0 {
		t.Fatalf("unit-scale fixture: converged=%v dof=%d, want converged with DOF 0", base.converged, base.dof)
	}
	for _, k := range []float64{1e-4, 1e-2, 1e2, 1e4} { // µm … hundreds of metres (kernel cm units)
		got := run(k)
		if got != base {
			t.Errorf("scale %g classified differently: %+v vs unit-scale %+v", k, got, base)
		}
	}
}

// TestMixedAngleDistanceConverges is acceptance criterion 2's mixed-unit case: a sketch
// combining an angle dimension (radians, O(1)) and a distance dimension (length, O(10²))
// converges — the nondimensionalised convergence test is not dominated by the larger unit.
func TestMixedAngleDistanceConverges(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	o := s.Points().Add(math.P2(0, 0))
	g.AddFix(o)
	pa := s.Points().Add(math.P2(5, 0))
	pb := s.Points().Add(math.P2(4, 1)) // ~14° off; will be driven to 30°
	l1 := s.Lines().Add(o, pa)
	l2 := s.Lines().Add(o, pb)
	g.AddFix(pa) // pin the reference arm
	if _, err := s.DimensionConstraints().AddDistance(o, pb, "5 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if _, err := s.DimensionConstraints().AddAngle(l1, l2, "30 deg"); err != nil {
		t.Fatalf("AddAngle: %v", err)
	}
	if r := s.Solve(); !r.Converged {
		t.Fatalf("mixed angle+distance solve did not converge: residual %g", r.Residual)
	}
	if d := float64(o.Position().DistanceTo(pb.Position())); stdmath.Abs(d-5) > 1e-6 {
		t.Errorf("distance %v after solve, want 5", d)
	}
}

// TestNearParallelConverges is acceptance criterion 1's near-parallel case: two almost-
// parallel lines driven to exactly parallel — an ill-conditioned system where the QR step
// (no squared condition number) converges.
func TestNearParallelConverges(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	l1 := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))
	g.AddFix(l1.A)
	g.AddFix(l1.B)
	l2 := s.Lines().AddByTwoPoints(math.P2(0, 1), math.P2(10, 1.0001)) // 0.0006° off parallel
	g.AddFix(l2.A)
	g.AddParallel(l1, l2)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("near-parallel solve did not converge: residual %g", r.Residual)
	}
	// l2 must become exactly parallel to l1 (l1 is horizontal): l2's free end shares l2.A's Y.
	if dy := float64(l2.B.Y - l2.A.Y); stdmath.Abs(dy) > 1e-7 {
		t.Errorf("lines not parallel after solve: Δy = %v", dy)
	}
}

// TestConflictDiagnosisReturnsOffendingConstraint is acceptance criterion 3 of #1420: an
// unsatisfiable sketch reports the offending constraint subset, not a bare failure. Two
// points are fixed 3 apart while a distance dimension demands 10 — only the dimension
// cannot be satisfied.
func TestConflictDiagnosisReturnsOffendingConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(3, 0))
	g.AddFix(a)
	g.AddFix(b)
	dim, err := s.DimensionConstraints().AddDistance(a, b, "10 cm")
	if err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	if r := s.Solve(); r.Converged {
		t.Fatal("expected the contradictory sketch not to solve")
	}
	// The least-squares solve distributes the irreconcilable error across the participating
	// constraints (sorted most-severe first), so the diagnosis returns the offending subset
	// — not a bare failure — and the unsatisfiable distance dimension is among it.
	conflicts := s.ConflictingConstraints()
	if len(conflicts) == 0 {
		t.Fatal("conflict diagnosis returned no offending constraints for a contradictory sketch")
	}
	found := false
	for _, c := range conflicts {
		if c == Constraint(dim) {
			found = true
		}
	}
	if !found {
		t.Errorf("conflict subset %v does not include the unsatisfiable distance dimension", conflicts)
	}
}
