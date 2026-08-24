// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"strings"
	"testing"
)

func TestStoreSaveLoadProjectOpenPBRAppearances(t *testing.T) {
	fs := newFakeFS()
	store := NewStore("/proj/DesignData", fs)

	src := NewLibrary()
	dup, _ := src.DuplicateOpenPBRAppearance(DefaultOpenPBRAppearanceID, "Brushed Alu", SourceProject)
	spec := dup.Spec()
	spec.Base.Metalness = 1
	spec.Specular.Roughness = 0.2
	src.EditOpenPBRAppearance(dup.ID(), spec)
	if err := store.Save(src); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// A second session loads the project library and sees the custom asset.
	dst := NewLibrary()
	if err := store.Load(dst); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := dst.OpenPBRAppearance(dup.ID())
	if !ok || got.Source() != SourceProject || got.Base().Metalness != 1 || got.Specular().Roughness != 0.2 {
		t.Errorf("project openpbr appearance not loaded: %+v ok=%v", got, ok)
	}
	// Built-ins must NOT be written to the project library file.
	if data, ok := fs.files["/proj/DesignData/openpbr-appearances.yaml"]; ok &&
		strings.Contains(string(data), "id: "+DefaultOpenPBRAppearanceID) {
		t.Error("built-in openpbr appearance was persisted to the project library")
	}
}

func TestStoreLoadEmptyOnFirstRunLeavesOpenPBRUntouched(t *testing.T) {
	store := NewStore("/proj/DesignData", newFakeFS())
	lib := NewLibrary()
	before := len(lib.OpenPBRAppearances())
	if err := store.Load(lib); err != nil {
		t.Fatalf("Load on empty project: %v", err)
	}
	if len(lib.OpenPBRAppearances()) != before {
		t.Errorf("empty project Load changed the openpbr library (%d → %d)", before, len(lib.OpenPBRAppearances()))
	}
}
