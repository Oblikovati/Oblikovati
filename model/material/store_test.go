// SPDX-License-Identifier: GPL-2.0-only

package material

import (
	"os"
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
	dup, _ := src.DuplicateAppearance(DefaultAppearanceID, "Brushed Alu", SourceProject)
	spec := dup.Spec()
	spec.Base.Metalness = 1
	spec.Specular.Roughness = 0.2
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
	if !ok || got.Source() != SourceProject || got.Base().Metalness != 1 || got.Specular().Roughness != 0.2 {
		t.Errorf("project appearance not loaded: %+v ok=%v", got, ok)
	}
	if _, ok := dst.Material("shop-alu"); !ok {
		t.Errorf("project material not loaded; have %d materials", len(dst.Materials()))
	}
	// Built-ins must NOT be written to the project library file.
	if data, ok := fs.files[store.materialPath()]; !ok {
		t.Error("Save did not write the project materials file")
	} else if strings.Contains(string(data), "id: steel") {
		t.Error("built-in material was persisted to the project library")
	}
	if data, ok := fs.files[store.appearancePath()]; !ok {
		t.Error("Save did not write the project appearances file")
	} else if strings.Contains(string(data), "id: "+DefaultAppearanceID) {
		t.Error("built-in appearance was persisted to the project library")
	}
}

// TestLoadProjectLibraryMigratesLegacyShapedAppearance is M46-F04's regression for the
// project-library path: an old appearances.yaml saved before the OpenPBR consolidation
// (5-scalar shape, a top-level "albedo" key) must load correctly via the same
// AppearanceRecipe.UnmarshalYAML shape-sniff the document-recipe path uses — covered
// "for free" since Store.Load unmarshals into the same AppearanceRecipe type.
func TestLoadProjectLibraryMigratesLegacyShapedAppearance(t *testing.T) {
	raw, err := os.ReadFile("../../test-utilities/openpbr-appearance-migration/old-project-library.yaml")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	fs := newFakeFS()
	store := NewStore("/proj/DesignData", fs)
	fs.files[store.appearancePath()] = raw

	lib := NewLibrary()
	if err := store.Load(lib); err != nil {
		t.Fatalf("Load: %v", err)
	}
	a, ok := lib.Appearance("my-custom-red")
	if !ok {
		t.Fatal("legacy-shaped project appearance was not migrated")
	}
	if a.Source() != SourceProject {
		t.Errorf("migrated appearance source = %q, want project", a.Source())
	}
	if a.Base().Metalness != 0 || a.Base().Color == (Color3{}) {
		t.Errorf("migrated appearance base = %+v, want a non-zero color and metalness 0", a.Base())
	}
	if a.Specular().Roughness != 0.4 {
		t.Errorf("migrated appearance specular roughness = %v, want 0.4", a.Specular().Roughness)
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
