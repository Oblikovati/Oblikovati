// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
	"oblikovati.org/model/compdef"
	"oblikovati.org/model/feature"
	"oblikovati.org/model/sketch"
)

// M40 S6 (#1641): body names, color styles and sketch/display settings ride the SAME undo snapshot as
// the recipe, so a metadata mutation is undoable and undo can never leave a recipe/metadata hybrid.

// addSecondSquareBody extrudes a second, separate box body into def — a recorded-able geometry change.
func addSecondSquareBody(def *compdef.PartComponentDefinition) {
	sk := def.Sketches().Add(sketch.XYPlane())
	c0, c1 := sk.Points().Add(math.P2(10, 0)), sk.Points().Add(math.P2(14, 0))
	c2, c3 := sk.Points().Add(math.P2(14, 3)), sk.Points().Add(math.P2(10, 3))
	sk.Lines().Add(c0, c1)
	sk.Lines().Add(c1, c2)
	sk.Lines().Add(c2, c3)
	sk.Lines().Add(c3, c0)
	feature.NewExtrudeFeatures(def.Features()).AddByDistanceExtent(sk, 0, ops.NewBody, func() float64 { return 5 })
	def.Recompute()
}

// TestMetadataMixedHistoryNoHybrid pins the divergence bug: a geometry step recorded AFTER a metadata
// step must undo without leaving the metadata behind (the pre-#1641 hybrid). Undoing both the geometry
// and the color returns the document to its exact starting state.
func TestMetadataMixedHistoryNoHybrid(t *testing.T) {
	s, def := boxBodySession(t)
	key := bodyKeyOf(t, def)
	s.EnsureActiveEditBaseline() // baseline: one box body, no color

	if err := s.AssignColorStyleToBody(key, "Brass"); err != nil { // metadata step
		t.Fatalf("assign: %v", err)
	}
	addSecondSquareBody(def) // geometry step
	s.RecordActiveEdit("Geometry")
	if def.SurfaceBodies().Count() != 2 {
		t.Fatalf("geometry step should have produced 2 bodies, got %d", def.SurfaceBodies().Count())
	}

	if err := s.Undo(); err != nil { // undo geometry
		t.Fatalf("undo geometry: %v", err)
	}
	if err := s.Undo(); err != nil { // undo color
		t.Fatalf("undo color: %v", err)
	}
	if def.SurfaceBodies().Count() != 1 {
		t.Errorf("geometry not reverted after undo: %d bodies, want 1", def.SurfaceBodies().Count())
	}
	if _, ok := s.BodyColorStyle(key); ok {
		t.Error("color survived the undo — a recipe/metadata hybrid (#1641)")
	}
}

// TestSketchSettingsUndoable checks a per-document sketch-settings change is a real undo step (S6).
func TestSketchSettingsUndoable(t *testing.T) {
	s, _ := boxBodySession(t)
	base, _ := s.DocumentSketchSettings(0)
	changed := base
	changed.InferConstraints = !base.InferConstraints
	if _, err := s.SetDocumentSketchSettings(0, changed); err != nil {
		t.Fatalf("set sketch settings: %v", err)
	}
	if got, _ := s.DocumentSketchSettings(0); got.InferConstraints != changed.InferConstraints {
		t.Fatalf("sketch settings not applied")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got, _ := s.DocumentSketchSettings(0); got.InferConstraints != base.InferConstraints {
		t.Errorf("sketch-settings change not undone: InferConstraints=%v, want %v (#1641)", got.InferConstraints, base.InferConstraints)
	}
}

// TestDisplaySettingsUndoable checks a per-document display-settings change is a real undo step (S6).
func TestDisplaySettingsUndoable(t *testing.T) {
	s, _ := boxBodySession(t)
	base := s.DisplaySettings(0)
	changed := base
	changed.ShowObjectShadows = !base.ShowObjectShadows
	s.SetDisplaySettings(0, changed)
	if got := s.DisplaySettings(0); got.ShowObjectShadows != changed.ShowObjectShadows {
		t.Fatalf("display settings not applied")
	}
	if err := s.Undo(); err != nil {
		t.Fatalf("undo: %v", err)
	}
	if got := s.DisplaySettings(0); got.ShowObjectShadows != base.ShowObjectShadows {
		t.Errorf("display-settings change not undone: ShowObjectShadows=%v, want %v (#1641)", got.ShowObjectShadows, base.ShowObjectShadows)
	}
}
