// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/api/wire"
)

// TestSketchSettingsRoundTrip reads the active document's sketch settings (defaults), writes a
// changed value back, and checks the reply reflects it (#147).
func TestSketchSettingsRoundTrip(t *testing.T) {
	r, s := seededSession(t)

	var got wire.SketchSettingsResult
	call(t, r, s, wire.MethodDocumentGetSketchSettings, `{"document":0}`, &got)
	if !got.Settings.InferConstraints || !got.Settings.AutoApplyConstraints {
		t.Fatalf("default settings = %+v, want inference + auto-apply on", got.Settings)
	}

	want := types.SketchSettings{InferConstraints: true, AutoApplyConstraints: false, ConstraintPriority: types.PriorityParallelPerpendicular}
	var back wire.SketchSettingsResult
	call(t, r, s, wire.MethodDocumentSetSketchSettings, mustJSON(t, wire.SetSketchSettingsArgs{Document: 0, Settings: want}), &back)
	if back.Settings != want {
		t.Errorf("set settings = %+v, want %+v", back.Settings, want)
	}

	// A fresh read returns the stored value — it persisted on the document.
	var reread wire.SketchSettingsResult
	call(t, r, s, wire.MethodDocumentGetSketchSettings, `{"document":0}`, &reread)
	if reread.Settings != want {
		t.Errorf("re-read settings = %+v, want %+v (not stored on the document)", reread.Settings, want)
	}
}
