// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// TestOpenWarnsOnInterestsFromAbsentClients: a document carrying interests
// from a client not present this session surfaces a status-bar warning on
// open — never an error (M03-F10, #611).
func TestOpenWarnsOnInterestsFromAbsentClients(t *testing.T) {
	t.Parallel()
	s := NewSession()
	d, err := s.Workspace().Add(doc.Part, "augmented.obk", true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := d.Interests().Add(types.DocumentInterestRecord{
		ClientID: "com.x.toolpaths", Name: "toolpath-recipes", InterestType: types.Interested,
	}); err != nil {
		t.Fatalf("Add interest: %v", err)
	}

	// The open hook fires on the workspace event; emit it as ws.Open would.
	event.Emit(s.Workspace().Events(), event.After, doc.DocumentOpened{FullDocumentName: "augmented.obk"})
	if !strings.Contains(s.Notice(), "com.x.toolpaths") {
		t.Fatalf("notice = %q, want the absent client named", s.Notice())
	}

	// With the client present (registered as a client app) there is no warning.
	if _, err := s.ClientApps().Register("com.x.toolpaths"); err != nil {
		t.Fatalf("Register client app: %v", err)
	}
	s.SetNotice("")
	event.Emit(s.Workspace().Events(), event.After, doc.DocumentOpened{FullDocumentName: "augmented.obk"})
	if s.Notice() != "" {
		t.Errorf("notice = %q, want none when the client is present", s.Notice())
	}
}
