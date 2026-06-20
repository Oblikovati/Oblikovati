// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/sketch"
)

// TestSketchInferenceOptionsReadActiveDocument checks the per-document wiring (#147): the sketch
// tools' inference options come from the active document's settings, and editing them persists on
// that document (so SetDocumentSketchSettings sees the same value).
func TestSketchInferenceOptionsReadActiveDocument(t *testing.T) {
	s, _ := emptyPartSession(t)

	// A fresh document yields the defaults (inference + auto-apply on).
	if o := s.SketchInferenceOptions(); !o.InferEnabled || !o.ConstrainEnabled {
		t.Fatalf("default inference options = %+v, want inference + constrain on", o)
	}

	// Editing through the sketch-tools setter writes to the active document.
	s.SetSketchInferenceOptions(sketch.InferenceOptions{InferEnabled: true, ConstrainEnabled: false, Priority: types.PriorityParallelPerpendicular})
	if o := s.SketchInferenceOptions(); o.ConstrainEnabled || o.Priority != types.PriorityParallelPerpendicular {
		t.Errorf("after edit, options = %+v, want constrain off / parallelPerpendicular", o)
	}

	// The document-addressed API reflects the same state (0 ⇒ active document).
	got, err := s.DocumentSketchSettings(0)
	if err != nil {
		t.Fatalf("DocumentSketchSettings: %v", err)
	}
	if got.AutoApplyConstraints || got.ConstraintPriority != types.PriorityParallelPerpendicular {
		t.Errorf("document settings = %+v, want auto-apply off / parallelPerpendicular", got)
	}
}

// TestSetDocumentSketchSettingsNoActiveDocument checks the addressed-document error path.
func TestSetDocumentSketchSettingsNoActiveDocument(t *testing.T) {
	s := NewSession() // no document open
	if _, err := s.DocumentSketchSettings(0); err == nil {
		t.Error("DocumentSketchSettings with no active document should error")
	}
}
