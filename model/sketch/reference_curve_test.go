// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestReferenceCircleGroundedAndConstrainable proves the phase-3 goal: a projected circle built as a
// concrete reference entity is a first-class sketch curve — a free line constrained TANGENT to it
// solves to tangency while the circle itself stays fixed (grounded, driven by its source). This is
// the capability the ProjectedCurve wrapper could not offer (ADR-0055 phase 3).
func TestReferenceCircleGroundedAndConstrainable(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	circ, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 3)
	c2, ok := geom.ProjectCurveToPlane(sketchPlaneToGeom(s.plane), circ)
	if !ok {
		t.Fatal("projection failed")
	}
	ent := s.addReferenceCurve(c2)
	ref, isCircle := ent.(*Circle)
	if !isCircle {
		t.Fatalf("addReferenceCurve = %T, want *Circle", ent)
	}
	if !ref.IsReference() {
		t.Fatal("projected circle is not flagged reference")
	}

	line := s.lines.AddByTwoPoints(math.P2(-5, 5), math.P2(5, 5)) // distance 5 from origin — not tangent
	s.GeometricConstraints().AddTangent(line, ref)
	if res := s.Solve(); !res.Converged {
		t.Fatalf("solve did not converge: %+v", res)
	}
	if float64(ref.Radius) != 3 || !ref.Center.Position().IsEqualTo(math.P2(0, 0), 1e-9) {
		t.Fatalf("reference circle moved to centre %v r %g; it must stay fixed", ref.Center.Position(), float64(ref.Radius))
	}
	d := stdmath.Abs(perpDistanceToLine(line.A.Position(), line.B.Position(), math.P2(0, 0)))
	if stdmath.Abs(d-3) > 1e-6 {
		t.Fatalf("solved line distance from the circle centre = %.4f, want tangent (3)", d)
	}
}
