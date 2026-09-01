// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/sketch"
)

// extrudedPartSession builds a session whose part holds one committed extrude solid, returning the
// session and definition — a real feature to (re)evaluate when a parameter is edited.
func extrudedPartSession(t *testing.T) (*Session, profileFeatureCount) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	rect := NewRectangleTool()
	s.StartTool(rect)
	s.Click(40, 40)
	s.Click(160, 160) // auto-commits the rectangle
	s.ExitSketch()
	s.SetPicker(stubPicker{sel: ProfileHandle{Sketch: sk, ProfileIndex: 0}})
	ext := NewExtrudeTool()
	s.StartTool(ext)
	s.Click(100, 100)
	ext.SetDistance(8)
	if err := s.OK(); err != nil {
		t.Fatalf("extrude OK: %v", err)
	}
	if def.Features().Count() == 0 {
		t.Fatal("no feature committed by the extrude")
	}
	return s, def.Features().Item(0).RecomputeCount
}

// profileFeatureCount reads the committed feature's recompute count on demand.
type profileFeatureCount func() int

// TestParameterEditViaAppPathRebuildsFeature is the #1413 regression: editing a parameter through the
// app dialog verb (editParameters) must invalidate the feature program and rebuild, not return cached,
// stale geometry. Before the fix, editParameters called Recompute without MarkAllDirty, so the engine
// found nothing dirty (feature inputs are read live, not tracked as deps) and handed back the cached
// bodies — a silent stale-geometry bug the router path did not have. The feature's recompute count must
// advance across the edit.
func TestParameterEditViaAppPathRebuildsFeature(t *testing.T) {
	t.Parallel()
	s, recomputeCount := extrudedPartSession(t)
	if err := s.AddNumericUserParameter("len", "10 mm"); err != nil {
		t.Fatalf("add parameter: %v", err)
	}
	id := mustParamID(t, partParams(t, s), "len")

	before := recomputeCount() // after the add, isolate the edit's effect
	if err := s.SetParameterEquation(id, "25 mm"); err != nil {
		t.Fatalf("edit parameter via the app path: %v", err)
	}
	if after := recomputeCount(); after <= before {
		t.Errorf("feature recompute count %d→%d: the app parameter edit did not rebuild (stale geometry, #1413)", before, after)
	}
}

// TestParameterEditPathsInvalidateIdentically is the #1413 parity check: the app dialog verb and the
// import verb (the two app-side parameter-edit seams) both route through RecomputeAfterChange,
// so each forces a rebuild. Editing the same part the same way through either must re-evaluate the
// feature — the single shared seam, no divergence.
func TestParameterEditPathsInvalidateIdentically(t *testing.T) {
	t.Parallel()
	for _, edit := range []struct {
		name string
		run  func(t *testing.T, s *Session)
	}{
		{"dialog equation verb", func(t *testing.T, s *Session) {
			id := mustParamID(t, partParams(t, s), "len")
			if err := s.SetParameterEquation(id, "25 mm"); err != nil {
				t.Fatalf("SetParameterEquation: %v", err)
			}
		}},
		{"xml import verb", func(t *testing.T, s *Session) {
			if _, _, err := s.ImportParameters(`<parameters><parameter name="len" expression="25 mm"/></parameters>`); err != nil {
				t.Fatalf("ImportParameters: %v", err)
			}
		}},
	} {
		t.Run(edit.name, func(t *testing.T) {
			s, recomputeCount := extrudedPartSession(t)
			if err := s.AddNumericUserParameter("len", "10 mm"); err != nil {
				t.Fatalf("add parameter: %v", err)
			}
			before := recomputeCount()
			edit.run(t, s)
			if after := recomputeCount(); after <= before {
				t.Errorf("%s: recompute count %d→%d, want a rebuild (every parameter-edit seam must invalidate)", edit.name, before, after)
			}
		})
	}
}
