// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"testing"

	"oblikovati.org/api/types"
)

// referencingWorkspace is one assembly-like owner in /asm referencing a part
// in the same tree — the canonical descriptor fixture.
func referencingWorkspace(t *testing.T) (*Workspace, *Document, *Document) {
	t.Helper()
	ws := NewWorkspace(newFakeStore(), nil)
	part, _ := ws.Add(Part, "/asm/parts/pin.obk", true)
	if err := ws.Save(part); err != nil {
		t.Fatalf("Save part: %v", err)
	}
	owner, _ := ws.Add(Assembly, "/asm/frame.obk", true)
	if _, err := owner.AddReference(part.FullDocumentName()); err != nil {
		t.Fatalf("AddReference: %v", err)
	}
	if err := ws.Save(owner); err != nil {
		t.Fatalf("Save owner: %v", err)
	}
	return ws, owner, part
}

// TestSaveSnapshotsFileReferences: saving records the as-saved descriptor with
// the target's identity, the owner-relative spelling, and upToDate status.
func TestSaveSnapshotsFileReferences(t *testing.T) {
	_, owner, part := referencingWorkspace(t)
	refs := owner.FileReferences()
	if len(refs) != 1 {
		t.Fatalf("references = %d, want the one part reference", len(refs))
	}
	r := refs[0]
	if r.FullFileName() != "/asm/parts/pin.obk" || r.RelativeFileName() != "parts/pin.obk" {
		t.Errorf("names = (%q, %q), want the absolute and owner-relative spellings",
			r.FullFileName(), r.RelativeFileName())
	}
	if r.LocationType() != types.LocationOwnerDirectory {
		t.Errorf("location = %v, want ownerDirectory", r.LocationType())
	}
	if r.ReferencedFileInternalName() != part.FileIdentity().InternalName || r.FileSaveCounter() != 1 {
		t.Errorf("recorded identity = (%q, %d), want the part's GUID at save counter 1",
			r.ReferencedFileInternalName(), r.FileSaveCounter())
	}
	if r.Status() != types.ReferenceUpToDate || !r.DocumentFound() || r.DifferentDocument() {
		t.Errorf("status = %v (found=%v different=%v), want a clean upToDate reference",
			r.Status(), r.DocumentFound(), r.DifferentDocument())
	}
}

// TestReferenceGoesOutOfDateWhenTargetSaves: the target saving again after the
// owner recorded it reports outOfDate on the owner's record.
func TestReferenceGoesOutOfDateWhenTargetSaves(t *testing.T) {
	ws, owner, part := referencingWorkspace(t)
	if err := ws.Save(part); err != nil {
		t.Fatalf("Save part again: %v", err)
	}
	if got := owner.FileReferences()[0].Status(); got != types.ReferenceOutOfDate {
		t.Errorf("status after target re-save = %v, want outOfDate", got)
	}
}

// TestMissingReferenceReportsAndRepairs: a vanished target reports missing
// with full detail, and ReplaceReference re-points record and graph edge.
func TestMissingReferenceReportsAndRepairs(t *testing.T) {
	ws, owner, part := referencingWorkspace(t)
	store := ws.store.(*fakeStore)
	delete(store.saved, part.FullDocumentName())
	if err := ws.Close(part, true); err != nil {
		t.Fatalf("Close part: %v", err)
	}

	r := owner.FileReferences()[0]
	if r.Status() != types.ReferenceMissing || !r.ReferenceMissing() || r.DocumentFound() {
		t.Fatalf("status = %v, want a missing reference", r.Status())
	}

	moved, _ := ws.Add(Part, "/asm/parts/pin-v2.obk", true)
	if err := ws.Save(moved); err != nil {
		t.Fatalf("Save moved part: %v", err)
	}
	if err := r.ReplaceReference("/no/such/file.obk"); err == nil {
		t.Error("replacing with a nonexistent target must fail")
	}
	if err := r.ReplaceReference("/asm/parts/pin-v2.obk"); err != nil {
		t.Fatalf("ReplaceReference: %v", err)
	}
	if r.Status() != types.ReferenceReplaced || r.ResolvedFileName() != "/asm/parts/pin-v2.obk" {
		t.Errorf("after repair status = %v resolved = %q, want replaced → pin-v2",
			r.Status(), r.ResolvedFileName())
	}
	if !r.DifferentDocument() || !r.ReferenceLocationDifferent() {
		t.Error("a repaired reference must report it resolves to a different document")
	}
	if docs := owner.ReferencedDocuments(); len(docs) != 1 || docs[0].FullDocumentName() != "/asm/parts/pin-v2.obk" {
		t.Errorf("graph resolution after repair = %v, want the replacement", docs)
	}

	// Saving makes the repair the as-saved truth.
	if err := ws.Save(owner); err != nil {
		t.Fatalf("Save owner: %v", err)
	}
	r = owner.FileReferences()[0]
	if r.FullFileName() != "/asm/parts/pin-v2.obk" || r.Status() != types.ReferenceUpToDate {
		t.Errorf("after save = (%q, %v), want the replacement persisted upToDate", r.FullFileName(), r.Status())
	}
}

// TestFileViewExposesIdentityAndReferences: the File object bridges identity,
// load state and the reference records (M03-F07).
func TestFileViewExposesIdentityAndReferences(t *testing.T) {
	ws, owner, _ := referencingWorkspace(t)
	f, ok := ws.FileByName(owner.FullFileName())
	if !ok {
		t.Fatal("FileByName must find the open owner")
	}
	if f.InternalName() != owner.FileIdentity().InternalName || f.FileSaveCounter() != 1 {
		t.Errorf("file identity = (%q, %d), want the owner's", f.InternalName(), f.FileSaveCounter())
	}
	if !f.Loaded() || len(f.Documents()) != 1 || len(f.References()) != 1 {
		t.Errorf("file view = (loaded=%v docs=%d refs=%d), want the loaded single-document file",
			f.Loaded(), len(f.Documents()), len(f.References()))
	}
	if _, ok := ws.FileByName("/nowhere.obk"); ok {
		t.Error("FileByName must miss unknown names")
	}
}
