// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// TestIdentityCollisionSetsNotice proves the session surfaces an open-time identity
// reassignment to the user: when the workspace fires DocumentIdentityReassigned, the watcher
// wired in newSession sets a notice naming the affected document.
func TestIdentityCollisionSetsNotice(t *testing.T) {
	t.Parallel()
	s := NewSession()
	d, err := s.Workspace().Add(doc.Drawing, "box.odd", true)
	if err != nil {
		t.Fatalf("add: %v", err)
	}

	event.Emit(s.Workspace().Events(), event.After, doc.DocumentIdentityReassigned{
		Document:             d,
		PreviousInternalName: "shared-guid",
		NewInternalName:      "fresh-guid",
	})

	notice := s.Notice()
	if !strings.Contains(notice, d.DisplayName()) {
		t.Errorf("notice %q should name the reassigned document %q", notice, d.DisplayName())
	}
	if !strings.Contains(strings.ToLower(notice), "new id") {
		t.Errorf("notice %q should tell the user a new id was assigned", notice)
	}
}
