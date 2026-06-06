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
