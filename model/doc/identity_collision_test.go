// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/event"
)

// TestOpenWithCollidingIdentityReassigns proves two files that carry the same persisted
// identity GUID (e.g. one copied outside the application) cannot be open at once sharing it:
// opening the second mints it a fresh GUID, marks it dirty so the new id is written on its
// next save, and fires DocumentIdentityReassigned so the host can notify the user.
func TestOpenWithCollidingIdentityReassigns(t *testing.T) {
	const sharedGUID = "11111111-1111-4111-8111-111111111111"
	store := newFakeStore()
	store.saved["original.opd"] = storedDoc{docType: Part, displayName: "original", internalName: sharedGUID}
	store.saved["clone.opd"] = storedDoc{docType: Part, displayName: "clone", internalName: sharedGUID}
	ws := NewWorkspace(store)

	var reassigned *DocumentIdentityReassigned
	event.Subscribe(ws.Events(), event.After, func(_ event.Context, e DocumentIdentityReassigned) event.Outcome {
		reassigned = &e
		return event.Continue()
	})

	first, err := ws.Open("original.opd", true)
	if err != nil {
		t.Fatalf("open original: %v", err)
	}
	if first.FileIdentity().InternalName != sharedGUID {
		t.Fatalf("first document id = %q, want the stored %q", first.FileIdentity().InternalName, sharedGUID)
	}

	second, err := ws.Open("clone.opd", true)
	if err != nil {
		t.Fatalf("open clone: %v", err)
	}

	if second.FileIdentity().InternalName == sharedGUID {
		t.Fatal("second document kept the colliding identity — two open files share a GUID")
	}
	if first.FileIdentity().InternalName == second.FileIdentity().InternalName {
		t.Fatalf("both open documents have id %q — identities must be unique within a session",
			first.FileIdentity().InternalName)
	}
	if !second.Dirty() {
		t.Error("reassigned document must be dirty so its new id is written on the next save")
	}
	if reassigned == nil {
		t.Fatal("no DocumentIdentityReassigned event fired")
	}
	if reassigned.Document != second || reassigned.PreviousInternalName != sharedGUID {
		t.Errorf("event = %+v, want clone reassigned away from %q", reassigned, sharedGUID)
	}
	if reassigned.NewInternalName != second.FileIdentity().InternalName {
		t.Errorf("event new id %q != document id %q", reassigned.NewInternalName, second.FileIdentity().InternalName)
	}
}

// TestClosingFreesIdentityForReopen proves the GUID index releases an identity on close, so a
// file can be reopened under its original id once the colliding document is gone.
func TestClosingFreesIdentityForReopen(t *testing.T) {
	const guid = "22222222-2222-4222-8222-222222222222"
	store := newFakeStore()
	store.saved["a.opd"] = storedDoc{docType: Part, displayName: "a", internalName: guid}
	ws := NewWorkspace(store)

	d, err := ws.Open("a.opd", true)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := ws.Close(d, true); err != nil {
		t.Fatalf("close: %v", err)
	}
	reopened, err := ws.Open("a.opd", true)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	if reopened.FileIdentity().InternalName != guid {
		t.Errorf("reopened id = %q, want the original %q (no live collision after close)",
			reopened.FileIdentity().InternalName, guid)
	}
}
