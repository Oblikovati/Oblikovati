// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"strings"
	"testing"
)

// fakeFS is an in-memory FileSystem for store tests — no real disk IO.
type fakeFS struct{ files map[string][]byte }

func newFakeFS() *fakeFS { return &fakeFS{files: map[string][]byte{}} }

func (f *fakeFS) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, &missingError{path}
	}
	return data, nil
}

func (f *fakeFS) WriteFile(path string, data []byte) error {
	f.files[path] = data
	return nil
}

type missingError struct{ path string }

func (e *missingError) Error() string { return "missing: " + e.path }

func TestStoreSaveLoadProjectAssets(t *testing.T) {
	fs := newFakeFS()
	store := NewStore("/proj/DesignData", fs)

	src := NewLibrary()
	dup, _ := src.DuplicateAppearance("steel", "Brushed Alu", SourceProject)
	spec := dup.Spec()
	spec.Albedo = mustColor("#c0c4c8ff")
	src.EditAppearance(dup.ID(), spec)
	if _, err := src.DuplicateMaterial("aluminum-6061", "Shop Alu", SourceProject); err != nil {
		t.Fatalf("DuplicateMaterial: %v", err)
	}
	if err := store.Save(src); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// A second session loads the project library and sees the custom assets.
	dst := NewLibrary()
	if err := store.Load(dst); err != nil {
		t.Fatalf("Load: %v", err)
	}
	got, ok := dst.Appearance(dup.ID())
	if !ok || got.Source() != SourceProject || got.Albedo() != mustColor("#c0c4c8ff") {
		t.Errorf("project appearance not loaded: %v ok=%v", got, ok)
	}
	if _, ok := dst.Material("shop-alu"); !ok {
		t.Errorf("project material not loaded; have %d materials", len(dst.Materials()))
	}
	// Built-ins must NOT be written to the project library file.
	if data, ok := fs.files["/proj/DesignData/materials.yaml"]; ok && strings.Contains(string(data), "id: steel") {
		t.Error("built-in material was persisted to the project library")
	}
}

func TestStoreLoadEmptyOnFirstRun(t *testing.T) {
	store := NewStore("/proj/DesignData", newFakeFS())
	lib := NewLibrary()
	before := len(lib.Materials())
	if err := store.Load(lib); err != nil {
		t.Fatalf("Load on empty project: %v", err)
	}
	if len(lib.Materials()) != before {
		t.Errorf("empty project Load changed the library (%d → %d)", before, len(lib.Materials()))
	}
}
