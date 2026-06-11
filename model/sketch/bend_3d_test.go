// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	gmath "oblikovati.org/math"
)

// rightAngleRails builds the bend fixture: two chained lines forming a right angle at
// (1,0,0) — the corner Line3DTool would produce (coincident but distinct endpoints).
func rightAngleRails(t *testing.T) (*Sketch3D, *Line3D, *Line3D) {
	t.Helper()
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	l2 := s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(1, 1, 0))
	return s, l1, l2
}

func TestAddBend3DTrimsAndStaysTangent(t *testing.T) {
	s, l1, l2 := rightAngleRails(t)
	arc, err := s.AddBend3D(l1, l2, 0.25)
	if err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}
	// Right angle ⇒ trim = r: line1 now ends at (0.75,0,0), line2 starts at (1,0.25,0),
	// the arc center sits at (0.75,0.25,0) with radius 0.25.
	if got := l1.B.Position(); float64(got.DistanceTo(gmath.P3(0.75, 0, 0))) > 1e-9 {
		t.Errorf("line1 trimmed end = %v, want (0.75,0,0)", got)
	}
	if got := l2.A.Position(); float64(got.DistanceTo(gmath.P3(1, 0.25, 0))) > 1e-9 {
		t.Errorf("line2 trimmed end = %v, want (1,0.25,0)", got)
	}
	if got := arc.Center.Position(); float64(got.DistanceTo(gmath.P3(0.75, 0.25, 0))) > 1e-9 {
		t.Errorf("arc center = %v, want (0.75,0.25,0)", got)
	}
	if r := float64(arc.Radius()); stdmath.Abs(r-0.25) > 1e-9 {
		t.Errorf("arc radius = %v, want 0.25", r)
	}
	// The maintaining constraint exists and is satisfied as built.
	if s.GeometricConstraints3D().Count() != 1 {
		t.Fatalf("constraints = %d, want the auto-added bend", s.GeometricConstraints3D().Count())
	}
	residualsNear(t, s.GeometricConstraints3D().Item(0), 1e-9)
	// The arc shares the trimmed endpoints, so the rail remains one connected chain.
	if paths := s.Paths3D(); len(paths) != 1 {
		t.Errorf("paths after bend = %d, want 1 connected chain", len(paths))
	}
}

func TestBend3DConstraintRestoresTangencyAfterPerturbation(t *testing.T) {
	s, l1, l2 := rightAngleRails(t)
	arc, err := s.AddBend3D(l1, l2, 0.25)
	if err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}
	// Drag the arc's start off the fillet; the solve must restore a tangent join at
	// the held radius.
	arc.Start.SetPosition(gmath.P3(0.6, 0.1, 0.05))
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve did not converge: %+v", res)
	}
	residualsNear(t, s.GeometricConstraints3D().Item(0), 1e-6)
	if r := float64(arc.Radius()); stdmath.Abs(r-0.25) > 1e-6 {
		t.Errorf("radius after solve = %v, want held at 0.25", r)
	}
}

func TestAddBend3DSplitsASharedCornerPoint(t *testing.T) {
	s := NewSketches3D().Add()
	// One TRUE shared endpoint object between the lines (the restore-path shape).
	corner := s.newPoint3D(gmath.P3(1, 0, 0))
	l1 := s.addLine3DPts(s.newPoint3D(gmath.P3(0, 0, 0)), corner)
	l2 := s.addLine3DPts(corner, s.newPoint3D(gmath.P3(1, 1, 0)))
	if _, err := s.AddBend3D(l1, l2, 0.25); err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}
	if l1.B == l2.A {
		t.Error("bend left the corner as one shared point; each line needs its own trim point")
	}
	if got := l2.A.Position(); float64(got.DistanceTo(gmath.P3(1, 0.25, 0))) > 1e-9 {
		t.Errorf("split corner of line2 = %v, want (1,0.25,0)", got)
	}
}

func TestAddBend3DRejectsDegenerateInputs(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	gap := s.AddLine3D(gmath.P3(5, 5, 5), gmath.P3(6, 5, 5))
	if _, err := s.AddBend3D(l1, gap, 0.2); err == nil {
		t.Error("expected an error for lines that do not share an endpoint")
	}
	collinear := s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(2, 0, 0))
	if _, err := s.AddBend3D(l1, collinear, 0.2); err == nil {
		t.Error("expected an error for collinear lines")
	}
	short := s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(1, 0.1, 0))
	if _, err := s.AddBend3D(l1, short, 0.5); err == nil {
		t.Error("expected an error when the radius trims past a line's far end")
	}
	if _, err := s.AddBend3D(l1, short, -1); err == nil {
		t.Error("expected an error for a non-positive radius")
	}
}

func TestBend3DRoundTrip(t *testing.T) {
	src := NewSketches3D()
	s := src.Add()
	l1 := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(1, 0, 0))
	l2 := s.AddLine3D(gmath.P3(1, 0, 0), gmath.P3(1, 1, 0))
	if _, err := s.AddBend3D(l1, l2, 0.25); err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}

	data, err := src.MarshalRecipe3D()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	dst := NewSketches3D()
	if err := dst.ApplyRecipe3D(data); err != nil {
		t.Fatalf("apply: %v", err)
	}
	got := dst.Item(0)
	if got.GeometricConstraints3D().Count() != 1 {
		t.Fatalf("restored constraints = %d, want 1", got.GeometricConstraints3D().Count())
	}
	bend, ok := got.GeometricConstraints3D().Item(0).(*Bend3D)
	if !ok {
		t.Fatalf("restored = %T, want *Bend3D", got.GeometricConstraints3D().Item(0))
	}
	if stdmath.Abs(bend.Radius-0.25) > 1e-12 {
		t.Errorf("restored radius = %v, want 0.25", bend.Radius)
	}
	residualsNear(t, bend, 1e-9) // satisfied layout restores satisfied
	if res := got.Solve(); !res.Converged {
		t.Fatalf("restored sketch does not solve: %+v", res)
	}
}

// TestAddBend3DIsNotRedundant is the regression for the #145 audit finding: a
// fresh AddBend3D shares its join points with the arc, and the constraint used to
// emit six identically-zero G0 rows for them — zero Jacobian rank, so the DOF
// analysis reported the sketch over-constrained and the solve left it unhealthy.
func TestAddBend3DIsNotRedundant(t *testing.T) {
	s, l1, l2 := rightAngleRails(t)
	if _, err := s.AddBend3D(l1, l2, 0.25); err != nil {
		t.Fatalf("AddBend3D: %v", err)
	}
	if a := s.AnalyzeConstraints(); a.Redundant != 0 || a.Status == OverConstrained {
		t.Errorf("DOF analysis = %+v, want no redundant equations", a)
	}
	if res := s.Solve(); !res.Converged || res.Status == OverConstrained {
		t.Errorf("solve = %+v, want converged and not over-constrained", res)
	}
}

// TestBend3DSplitJoinKeepsG0Rows pins the wire-addConstraint shape: when the join
// endpoints are split points (not the arc's own), the two G0 rows per join must
// stay, or the constraint could no longer pull a split bend together on solve.
func TestBend3DSplitJoinKeepsG0Rows(t *testing.T) {
	s := NewSketches3D().Add()
	l1 := s.AddLine3D(gmath.P3(0, 0, 0), gmath.P3(0.75, 0, 0))
	l2 := s.AddLine3D(gmath.P3(1, 0.25, 0), gmath.P3(1, 1, 0))
	arc := s.AddArc3D(gmath.P3(0.75, 0.25, 0), gmath.P3(0.76, 0, 0), gmath.P3(1, 0.26, 0), true)
	bend, err := NewBend3D(arc, l1, l2)
	if err != nil {
		t.Fatalf("NewBend3D: %v", err)
	}
	if n := len(bend.Residuals()); n != 10 {
		t.Errorf("split-join residual rows = %d, want 10 (2×3 G0 + 2 tangency + circle + radius)", n)
	}
}
