// SPDX-License-Identifier: GPL-2.0-only

package persistence

import (
	"os"
	"path/filepath"
	"testing"

	"oblikovati.org/model/doc"
)

// TestSaveCopyMintsAFreshFile: the copy carries the source content under a NEW
// identity, while the source keeps its binding, dirty state and identity
// (M03-F09, #610).
func TestSaveCopyMintsAFreshFile(t *testing.T) {
	dir := t.TempDir()
	store := NewPackageStore()
	ws := doc.NewWorkspace(store)
	src, err := ws.Add(doc.Part, filepath.Join(dir, "bracket.obk"), true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := ws.Save(src); err != nil {
		t.Fatalf("Save: %v", err)
	}
	src.MarkDirty()

	target := filepath.Join(dir, "bracket-rev3.obk")
	if err := ws.SaveCopy(src, target, doc.CopyMetadata{DisplayName: "Bracket rev3"}); err != nil {
		t.Fatalf("SaveCopy: %v", err)
	}
	if src.FullFileName() != filepath.Join(dir, "bracket.obk") || !src.Dirty() {
		t.Error("the source must keep its binding and dirty state")
	}

	copyDoc, err := doc.NewWorkspace(store).Open(target, true)
	if err != nil {
		t.Fatalf("Open copy: %v", err)
	}
	if copyDoc.DisplayName() != "Bracket rev3" {
		t.Errorf("copy display name = %q, want the override", copyDoc.DisplayName())
	}
	srcID, copyID := src.FileIdentity(), copyDoc.FileIdentity()
	if copyID.InternalName == srcID.InternalName || copyID.InternalName == "" {
		t.Errorf("copy internal name = %q, want a fresh GUID (source %q)", copyID.InternalName, srcID.InternalName)
	}
	if copyID.DatabaseRevisionID != srcID.DatabaseRevisionID {
		t.Errorf("copy database revision = %q, want the source's %q (same model)",
			copyID.DatabaseRevisionID, srcID.DatabaseRevisionID)
	}

	if err := ws.SaveCopy(src, src.FullFileName(), doc.CopyMetadata{}); err == nil {
		t.Error("copying onto the source path must fail")
	}
}

// TestSaveRetainsOldVersions: with retention on, each save archives the prior
// file as OldVersions/<name>.<n>.obk (.1 newest), pruned to the count.
func TestSaveRetainsOldVersions(t *testing.T) {
	dir := t.TempDir()
	store := NewPackageStore()
	store.SetOldVersionsToKeep(2)
	ws := doc.NewWorkspace(store)
	d, err := ws.Add(doc.Part, filepath.Join(dir, "bracket.obk"), true)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	for range [4]int{} {
		if err := ws.Save(d); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	old := filepath.Join(dir, "OldVersions")
	entries, err := os.ReadDir(old)
	if err != nil {
		t.Fatalf("ReadDir(OldVersions): %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("retained %d versions, want pruned to 2: %v", len(entries), entries)
	}
	for _, name := range []string{"bracket.1.obk", "bracket.2.obk"} {
		if _, err := os.Stat(filepath.Join(old, name)); err != nil {
			t.Errorf("missing retained version %s: %v", name, err)
		}
	}
	// .1 must be the most recent prior version: it carries save counter 3 (the
	// state just before the 4th save).
	pkg, err := OpenPackage(filepath.Join(old, "bracket.1.obk"))
	if err != nil {
		t.Fatalf("open retained version: %v", err)
	}
	if id := pkg.Identity(); id == nil || id.SaveCounter != 3 {
		t.Errorf("OldVersions/.1 identity = %+v, want save counter 3", pkg.Identity())
	}
}
