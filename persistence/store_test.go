// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"path/filepath"
	"testing"

	"github.com/Oblikovati/oblikovati/model/doc"
)

func TestWorkspaceRoundTripsThroughPackageStore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bracket.obk")
	store := NewPackageStore()

	ws := doc.NewWorkspace(store)
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
	reopened, err := doc.NewWorkspace(store).Open(path, true)
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

	ws := doc.NewWorkspace(store)
	d, err := ws.Add(doc.Part, src, true) // derived name "original", no override
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ws.Save(d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	editing := doc.NewWorkspace(store)
	reopened, err := editing.Open(src, true)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := editing.SaveAs(reopened, dst); err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	final, err := doc.NewWorkspace(store).Open(dst, true)
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
	if _, err := NewPackageStore().Load(bogus); err == nil {
		t.Error("Load accepted a package with no manifest as a document")
	}
}
