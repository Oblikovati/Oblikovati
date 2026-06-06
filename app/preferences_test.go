// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/model/param"
)

func TestDefaultGridSettings(t *testing.T) {
	g := NewGridSettings()
	if !g.Visible || g.MajorEvery <= 0 {
		t.Errorf("default grid = %+v, want visible with a positive major interval", g)
	}
	// The default spacing is 1 cm = 10 mm.
	if g.SpacingModel() != 1.0 {
		t.Errorf("default spacing = %v cm, want 1", g.SpacingModel())
	}
	if v := g.SpacingIn(param.DefaultUnitsOfMeasure()); v != 10 {
		t.Errorf("default spacing in mm = %v, want 10", v)
	}
}

func TestGridSpacingRespectsDocumentUnits(t *testing.T) {
	g := NewGridSettings()
	mm := param.DefaultUnitsOfMeasure() // length unit "mm"
	if err := g.SetSpacingIn(5, mm); err != nil {
		t.Fatalf("SetSpacingIn: %v", err)
	}
	// 5 mm is 0.5 cm in model/database units.
	if g.SpacingModel() != 0.5 {
		t.Errorf("5 mm spacing = %v cm model, want 0.5", g.SpacingModel())
	}
	// Switching the document to inches re-presents the same model spacing.
	in := param.DefaultUnitsOfMeasure()
	if err := in.SetPreferred(param.Length, "in"); err != nil {
		t.Fatalf("SetPreferred in: %v", err)
	}
	if err := g.SetSpacingIn(1, in); err != nil { // 1 inch
		t.Fatalf("SetSpacingIn inch: %v", err)
	}
	if got := g.SpacingModel(); got < 2.53 || got > 2.55 { // 1 in = 2.54 cm
		t.Errorf("1 inch spacing = %v cm model, want ~2.54", got)
	}
}

func TestGridSpacingRejectsNonPositive(t *testing.T) {
	g := NewGridSettings()
	if err := g.SetSpacingIn(0, param.DefaultUnitsOfMeasure()); err == nil {
		t.Error("zero spacing should error")
	}
}

func TestSessionGridAndDocumentUnits(t *testing.T) {
	s, _ := emptyPartSession(t)
	if g := s.Grid(); g == nil || s.Grid() != g {
		t.Error("Grid() should return a stable settings object")
	}
	if name := s.DocumentUnits().PreferredName(param.Length); name != "mm" {
		t.Errorf("document length unit = %q, want mm (metric default)", name)
	}
}

func TestDocumentUnitsWithoutPartIsMetric(t *testing.T) {
	s := NewSession() // no document
	if name := s.DocumentUnits().PreferredName(param.Length); name != "mm" {
		t.Errorf("no-part document units = %q, want the mm default", name)
	}
}
