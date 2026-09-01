// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"encoding/json"
	"testing"

	"oblikovati.org/addin/modelaccess"
	"oblikovati.org/addin/opregistry"
	"oblikovati.org/api/wire"
	"oblikovati.org/app"
)

// markPartDirty drives the active part into a needs-update state (a parameter edit would do this
// in the model; the test marks every feature dirty directly).
func markPartDirty(t *testing.T, s *app.Session) {
	t.Helper()
	part, err := modelaccess.ActivePart(s)
	if err != nil {
		t.Fatalf("active part: %v", err)
	}
	part.Features().MarkAllDirty()
}

// TestRequiresUpdateReflectsDirtyAndClears: documents.requiresUpdate is false on a freshly
// recomputed part, true once a feature is dirtied, and false again after documents.update (#139).
func TestRequiresUpdateReflectsDirtyAndClears(t *testing.T) {
	t.Parallel()
	r, s := boxBodySession(t) // extrude already recomputed → clean

	var st wire.RequiresUpdateResult
	call(t, r, s, "documents.requiresUpdate", `{}`, &st)
	if st.RequiresUpdate {
		t.Errorf("freshly built part requiresUpdate=true, want false")
	}

	markPartDirty(t, s)
	call(t, r, s, "documents.requiresUpdate", `{}`, &st)
	if !st.RequiresUpdate {
		t.Errorf("dirtied part requiresUpdate=false, want true")
	}

	var up wire.UpdateDocumentResult
	call(t, r, s, "documents.update", `{}`, &up)
	if up.RequiresUpdate || len(up.Errors) != 0 {
		t.Errorf("after update = %+v, want requiresUpdate=false and no errors", up)
	}
	call(t, r, s, "documents.requiresUpdate", `{}`, &st)
	if st.RequiresUpdate {
		t.Errorf("after update requiresUpdate=true, want false")
	}
}

// TestDocumentsRebuildRecomputes: documents.rebuild recomputes the whole program and leaves the
// part up to date with no errors on a healthy model.
func TestDocumentsRebuildRecomputes(t *testing.T) {
	t.Parallel()
	r, s := boxBodySession(t)
	var up wire.UpdateDocumentResult
	call(t, r, s, "documents.rebuild", `{}`, &up)
	if up.RequiresUpdate || len(up.Errors) != 0 {
		t.Errorf("rebuild of a healthy part = %+v, want requiresUpdate=false and no errors", up)
	}
	// The body still exists after a full rebuild.
	if faces := partBodyFaces(t, s); faces != 6 {
		t.Errorf("after rebuild: %d faces, want 6 (the box survived)", faces)
	}
}

// TestDocumentsUpdateReportsSickFeatures: a recompute that leaves a feature sick fails a strict
// update but succeeds — reporting the sick feature — with acceptErrorsAndContinue (Update2, #139).
func TestDocumentsUpdateReportsSickFeatures(t *testing.T) {
	t.Parallel()
	r, s := boxBodySession(t)
	// A fillet on a non-existent edge recomputes sick (a lost reference).
	args, _ := json.Marshal(map[string]any{
		"kind": "fillet",
		"args": map[string]any{"edgeRefs": []string{"bogus-edge-key"}, "radius": "2 mm"},
	})
	call(t, r, s, "features.add", string(args), &struct {
		Bodies int `json:"bodies"`
	}{})

	if _, err := r.Handle(s, "documents.update", []byte(`{}`)); err == nil {
		t.Error("strict documents.update should fail when a feature is sick")
	}

	var up wire.UpdateDocumentResult
	call(t, r, s, "documents.update", `{"acceptErrorsAndContinue":true}`, &up)
	if len(up.Errors) != 1 || up.Errors[0].Kind != "fillet" {
		t.Errorf("update errors = %+v, want one fillet", up.Errors)
	}
}

// TestDocumentsUpdateNoActivePart: the methods error without an active part.
func TestDocumentsUpdateNoActivePart(t *testing.T) {
	t.Parallel()
	r := New(opregistry.Default())
	s := app.NewSession()
	for _, m := range []string{"documents.update", "documents.rebuild", "documents.requiresUpdate"} {
		if _, err := r.Handle(s, m, []byte(`{}`)); err == nil {
			t.Errorf("%s with no active part should fail", m)
		}
	}
}
