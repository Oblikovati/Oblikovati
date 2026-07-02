// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"errors"
	"testing"

	"oblikovati.org/event"
	"oblikovati.org/model/doc"
)

// fakeDocStore is an in-memory doc.Store: enough persistence for the file-UI
// hook tests to exercise real open/save flows without touching disk.
type fakeDocStore struct{ saved map[string]doc.DocumentType }

func newFakeDocStore() *fakeDocStore { return &fakeDocStore{saved: map[string]doc.DocumentType{}} }

func (f *fakeDocStore) Save(d *doc.Document) error {
	f.saved[d.FullDocumentName()] = d.DocumentType()
	return nil
}

func (f *fakeDocStore) SaveCopy(d *doc.Document, target string, _ doc.CopyMetadata) error {
	f.saved[target] = d.DocumentType()
	return nil
}

func (f *fakeDocStore) Load(name string, factories doc.ContentFactories) (*doc.Document, error) {
	t, ok := f.saved[name]
	if !ok {
		return nil, errors.New("fakeDocStore: nothing at " + name)
	}
	return doc.Restore(t, name, "", factories)
}

func (f *fakeDocStore) Exists(name string) bool { _, ok := f.saved[name]; return ok }

// TestFileNewVetoBlocksNewPart: a Before FileNew handler vetoing (a vault's
// checkout-first policy) stops the document from being created.
func TestFileNewVetoBlocksNewPart(t *testing.T) {
	s := NewSession()
	var after []FileNew
	event.Subscribe(s.Events(), event.After, func(_ event.Context, e FileNew) event.Outcome {
		after = append(after, e)
		return event.Continue()
	})

	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if len(after) != 1 || after[0].DocumentType != doc.Part {
		t.Fatalf("After events = %+v, want one part FileNew", after)
	}

	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e FileNew) event.Outcome {
		return event.Veto("check out a project first")
	})
	if _, err := s.NewPart(); err == nil {
		t.Fatal("a vetoed FileNew must block the create")
	}
	if n := len(s.Workspace().Documents()); n != 1 {
		t.Errorf("documents after veto = %d, want still 1", n)
	}
	if len(after) != 1 {
		t.Errorf("After events after veto = %d, want still 1", len(after))
	}
}

// TestFileDialogHooksSupplyPaths: a handler answering a dialog hook replaces
// the head's explorer — the seam returns the supplied path; unanswered hooks
// report false so the head shows the dialog.
func TestFileDialogHooksSupplyPaths(t *testing.T) {
	s := NewSession()
	if _, ok := s.HookFileOpenDialog(); ok {
		t.Error("an unanswered open hook must report false")
	}

	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e FileOpenDialog) event.Outcome {
		e.Supply("/vault/bracket.obk")
		return event.Handle()
	})
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e FileSaveAsDialog) event.Outcome {
		if !e.SaveCopyAs {
			e.Supply("/vault/out.obk")
		}
		return event.Handle()
	})
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e FileNewDialog) event.Outcome {
		e.Supply("/templates/steel-part.obk")
		return event.Handle()
	})

	if path, ok := s.HookFileOpenDialog(); !ok || path != "/vault/bracket.obk" {
		t.Errorf("open hook = (%q, %v), want the supplied vault path", path, ok)
	}
	if path, ok := s.HookFileSaveAsDialog(false); !ok || path != "/vault/out.obk" {
		t.Errorf("save-as hook = (%q, %v), want the supplied destination", path, ok)
	}
	if _, ok := s.HookFileSaveAsDialog(true); ok {
		t.Error("the save-copy-as variant was answered though the handler skips it")
	}
	if path, ok := s.HookFileNewDialog(); !ok || path != "/templates/steel-part.obk" {
		t.Errorf("new hook = (%q, %v), want the supplied template", path, ok)
	}
}

// TestOpenDocumentFromMRUHonorsVetoAndRecords: recent-files entries open behind
// the vetoable hook, and open/save-as activity maintains the bounded MRU list.
func TestOpenDocumentFromMRUHonorsVetoAndRecords(t *testing.T) {
	store := newFakeDocStore()
	s := NewSessionWithStore(store)
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	if err := s.SaveActiveDocumentAs("/models/a.obk"); err != nil {
		t.Fatalf("SaveActiveDocumentAs: %v", err)
	}
	if got := s.RecentDocuments(); len(got) != 1 || got[0] != "/models/a.obk" {
		t.Fatalf("RecentDocuments after save-as = %v, want [/models/a.obk]", got)
	}
	if err := s.Workspace().Close(s.ActiveDocument(), true); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := s.OpenDocumentFromMRU("/models/a.obk"); err != nil {
		t.Fatalf("OpenDocumentFromMRU: %v", err)
	}
	if got := s.RecentDocuments(); len(got) != 1 || got[0] != "/models/a.obk" {
		t.Errorf("RecentDocuments must dedupe re-opens, got %v", got)
	}

	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e FileOpenFromMRU) event.Outcome {
		return event.Veto("the vault says this file moved")
	})
	_, err := s.OpenDocumentFromMRU("/models/a.obk")
	var veto *doc.VetoError
	if !errors.As(err, &veto) {
		t.Fatalf("vetoed MRU open = %v, want a *doc.VetoError", err)
	}
}

// TestPopulateFileMetadataCollectsAroundSave: handlers contribute entries when
// a document saves, queryable afterwards per document.
func TestPopulateFileMetadataCollectsAroundSave(t *testing.T) {
	s := NewSessionWithStore(newFakeDocStore())
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("NewPart: %v", err)
	}
	event.Subscribe(s.Events(), event.Before, func(_ event.Context, e PopulateFileMetadata) event.Outcome {
		e.Add("author", "vm")
		e.Add("revision", "B")
		return event.Handle()
	})

	if err := s.SaveActiveDocumentAs("/models/meta.obk"); err != nil {
		t.Fatalf("SaveActiveDocumentAs: %v", err)
	}
	got := s.FileMetadata(s.ActiveDocument().ID())
	if len(got) != 2 || got[0] != (FileMetadataValue{Name: "author", Value: "vm"}) {
		t.Errorf("FileMetadata = %+v, want the two contributed entries", got)
	}
}
