// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"path/filepath"
	"testing"

	"oblikovati.org/model/contentset"
	"oblikovati.org/model/doc"
)

func TestWorkspaceRoundTripsThroughPackageStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bracket.obk")
	store := NewPackageStore()

	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	d.SetDisplayName("Bracket")
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if !store.Exists(path) {
		t.Fatalf("package not written to %q", path)
	}

	// Reopen in a fresh workspace to force the on-disk load path.
	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if reopened.DocumentType() != doc.Part {
		t.Errorf("reopened type = %v, want Part", reopened.DocumentType())
	}
	if reopened.DisplayName() != "Bracket" {
		t.Errorf("reopened display name = %q, want Bracket", reopened.DisplayName())
	}
	if !reopened.Open() || reopened.Dirty() {
		t.Error("reopened document should be open and clean")
	}
}

// TestBodyNameSurvivesStoreRoundTrip exercises the store/restore hooks (#1078): a renamed body
// reopens with its stored name, and the reopened document is clean (restore does not dirty it).
func TestBodyNameSurvivesStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "case.obk")
	store := NewPackageStore()

	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := ws.Add(doc.Part, path, true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	d.SetBodyName("body-ref-key", "Housing")
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	reopened, err := doc.NewWorkspace(store, contentset.Default()).Open(path, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if name, ok := reopened.BodyName("body-ref-key"); !ok || name != "Housing" {
		t.Errorf("reopened body name = (%q, %v), want (Housing, true)", name, ok)
	}
	if reopened.Dirty() {
		t.Error("reopened document should be clean (RestoreBodyNames must not dirty it)")
	}
}

func TestPackageStoreExists(t *testing.T) {
	dir := t.TempDir()
	store := NewPackageStore()
	if store.Exists(filepath.Join(dir, "nope.obk")) {
		t.Error("Exists reported a missing file as present")
	}
	if store.Exists(dir) {
		t.Error("Exists reported a directory as a package file")
	}
}

// TestDerivedDisplayNameFollowsSaveAs guards the Restore refinement: a document
// saved with a name derived from its file (no explicit override) must, after
// save-as to a new path, reopen under the NEW base name — not the stale original.
func TestDerivedDisplayNameFollowsSaveAs(t *testing.T) {
	dir := t.TempDir()
	store := NewPackageStore()
	src := filepath.Join(dir, "original.obk")
	dst := filepath.Join(dir, "renamed.obk")

	ws := doc.NewWorkspace(store, contentset.Default())
	d, err := ws.Add(doc.Part, src, true) // derived name "original", no override
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	editing := doc.NewWorkspace(store, contentset.Default())
	reopened, err := editing.Open(src, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := editing.SaveAs(reopened, dst); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	final, err := doc.NewWorkspace(store, contentset.Default()).Open(dst, true)
	if err != nil {
		t.Fatalf("Open renamed: %v", err)
	}
	if got := final.DisplayName(); got != "renamed" {
		t.Errorf("display name after save-as = %q, want %q (derived names must follow the file)", got, "renamed")
	}
}

func TestPackageStoreLoadRejectsNonPackage(t *testing.T) {
	dir := t.TempDir()
	bogus := filepath.Join(dir, "bogus.obk")
	if err := WriteDataToFile(bogus, "client.bin", []byte("no manifest here")); err != nil {
		t.Fatalf("seed bogus package: %v", err)
	}
	if _, err := NewPackageStore().Load(bogus, nil); err == nil {
		t.Error("Load accepted a package with no manifest as a document")
	}
}
