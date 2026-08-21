// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketchConstraintGlyphs3DFollowsTheToggle is the #1998 (3D) app wiring: a 3D sketch's
// constraint markers appear only when Show Constraints is on, mirror the model glyphs, and the
// Sketch3D.ShowConstraints command flips them.
func TestSketchConstraintGlyphs3DFollowsTheToggle(t *testing.T) {
	s, sk := sketch3DSession(t)
	l1 := sk.AddLine3D(math.P3(0, 0, 0), math.P3(2, 0, 0))
	l2 := sk.AddLine3D(math.P3(0, 1, 0), math.P3(2, 1, 0))
	sk.GeometricConstraints3D().Add(sketch.NewParallel3D(l1, l2))

	if got := s.SketchConstraintGlyphs3D(); len(got) != 0 {
		t.Fatalf("constraints hidden by default, but got %d 3D markers", len(got))
	}

	if err := s.Execute("Sketch3D.ShowConstraints"); err != nil {
		t.Fatalf("Sketch3D.ShowConstraints: %v", err)
	}
	if !s.ShowSketchConstraints() {
		t.Fatal("Sketch3D.ShowConstraints did not turn the markers on")
	}
	got := s.SketchConstraintGlyphs3D()
	if len(got) != 2 {
		t.Fatalf("parallel of two 3D lines drew %d markers, want 2 (one per line)", len(got))
	}
	if got[0].Kind != sketch.ParallelKind {
		t.Errorf("marker kind = %v, want ParallelKind", got[0].Kind)
	}

	// Toggling again hides them, so the 3D command drives the same state as F8 / the 2D toggle.
	if err := s.Execute("Sketch3D.ShowConstraints"); err != nil {
		t.Fatalf("Sketch3D.ShowConstraints (off): %v", err)
	}
	if got := s.SketchConstraintGlyphs3D(); len(got) != 0 {
		t.Errorf("after toggling off, got %d 3D markers, want 0", len(got))
	}
}

// TestSketchConstraintGlyphs3DEmptyWithout3DSketch: the accessor is safe (and empty) when no 3D
// sketch is open, even with the toggle on.
func TestSketchConstraintGlyphs3DEmptyWithout3DSketch(t *testing.T) {
	s := NewSession()
	s.SetShowSketchConstraints(true)
	if got := s.SketchConstraintGlyphs3D(); got != nil {
		t.Errorf("no active 3D sketch, but got %d markers", len(got))
	}
}
