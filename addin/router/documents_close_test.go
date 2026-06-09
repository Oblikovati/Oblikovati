// SPDX-License-Identifier: GPL-2.0-only

package router

import (
	"strconv"
	"testing"

	"oblikovati.org/api/wire"
)

// TestCloseAllDocuments verifies the documents.closeAll endpoint clears the
// workspace so a test session can start fresh, and that documents.create then
// works again with a previously-used name (the collision that motivated this).
func TestCloseAllDocuments(t *testing.T) {
	r, s := seededSession(t) // one open part document
	call(t, r, s, wire.MethodDocumentsCreate, `{"type":"part","name":"part-a"}`, nil)
	if n := s.Workspace().Count(); n < 2 {
		t.Fatalf("expected >=2 open documents, got %d", n)
	}

	var res wire.CloseDocumentsResult
	call(t, r, s, wire.MethodDocumentsCloseAll, `{"force":true}`, &res)
	if res.Closed < 2 {
		t.Errorf("closed %d documents, want >=2", res.Closed)
	}
	if n := s.Workspace().Count(); n != 0 {
		t.Errorf("workspace has %d documents after closeAll, want 0", n)
	}

	// The freed name is reusable now.
	call(t, r, s, wire.MethodDocumentsCreate, `{"type":"part","name":"part-a"}`, nil)
	if n := s.Workspace().Count(); n != 1 {
		t.Errorf("workspace has %d documents after re-create, want 1", n)
	}
}

// TestCloseDocumentByID closes a single document by id and leaves the rest.
func TestCloseDocumentByID(t *testing.T) {
	r, s := seededSession(t)
	var created wire.DocumentInfo
	call(t, r, s, wire.MethodDocumentsCreate, `{"type":"part","name":"to-close"}`, &created)
	before := s.Workspace().Count()

	var res wire.CloseDocumentsResult
	call(t, r, s, wire.MethodDocumentsClose, `{"id":`+strconv.FormatUint(created.ID, 10)+`,"force":true}`, &res)
	if res.Closed != 1 {
		t.Errorf("closed %d, want 1", res.Closed)
	}
	if n := s.Workspace().Count(); n != before-1 {
		t.Errorf("workspace has %d documents, want %d", n, before-1)
	}
}
