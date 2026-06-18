//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/compdef"
)

// twoDocSession returns a session with two open parts (each with a recorded edit) and their ids.
func twoDocSession(t *testing.T) (*app.Session, []historyDoc) {
	t.Helper()
	s := app.NewSession()
	for _, name := range []string{"alpha.obk", "beta.obk"} {
		if _, err := compdef.AddPart(s.Workspace(), name, true); err != nil {
			t.Fatalf("AddPart %s: %v", name, err)
		}
		s.EnsureActiveEditBaseline()
		if err := s.AddNumericUserParameter("p", "1 cm"); err != nil {
			t.Fatalf("edit %s: %v", name, err)
		}
	}
	return s, historyDocuments(s)
}

// TestHistoryDocumentsListsOpenModels covers historyDocuments: every open part appears.
func TestHistoryDocumentsListsOpenModels(t *testing.T) {
	_, docs := twoDocSession(t)
	if len(docs) != 2 {
		t.Fatalf("historyDocuments = %d, want 2", len(docs))
	}
}

// TestEnsureAndSelectedHistoryDocs covers ensureHistorySelection (default-select branch and the
// already-selected early return) and selectedHistoryDocs filtering.
func TestEnsureAndSelectedHistoryDocs(t *testing.T) {
	_, docs := twoDocSession(t)
	for k := range historyBrowserSelection { // start from a clean selection
		delete(historyBrowserSelection, k)
	}

	// Nothing selected ⇒ ensure picks the first document.
	ensureHistorySelection(docs)
	if got := selectedHistoryDocs(docs); len(got) != 1 || got[0].id != docs[0].id {
		t.Fatalf("default selection = %v, want exactly the first document", got)
	}

	// Already selected ⇒ ensure is a no-op (early return), selection unchanged.
	ensureHistorySelection(docs)
	if got := selectedHistoryDocs(docs); len(got) != 1 {
		t.Fatalf("ensure changed an existing selection: %v", got)
	}

	// Selecting both shows both columns.
	historyBrowserSelection[docs[1].id] = true
	if got := selectedHistoryDocs(docs); len(got) != 2 {
		t.Fatalf("both selected = %v, want 2", got)
	}
}
