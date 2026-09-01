// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"strings"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app/options"
	"oblikovati.org/model/doc"
)

// savedSession is a session over the in-memory fakeDocStore with one saved
// part at parts/pin.opd referenced by an assembly at frame.oad.
func savedSession(t *testing.T) (*Session, *fakeDocStore, *doc.Document, *doc.Document) {
	t.Helper()
	store := newFakeDocStore()
	s := NewSessionWithStore(store)
	part, err := s.Workspace().Add(doc.Part, "parts/pin.opd", true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	if err := s.Workspace().Save(part); err != nil {
		t.Fatalf("Save part: %v", err)
	}
	owner, err := s.Workspace().Add(doc.Assembly, "frame.oad", true)
	if err != nil {
		t.Fatalf("Add owner: %v", err)
	}
	if _, err := owner.AddReference(part.FullDocumentName()); err != nil {
		t.Fatalf("AddReference: %v", err)
	}
	return s, store, owner, part
}

// TestSaveDependentsPolicySavesDirtyReferences: with the policy on, saving the
// owner first saves its dirty referenced documents (M03-F09).
func TestSaveDependentsPolicySavesDirtyReferences(t *testing.T) {
	t.Parallel()
	s, _, owner, part := savedSession(t)
	part.MarkDirty()

	if err := s.SaveDocument(owner); err != nil {
		t.Fatalf("SaveDocument without the policy: %v", err)
	}
	if !part.Dirty() {
		t.Fatal("without the policy the dependent must stay dirty")
	}

	if err := s.SetSaveOptions(options.Save{Thumbnail: types.ThumbnailNone, SaveDependents: true}); err != nil {
		t.Fatalf("SetSaveOptions: %v", err)
	}
	if err := s.SaveDocument(owner); err != nil {
		t.Fatalf("SaveDocument with the policy: %v", err)
	}
	if part.Dirty() {
		t.Error("the dirty dependent must be saved before the owner")
	}
}

// TestSetSaveOptionsRejectsDeadSettings: unsupported thumbnail modes and
// negative retention are rejected, never persisted (the no-dead-settings rule).
func TestSetSaveOptionsRejectsDeadSettings(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if err := s.SetSaveOptions(options.Save{Thumbnail: types.ThumbnailIsoViewOnSave}); err == nil {
		t.Error("an unimplemented thumbnail mode must be rejected")
	}
	if err := s.SetSaveOptions(options.Save{Thumbnail: types.ThumbnailNone, OldVersionsToKeep: -1}); err == nil {
		t.Error("negative retention must be rejected")
	}
	if err := s.SetSaveOptions(options.Save{Thumbnail: types.ThumbnailActiveWindowOnSave, OldVersionsToKeep: 3}); err != nil {
		t.Fatalf("a supported policy must be accepted: %v", err)
	}
	if got := s.Options().Save; got.Thumbnail != types.ThumbnailActiveWindowOnSave || got.OldVersionsToKeep != 3 {
		t.Errorf("save options = %+v, want the accepted policy applied", got)
	}
}

// TestSavePolicyControlContract: the contract.SaveOptions view reads and
// writes the same group.
func TestSavePolicyControlContract(t *testing.T) {
	t.Parallel()
	s := NewSession()
	c := s.SavePolicy()
	if c.Thumbnail() != types.ThumbnailNone || c.SaveDependents() || c.OldVersionsToKeep() != 0 {
		t.Fatalf("defaults = (%v, %v, %d), want none/false/0", c.Thumbnail(), c.SaveDependents(), c.OldVersionsToKeep())
	}
	c.SetSaveDependents(true)
	if !c.SaveDependents() {
		t.Error("SetSaveDependents must apply to the running session")
	}
	if err := c.SetThumbnail(types.ThumbnailImportFromFile); err == nil {
		t.Error("SetThumbnail must reject unimplemented modes")
	}
	if err := c.SetOldVersionsToKeep(5); err != nil || c.OldVersionsToKeep() != 5 {
		t.Errorf("SetOldVersionsToKeep = (%v, %d), want 5 applied", err, c.OldVersionsToKeep())
	}
}

// TestBatchSavePerFileOutcomes: one bad item does not abort the batch; the
// queue drains on execute (M03-F09).
func TestBatchSavePerFileOutcomes(t *testing.T) {
	t.Parallel()
	s, store, owner, part := savedSession(t)
	q := s.NewBatchSave()
	if err := q.AddFileToSave(part, "exports/pin-copy.opd"); err != nil {
		t.Fatalf("AddFileToSave: %v", err)
	}
	if err := q.AddFileToSave(owner, "exports/pin-copy.opd"); err == nil {
		t.Fatal("a duplicate target must be rejected at queue time")
	}
	if err := q.AddFileToSave(owner, "parts/pin.opd"); err != nil {
		t.Fatalf("AddFileToSave owner: %v", err)
	}
	if q.Count() != 2 {
		t.Fatalf("count = %d, want 2", q.Count())
	}

	// saveCopyAs: the part copies fine; the owner targets an OPEN file name,
	// which the workspace rejects — the batch must carry that per-file error.
	outcomes := q.ExecuteSaveCopyAs()
	if len(outcomes) != 2 || q.Count() != 0 {
		t.Fatalf("outcomes = %d (queued %d), want 2 outcomes and a drained queue", len(outcomes), q.Count())
	}
	if outcomes[0].Err != nil || !store.Exists("exports/pin-copy.opd") {
		t.Errorf("outcome[0] = %+v, want the pin copied", outcomes[0])
	}
	if outcomes[1].Err == nil || !strings.Contains(outcomes[1].Err.Error(), "open in this workspace") {
		t.Errorf("outcome[1] = %+v, want the open-target failure carried", outcomes[1])
	}
}
