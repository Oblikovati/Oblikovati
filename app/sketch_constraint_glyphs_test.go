// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Show Constraints: a geometric constraint used to be invisible and unremovable from the viewport
// — the solver honoured it, nothing listed it, nothing could select one — so a wrong auto-inferred
// relation could only be undone, never repaired. These drive the real click path.

// constrainedSketchSession returns a session editing a sketch holding two lines that share a
// corner, joined by a coincidence and a perpendicular, with markers shown.
func constrainedSketchSession(t *testing.T) (*Session, *sketch.Sketch, sketch.Constraint) {
	t.Helper()
	s, _ := emptyPartSession(t)
	sk, err := s.CreateSketch(sketch.XYPlane())
	if err != nil {
		t.Fatalf("CreateSketch: %v", err)
	}
	a := sk.Points().Add(math.P2(0, 0))
	b := sk.Points().Add(math.P2(4, 0))
	perp := sk.GeometricConstraints().AddPerpendicular(
		sk.Lines().Add(a, b),
		sk.Lines().Add(sk.Points().Add(math.P2(0, 0)), sk.Points().Add(math.P2(0, 4))),
	)
	s.SetShowSketchConstraints(true)
	return s, sk, perp
}

// TestConstraintMarkersOnlyDrawWhenShown: the toggle is the whole feature — hidden means nothing
// is drawn and, just as importantly, nothing is clickable.
func TestConstraintMarkersOnlyDrawWhenShown(t *testing.T) {
	s, _, _ := constrainedSketchSession(t)

	if len(s.SketchConstraintGlyphs()) == 0 {
		t.Fatal("markers are shown but none were produced")
	}
	s.SetShowSketchConstraints(false)
	if got := len(s.SketchConstraintGlyphs()); got != 0 {
		t.Errorf("%d markers produced while hidden", got)
	}
}

// TestClickingAMarkerSelectsItsConstraint is the pick regression: a marker sits on the geometry it
// relates, so it has to be tested before the curve underneath or it can never be grabbed.
func TestClickingAMarkerSelectsItsConstraint(t *testing.T) {
	s, _, perp := constrainedSketchSession(t)
	glyphs := s.SketchConstraintGlyphs()
	if len(glyphs) != 1 {
		t.Fatalf("got %d markers, want 1", len(glyphs))
	}
	px, py, ok := sketchToScreen(s, glyphs[0].At)
	if !ok {
		t.Fatal("the marker did not project to the viewport")
	}

	s.Click(px, py)

	got := s.SelectedSketchConstraints()
	if len(got) != 1 || got[0] != perp {
		t.Fatalf("clicking the marker selected %v, want the perpendicular constraint", got)
	}
}

// TestDeleteRemovesTheSelectedConstraint is the point of selecting one: Delete frees the degrees
// of freedom the relation held.
func TestDeleteRemovesTheSelectedConstraint(t *testing.T) {
	s, sk, perp := constrainedSketchSession(t)
	before := sk.GeometricConstraints().Count()
	s.applyPickToSelection(SketchConstraintHandle{Constraint: perp}, 0)

	if err := s.DeleteSelectedSketchEntities(); err != nil {
		t.Fatalf("delete: %v", err)
	}

	if got := sk.GeometricConstraints().Count(); got != before-1 {
		t.Errorf("constraint count %d, want %d — Delete did not remove the selected relation", got, before-1)
	}
	if len(s.SelectedSketchConstraints()) != 0 {
		t.Error("the deleted constraint is still selected")
	}
}

// TestHidingMarkersDropsTheSelection: a constraint must not stay selected while invisible, or a
// later Delete would remove a relation the user can no longer see.
func TestHidingMarkersDropsTheSelection(t *testing.T) {
	s, _, perp := constrainedSketchSession(t)
	s.applyPickToSelection(SketchConstraintHandle{Constraint: perp}, 0)

	s.SetShowSketchConstraints(false)

	if got := s.SelectedSketchConstraints(); len(got) != 0 {
		t.Errorf("%d constraints still selected after hiding the markers", len(got))
	}
}

// TestCoLocatedMarkersFanOut: several relations on the same point — the four coincidences at a
// rectangle corner — must each stay separately clickable rather than stacking into one target.
func TestCoLocatedMarkersFanOut(t *testing.T) {
	s, sk, _ := constrainedSketchSession(t)
	a := sk.Points().Add(math.P2(2, 2))
	b := sk.Points().Add(math.P2(2, 2))
	c := sk.Points().Add(math.P2(2, 2))
	sk.GeometricConstraints().AddCoincident(a, b)
	sk.GeometricConstraints().AddCoincident(a, c)

	var at []math.Point2
	for _, g := range s.SketchConstraintGlyphs() {
		at = append(at, g.At)
	}
	for i := range at {
		for j := i + 1; j < len(at); j++ {
			if at[i] == at[j] {
				t.Fatalf("markers %d and %d share the position %v; one of them can never be clicked", i, j, at[i])
			}
		}
	}
}

// TestSelectedMarkerReportsSelected: the overlay colours a selected marker, so the view has to say
// which one it is.
func TestSelectedMarkerReportsSelected(t *testing.T) {
	s, _, perp := constrainedSketchSession(t)
	s.applyPickToSelection(SketchConstraintHandle{Constraint: perp}, 0)

	glyphs := s.SketchConstraintGlyphs()
	if len(glyphs) != 1 || !glyphs[0].Selected {
		t.Errorf("marker Selected = %v, want true for the selected constraint", glyphs)
	}
}
