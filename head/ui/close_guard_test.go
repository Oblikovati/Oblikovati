// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati/app"
	"oblikovati/model/compdef"
)

// TestDirtyDocuments checks the graceful-close core lists exactly the documents with
// unsaved changes — the set the close prompt warns about.
func TestDirtyDocuments(t *testing.T) {
	s := app.NewSession()
	a, err := compdef.AddPart(s.Workspace(), "a.obk", true)
	if err != nil {
		t.Fatalf("AddPart a: %v", err)
	}
	b, err := compdef.AddPart(s.Workspace(), "b.obk", true)
	if err != nil {
		t.Fatalf("AddPart b: %v", err)
	}

	// Freshly created documents are dirty until saved.
	if got := dirtyDocuments(s); len(got) != 2 {
		t.Fatalf("dirtyDocuments = %d, want 2 (both new)", len(got))
	}

	a.ClearDirty()
	got := dirtyDocuments(s)
	if len(got) != 1 || got[0] != b {
		t.Errorf("after clearing a, dirtyDocuments = %v, want [b]", got)
	}

	b.ClearDirty()
	if got := dirtyDocuments(s); len(got) != 0 {
		t.Errorf("after clearing both, dirtyDocuments = %d, want 0", len(got))
	}
}

func TestDocumentCloseGuardRequestsPromptOnlyForDirtyDocument(t *testing.T) {
	s := app.NewSession()
	d, err := compdef.AddPart(s.Workspace(), "dirty.obk", true)
	if err != nil {
		t.Fatalf("AddPart dirty: %v", err)
	}
	var guard documentCloseGuard
	if guard.request(d) || guard.pending != d {
		t.Fatalf("request dirty = close-now with pending %+v, want prompt", guard.pending)
	}
	d.ClearDirty()
	guard.cancel()
	if !guard.request(d) || guard.pending != nil {
		t.Fatalf("request clean pending=%+v, want close-now and no prompt", guard.pending)
	}
}

func TestDocumentCloseGuardDiscardClosesWithoutSaving(t *testing.T) {
	s := app.NewSession()
	d, err := compdef.AddPart(s.Workspace(), "dirty.obk", true)
	if err != nil {
		t.Fatalf("AddPart dirty: %v", err)
	}
	guard := documentCloseGuard{pending: d}
	guard.discard(s)
	if guard.pending != nil {
		t.Fatalf("discard left pending %+v", guard.pending)
	}
	if got := len(s.Workspace().Documents()); got != 0 {
		t.Fatalf("documents after discard = %d, want 0", got)
	}
}

func TestDocumentCloseGuardCloseIfCleanWaitsUntilSaved(t *testing.T) {
	s := app.NewSession()
	d, err := compdef.AddPart(s.Workspace(), "dirty.obk", true)
	if err != nil {
		t.Fatalf("AddPart dirty: %v", err)
	}
	guard := documentCloseGuard{pending: d}
	if guard.closeIfClean(s) {
		t.Fatal("closeIfClean closed dirty document")
	}
	d.ClearDirty()
	if !guard.closeIfClean(s) || guard.pending != nil {
		t.Fatalf("closeIfClean pending=%+v, want closed clean document", guard.pending)
	}
}

func TestDocumentNeedsSaveAsForUnsavedPartName(t *testing.T) {
	s := app.NewSession()
	unsaved, err := compdef.AddPart(s.Workspace(), "Part1", true)
	if err != nil {
		t.Fatalf("AddPart unsaved: %v", err)
	}
	if !documentNeedsSaveAs(unsaved) {
		t.Fatal("documentNeedsSaveAs(Part1) = false, want true")
	}
	savedName, err := compdef.AddPart(s.Workspace(), "saved.obk", true)
	if err != nil {
		t.Fatalf("AddPart saved-name: %v", err)
	}
	if documentNeedsSaveAs(savedName) {
		t.Fatal("documentNeedsSaveAs(saved.obk) = true, want false")
	}
}
