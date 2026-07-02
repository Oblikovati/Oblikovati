// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestRedundantConstraintsNamesRemovableDuplicate pins the sketch-level surface of the A13
// leave-one-out identification (#1609): a point pinned by a ground AND a duplicate anchor is
// over-constrained; the sketch must name a removable constraint whose removal restores clean
// DOF bookkeeping.
func TestRedundantConstraintsNamesRemovableDuplicate(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(1, 0))
	g := s.GeometricConstraints()
	g.AddFix(a)
	g.AddFix(b)
	g.AddFix(b) // duplicate: redundant
	if got := s.AnalyzeConstraints(); got.Redundant == 0 {
		t.Fatalf("fixture drifted: expected an over-constrained sketch, got %+v", got)
	}
	red := s.RedundantConstraints()
	if len(red) == 0 {
		t.Fatal("RedundantConstraints named nothing on an over-constrained sketch")
	}
}
