// SPDX-License-Identifier: GPL-2.0-only

package doc

import "testing"

// TestFileIdentityMintedAtCreation: every document file gets a unique GUID and
// the creating software version from the moment it exists (M03-F07, #159).
func TestFileIdentityMintedAtCreation(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	a, _ := ws.Add(Part, "a.obk", true)
	b, _ := ws.Add(Part, "b.obk", true)
	idA, idB := a.FileIdentity(), b.FileIdentity()
	if idA.InternalName == "" || idA.InternalName == idB.InternalName {
		t.Fatalf("internal names = (%q, %q), want unique non-empty GUIDs", idA.InternalName, idB.InternalName)
	}
	if idA.VersionCreated == "" {
		t.Error("VersionCreated must carry the creating software version")
	}
	if idA.SaveCounter != 0 || idA.RevisionID != "" {
		t.Errorf("identity before first save = %+v, want no revision stamps yet", idA)
	}
}

// TestSaveBumpsIdentity: a save advances the counter and re-mints the content
// revision; the database revision re-mints only when the recipe changed.
func TestSaveBumpsIdentity(t *testing.T) {
	ws := NewWorkspace(newFakeStore())
	d, _ := ws.Add(Part, "a.obk", true)
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	first := d.FileIdentity()
	if first.SaveCounter != 1 || first.RevisionID == "" || first.DatabaseRevisionID == "" {
		t.Fatalf("identity after first save = %+v, want counter 1 and minted revisions", first)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	second := d.FileIdentity()
	if second.SaveCounter != 2 || second.RevisionID == first.RevisionID {
		t.Errorf("second save = %+v, want counter 2 and a fresh content revision", second)
	}
	if second.DatabaseRevisionID != first.DatabaseRevisionID {
		t.Errorf("database revision re-minted on an unchanged model: %q → %q",
			first.DatabaseRevisionID, second.DatabaseRevisionID)
	}
	if second.InternalName != first.InternalName {
		t.Error("the internal name must survive every save")
	}
}

// TestFailedSaveRollsIdentityBack: identity must never drift ahead of the
// bytes on disk.
func TestFailedSaveRollsIdentityBack(t *testing.T) {
	ws := NewWorkspace(&failingStore{})
	d, _ := ws.Add(Part, "a.obk", true)
	before := d.FileIdentity()
	if err := ws.Save(d); err == nil {
		t.Fatal("a failing store must surface the save error")
	}
	if got := d.FileIdentity(); got != before {
		t.Errorf("identity after failed save = %+v, want untouched %+v", got, before)
	}
}

// failingStore is a Store whose writes always fail.
type failingStore struct{}

func (s *failingStore) Save(*Document) error { return errNotStored{"write failed"} }
func (s *failingStore) SaveCopy(*Document, string, CopyMetadata) error {
	return errNotStored{"write failed"}
}
func (s *failingStore) Load(name string) (*Document, error) {
	return nil, errNotStored{name}
}
func (s *failingStore) Exists(string) bool { return false }

// TestCopyIdentityMintsFreshNameKeepsModelStamp: a copy is a NEW file — its internal name must
// differ from the source's — but it documents the source's model, so the database revision and
// digest carry over (a derived part keyed on them still matches). (M03-F09)
func TestCopyIdentityMintsFreshNameKeepsModelStamp(t *testing.T) {
	src := newFileIdentity()
	src.DatabaseRevisionID = "db-rev-123"
	src.ModelDigest = "digest-abc"

	cp := CopyIdentity(src)

	if cp.InternalName == src.InternalName || cp.InternalName == "" {
		t.Errorf("copy internal name = %q, want a fresh GUID distinct from source %q", cp.InternalName, src.InternalName)
	}
	if cp.DatabaseRevisionID != src.DatabaseRevisionID || cp.ModelDigest != src.ModelDigest {
		t.Errorf("copy model stamp = (%q, %q), want the source's (%q, %q)",
			cp.DatabaseRevisionID, cp.ModelDigest, src.DatabaseRevisionID, src.ModelDigest)
	}
}
