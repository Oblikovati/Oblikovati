// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/kernel/brep"
	gmath "oblikovati.org/math"
)

// TestAddHoleNotes: hole notes annotate a cylinder's rim with a leadered Ø callout that re-resolves
// the diameter when the model changes.
func TestAddHoleNotes(t *testing.T) {
	c := drawingWithCylinder(t, 2) // 2 cm radius → Ø40
	topBase(t, c.Sheets().Active().Views())
	hn, err := c.Sheets().Active().Annotations().AddHoleNotes("HN", "TOP", types.HoleNotePerHole, "")
	if err != nil {
		t.Fatalf("AddHoleNotes: %v", err)
	}
	if hn.Kind() != types.HoleNoteAnnotation || hn.RowCount() != 1 || hn.CurveCount() == 0 {
		t.Fatalf("hole notes = (%v, %d notes, %d curves), want a holeNote with 1 callout + leader", hn.Kind(), hn.RowCount(), hn.CurveCount())
	}
	if len(hn.Labels()) != 1 || hn.Labels()[0].Text != "Ø40.00" {
		t.Fatalf("hole note label = %v, want Ø40.00", hn.Labels())
	}

	wider, err := brep.SolidCylinder(gmath.P3(0, 0, 0), gmath.V3(0, 0, 1), 3, 5) // → Ø60
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	c.SetBodyResolver(fakeBodyResolver{body: wider})
	c.RecomputeViews()
	if hn.Labels()[0].Text != "Ø60.00" {
		t.Errorf("after the cylinder widened, hole note = %q, want Ø60.00 (re-resolved)", hn.Labels()[0].Text)
	}
}

// TestHoleNotesNeedHoles: a box has no holes, so hole notes error.
func TestHoleNotesNeedHoles(t *testing.T) {
	c := drawingWithBox(t)
	frontBase(t, c.Sheets().Active().Views())
	if _, err := c.Sheets().Active().Annotations().AddHoleNotes("HN", "FRONT", types.HoleNotePerHole, ""); err == nil {
		t.Error("AddHoleNotes on a box (no holes) = ok, want error")
	}
}

// TestFormatHoleNote: the format template substitutes {d} (diameter) and {n} (count); an empty
// template uses the default ("Ø<d>", or "<n>x Ø<d>" when count > 1).
func TestFormatHoleNote(t *testing.T) {
	cases := []struct {
		format     string
		diameterMM float64
		count      int
		want       string
	}{
		{"", 8, 1, "Ø8.00"},
		{"", 8, 3, "3x Ø8.00"},
		{"Ø{d} THRU", 8, 1, "Ø8.00 THRU"},
		{"TAP M8 x{n}", 8, 4, "TAP M8 x4"},
	}
	for _, c := range cases {
		if got := formatHoleNote(c.format, c.diameterMM, c.count); got != c.want {
			t.Errorf("formatHoleNote(%q, %g, %d) = %q, want %q", c.format, c.diameterMM, c.count, got, c.want)
		}
	}
}

// TestHoleNotesFormatOverride: a custom format flows through to a base view's hole callout, with the
// diameter still computed from the model.
func TestHoleNotesFormatOverride(t *testing.T) {
	c := drawingWithCylinder(t, 2) // Ø40
	topBase(t, c.Sheets().Active().Views())
	hn, err := c.Sheets().Active().Annotations().AddHoleNotes("HN", "TOP", types.HoleNotePerHole, "Ø{d} THRU")
	if err != nil {
		t.Fatalf("AddHoleNotes: %v", err)
	}
	if len(hn.Labels()) != 1 || hn.Labels()[0].Text != "Ø40.00 THRU" {
		t.Fatalf("formatted hole note = %v, want Ø40.00 THRU", hn.Labels())
	}
}

// TestRenderCombinedHoleNotes: combined grouping collapses equal-diameter holes into one
// "<n>x Ø<d>" callout per size, in encounter order, leaving distinct sizes as their own callouts.
func TestRenderCombinedHoleNotes(t *testing.T) {
	holes := []projectedHoleNote{
		{sx: 10, sy: 10, radiusMM: 4}, // Ø8.00
		{sx: 40, sy: 10, radiusMM: 6}, // Ø12.00
		{sx: 70, sy: 10, radiusMM: 4}, // Ø8.00 (groups with the first)
	}
	a := &DrawingAnnotation{kind: types.HoleNoteAnnotation, holeQuantity: types.HoleNoteCombined}
	renderCombinedHoleNotes(a, holes, "")
	if a.rowCount != 2 || len(a.labels) != 2 {
		t.Fatalf("combined notes = %d callouts, want 2 (Ø8 group + Ø12)", a.rowCount)
	}
	if a.labels[0].Text != "2x Ø8.00" || a.labels[1].Text != "Ø12.00" {
		t.Errorf("combined labels = [%q, %q], want [2x Ø8.00, Ø12.00]", a.labels[0].Text, a.labels[1].Text)
	}
}
