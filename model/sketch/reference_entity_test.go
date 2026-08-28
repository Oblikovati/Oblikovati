// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestReferenceEntityIsGrounded: a reference entity (its points in refPts, its scalar DOF skipped)
// adds no free degrees of freedom — the solver holds it fixed while free geometry moves to meet it
// (ADR-0055 phase 3). A plain circle contributes 3 DOF (centre x,y + radius); a reference one, 0.
func TestReferenceEntityIsGrounded(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	s.circles.AddByCenterRadius(math.P2(0, 0), 5)
	if dof := s.DegreesOfFreedom(); dof != 3 {
		t.Fatalf("a plain circle contributes DOF = %d, want 3", dof)
	}
	ref := s.circles.Add(s.newRefPoint(math.P2(10, 0)), 2)
	ref.SetReference(true)
	if dof := s.DegreesOfFreedom(); dof != 3 {
		t.Fatalf("a reference circle changed free DOF to %d, want still 3 (grounded)", dof)
	}
}
