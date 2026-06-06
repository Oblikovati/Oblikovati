// SPDX-License-Identifier: GPL-2.0-only

package app_test

import (
	"errors"
	"path/filepath"
	"testing"

	"oblikovati/app"
	"oblikovati/model/compdef"
	"oblikovati/model/doc"
	"oblikovati/persistence"
)

// storeBackedSession wires a real .obk PackageStore into a session rooted at dir, so
// these tests exercise the genuine save→reopen path (CONVENTIONS.md M03: persistence
// PBIs assert a real round-trip, not a fake).
func storeBackedSession() *app.Session {
	return app.NewSessionWithStore(persistence.NewPackageStore())
}

func TestSaveActiveDocumentAsThenReopen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bracket.obk")

	s := storeBackedSession()
	if _, err := compdef.AddPart(s.Workspace(), "Part1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.SaveActiveDocumentAs(path); err != nil {
		t.Fatalf("SaveActiveDocumentAs(%q): %v", path, err)
	}

	// A fresh session must reopen the package with identical identity.
	reopened, err := storeBackedSession().OpenDocument(path)
	if err != nil {
		t.Fatalf("OpenDocument(%q): %v", path, err)
	}
	if got := reopened.DocumentType(); got != doc.Part {
		t.Errorf("reopened type = %v, want %v", got, doc.Part)
	}
	if got := reopened.DisplayName(); got != "bracket" {
		t.Errorf("reopened display name = %q, want %q", got, "bracket")
	}
}

func TestSaveActiveDocumentNeedsPathBeforeFirstSaveAs(t *testing.T) {
	s := storeBackedSession()
	if _, err := compdef.AddPart(s.Workspace(), "Part1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	// "Part1" has no .obk extension, so File ▸ Save must defer to Save As.
	if err := s.SaveActiveDocument(); !errors.Is(err, app.ErrNeedsPath) {
		t.Fatalf("SaveActiveDocument() = %v, want ErrNeedsPath", err)
	}
}

func TestSaveActiveDocumentSucceedsAfterSaveAs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plate.obk")
	s := storeBackedSession()
	if _, err := compdef.AddPart(s.Workspace(), "Part1", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if err := s.SaveActiveDocumentAs(path); err != nil {
		t.Fatalf("SaveActiveDocumentAs: %v", err)
	}
	// The document now carries a real .obk path, so a plain Save writes through.
	if err := s.SaveActiveDocument(); err != nil {
		t.Errorf("SaveActiveDocument() after Save As = %v, want nil", err)
	}
}

func TestSaveActiveDocumentNoActiveDocument(t *testing.T) {
	if err := storeBackedSession().SaveActiveDocument(); !errors.Is(err, app.ErrNoActiveDoc) {
		t.Fatalf("SaveActiveDocument() with no doc = %v, want ErrNoActiveDoc", err)
	}
}
