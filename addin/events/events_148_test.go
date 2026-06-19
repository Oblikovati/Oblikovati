// SPDX-License-Identifier: GPL-2.0-only

package events

import (
	"testing"

	"oblikovati.org/app"
	"oblikovati.org/model/doc"
)

// TestForwardsDocumentClosedAndModelChanged (#148): closing a document and committing a model
// change both reach the add-in sink as document.closed and model.changed.
func TestForwardsDocumentClosedAndModelChanged(t *testing.T) {
	s := app.NewSession()
	var rec recorder
	Subscribe(s, rec.sink)

	d, err := s.Workspace().Add(doc.Part, "ev148.obk", true)
	if err != nil {
		t.Fatalf("add document: %v", err)
	}
	if err := s.Workspace().NotifyModelChanged(d, doc.ChangeDefinition{}); err != nil {
		t.Fatalf("notify model changed: %v", err)
	}
	if err := s.Workspace().Close(d, true); err != nil {
		t.Fatalf("close: %v", err)
	}

	got := rec.types()
	if !has(got, "model.changed") {
		t.Errorf("missing model.changed in %v", got)
	}
	if !has(got, "document.closed") {
		t.Errorf("missing document.closed in %v", got)
	}
}
