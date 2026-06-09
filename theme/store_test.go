// SPDX-License-Identifier: GPL-2.0-only

package theme

import (
	"path/filepath"
	"strings"
	"testing"

	"oblikovati.org/api/types"
)

// fakeFS is an in-memory FileSystem for store tests — no real disk IO, so the tests are
// fast and isolated (CLAUDE.md: mock filesystem with a named fake).
type fakeFS struct {
	files map[string][]byte
}

func newFakeFS() *fakeFS { return &fakeFS{files: map[string][]byte{}} }

func (f *fakeFS) ReadDir(dir string) ([]string, error) {
	var names []string
	for path := range f.files {
		if filepath.Dir(path) == dir {
			names = append(names, filepath.Base(path))
		}
	}
	return names, nil
}

func (f *fakeFS) ReadFile(path string) ([]byte, error) {
	data, ok := f.files[path]
	if !ok {
		return nil, &notFoundError{path}
	}
	return data, nil
}

func (f *fakeFS) WriteFile(path string, data []byte) error {
	f.files[path] = data
	return nil
}

func (f *fakeFS) Remove(path string) error {
	delete(f.files, path)
	return nil
}

type notFoundError struct{ path string }

func (e *notFoundError) Error() string { return "not found: " + e.path }

func TestStoreSaveLoadRoundTrip(t *testing.T) {
	fs := newFakeFS()
	store := NewStore("/cfg/oblikovati", fs)

	lib := NewLibrary(nil, "Dark")
	custom, _ := lib.Duplicate("Dark", "My Dark")
	lib.EditActiveColor(types.TokenChromeAccent, mustParse(t, "#ff7a45ff"))

	if err := store.SaveTheme(custom); err != nil {
		t.Fatalf("SaveTheme: %v", err)
	}
	if err := store.SaveActive(lib.ActiveName()); err != nil {
		t.Fatalf("SaveActive: %v", err)
	}

	customs, active, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if active != "My Dark" {
		t.Errorf("active = %q, want \"My Dark\"", active)
	}
	if len(customs) != 1 {
		t.Fatalf("loaded %d customs, want 1", len(customs))
	}
	got := customs[0]
	if got.Name() != "My Dark" || got.Kind() != KindCustom {
		t.Errorf("loaded theme = (%q,%q), want (\"My Dark\",custom)", got.Name(), got.Kind())
	}
	if got.Color(types.TokenChromeAccent) != mustParse(t, "#ff7a45ff") {
		t.Errorf("accent not round-tripped: got %v", got.Color(types.TokenChromeAccent).Hex())
	}
}

func TestStoreRefusesToSaveBuiltin(t *testing.T) {
	store := NewStore("/cfg/oblikovati", newFakeFS())
	if err := store.SaveTheme(DefaultDark()); err == nil {
		t.Error("SaveTheme(built-in) should fail")
	}
}

func TestStoreLoadEmptyOnFirstRun(t *testing.T) {
	store := NewStore("/cfg/oblikovati", newFakeFS())
	customs, active, err := store.Load()
	if err != nil {
		t.Fatalf("Load on empty store: %v", err)
	}
	if len(customs) != 0 || active != "" {
		t.Errorf("empty store: customs=%d active=%q, want 0 and \"\"", len(customs), active)
	}
}

func TestStoreRemoveTheme(t *testing.T) {
	fs := newFakeFS()
	store := NewStore("/cfg/oblikovati", fs)
	lib := NewLibrary(nil, "Dark")
	custom, _ := lib.Duplicate("Dark", "Temp")
	_ = store.SaveTheme(custom)

	if err := store.RemoveTheme("Temp"); err != nil {
		t.Fatalf("RemoveTheme: %v", err)
	}
	customs, _, _ := store.Load()
	if len(customs) != 0 {
		t.Errorf("after remove, loaded %d customs, want 0", len(customs))
	}
}

func TestStoreThemeFileIsReadableYAML(t *testing.T) {
	fs := newFakeFS()
	store := NewStore("/cfg/oblikovati", fs)
	custom, _ := NewLibrary(nil, "Dark").Duplicate("Dark", "My Dark")
	_ = store.SaveTheme(custom)
	for path, data := range fs.files {
		if strings.HasSuffix(path, ".yaml") {
			if !strings.Contains(string(data), "name: My Dark") {
				t.Errorf("theme file %q not human-readable YAML:\n%s", path, data)
			}
		}
	}
}

func mustParse(t *testing.T, hex string) Rgba {
	t.Helper()
	c, err := types.ParseHex(hex)
	if err != nil {
		t.Fatalf("ParseHex(%q): %v", hex, err)
	}
	return c
}
