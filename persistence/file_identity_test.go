// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
)

// TestFileIdentityRoundTripsThroughPackage: the identity block persists in the
// .obk and survives a reload in a fresh workspace; a further save continues
// the counter instead of restarting it (M03-F07, #159).
func TestFileIdentityRoundTripsThroughPackage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bracket.obk")
	store := NewPackageStore()

	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	saved := d.FileIdentity()

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !strings.Contains(string(raw), "internalName: "+saved.InternalName) {
		t.Errorf("the .obk must carry the identity block; got:\n%s", raw)
	}

	ws2 := doc.NewWorkspace(store, contentset.Default())
	reopened, err := ws2.Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	got := reopened.FileIdentity()
	if got.InternalName != saved.InternalName || got.SaveCounter != 1 {
		t.Fatalf("reloaded identity = %+v, want the persisted %+v", got, saved)
	}
	if err := ws2.Save(reopened); err != nil {
		t.Fatalf("Save after reload: %v", err)
	}
	if next := reopened.FileIdentity(); next.SaveCounter != 2 || next.InternalName != saved.InternalName {
		t.Errorf("identity after reload+save = %+v, want counter 2 under the same GUID", next)
	}
}

// TestFileReferencesRoundTripThroughPackage: as-saved reference records reload
// with the owner so a file whose targets vanished still reports what it tried
// to reference (M03-F07).
func TestFileReferencesRoundTripThroughPackage(t *testing.T) {
	dir := t.TempDir()
	store := NewPackageStore()
	ws := doc.NewWorkspace(store, contentset.Default())
	part, err := ws.Add(doc.Part, filepath.Join(dir, "pin.obk"), true)
	if err != nil {
		t.Fatalf("Add part: %v", err)
	}
	if err := ws.Save(part); err != nil {
		t.Fatalf("Save part: %v", err)
	}
	owner, err := ws.Add(doc.Assembly, filepath.Join(dir, "frame.obk"), true)
	if err != nil {
		t.Fatalf("Add owner: %v", err)
	}
	if _, err := owner.AddReference(part.FullDocumentName()); err != nil {
		t.Fatalf("AddReference: %v", err)
	}
	if err := ws.Save(owner); err != nil {
		t.Fatalf("Save owner: %v", err)
	}

	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(owner.FullFileName(), true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	refs := reopened.FileReferences()
	if len(refs) != 1 || refs[0].FullFileName() != part.FullFileName() {
		t.Fatalf("reloaded references = %+v, want the pin record", refs)
	}
	if refs[0].RelativeFileName() != "pin.obk" {
		t.Errorf("relative name = %q, want pin.obk", refs[0].RelativeFileName())
	}
	if refs[0].ReferencedFileInternalName() != part.FileIdentity().InternalName {
		t.Errorf("recorded target GUID = %q, want %q",
			refs[0].ReferencedFileInternalName(), part.FileIdentity().InternalName)
	}

}

// TestResaveWithoutLiveEdgesKeepsReferenceRecords: the reference graph is
// rebuilt lazily after a load, so saving a freshly reopened owner must keep
// the persisted records rather than wiping them.
func TestResaveWithoutLiveEdgesKeepsReferenceRecords(t *testing.T) {
	dir := t.TempDir()
	store := NewPackageStore()
	ws := doc.NewWorkspace(store, contentset.Default())
	part, _ := ws.Add(doc.Part, filepath.Join(dir, "pin.obk"), true)
	if err := ws.Save(part); err != nil {
		t.Fatalf("Save part: %v", err)
	}
	owner, _ := ws.Add(doc.Assembly, filepath.Join(dir, "frame.obk"), true)
	if _, err := owner.AddReference(part.FullDocumentName()); err != nil {
		t.Fatalf("AddReference: %v", err)
	}
	if err := ws.Save(owner); err != nil {
		t.Fatalf("Save owner: %v", err)
	}

	ws2 := doc.NewWorkspace(store, contentset.Default())
	reopened, err := ws2.Open(owner.FullFileName(), true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := ws2.Save(reopened); err != nil {
		t.Fatalf("re-Save: %v", err)
	}
	if refs := reopened.FileReferences(); len(refs) != 1 {
		t.Fatalf("references after edge-less re-save = %d, want the record kept", len(refs))
	}
}
