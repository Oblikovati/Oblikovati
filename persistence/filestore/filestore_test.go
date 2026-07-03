// SPDX-License-Identifier: GPL-2.0-only

package filestore

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// wardrobe is a representative config payload: scalars, a map, and omitempty tags —
// the shapes the six real stores persist.
type wardrobe struct {
	Coats  int               `yaml:"coats,omitempty"`
	Lining string            `yaml:"lining,omitempty"`
	Hooks  map[string]string `yaml:"hooks,omitempty"`
}

func TestFileStoreRoundTrip(t *testing.T) {
	s := New[wardrobe](filepath.Join(t.TempDir(), "wardrobe.yaml"))
	want := wardrobe{Coats: 3, Lining: "silk", Hooks: map[string]string{"left": "hat"}}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, found, err := s.Load()
	if err != nil || !found {
		t.Fatalf("Load: found=%v err=%v", found, err)
	}
	if got.Coats != 3 || got.Lining != "silk" || got.Hooks["left"] != "hat" {
		t.Errorf("round trip = %+v, want %+v", got, want)
	}
}

func TestFileStoreMissingFileIsZeroNotFound(t *testing.T) {
	s := New[wardrobe](filepath.Join(t.TempDir(), "absent.yaml"))
	got, found, err := s.Load()
	if err != nil || found {
		t.Fatalf("Load(missing): found=%v err=%v, want not-found, no error", found, err)
	}
	if got.Coats != 0 || got.Lining != "" || got.Hooks != nil {
		t.Errorf("Load(missing) = %+v, want zero value", got)
	}
}

func TestFileStoreCorruptYAMLNamesThePath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "corrupt.yaml")
	if err := os.WriteFile(path, []byte("coats: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, found, err := New[wardrobe](path).Load()
	if err == nil || found {
		t.Fatalf("Load(corrupt): found=%v err=%v, want an error", found, err)
	}
	// The store names the path with %q, which escapes the backslashes in a Windows path,
	// so the raw path is not a byte-for-byte substring there; compare against the same
	// quoted form (minus its surrounding quotes) so the assertion holds cross-platform.
	quoted := strconv.Quote(path)
	if !strings.Contains(err.Error(), quoted[1:len(quoted)-1]) {
		t.Errorf("corrupt-file error %q does not name the offending path %q", err, path)
	}
}

// TestFileStoreLoadIntoKeepsDefaultsForAbsentKeys pins the defaults-injection seam
// app/options relies on: keys absent from the YAML keep their pre-filled value.
func TestFileStoreLoadIntoKeepsDefaultsForAbsentKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "partial.yaml")
	if err := os.WriteFile(path, []byte("coats: 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	v := wardrobe{Coats: 1, Lining: "wool"}
	found, err := New[wardrobe](path).LoadInto(&v)
	if err != nil || !found {
		t.Fatalf("LoadInto: found=%v err=%v", found, err)
	}
	if v.Coats != 7 || v.Lining != "wool" {
		t.Errorf("LoadInto = %+v, want coats from file, lining kept from defaults", v)
	}
}

// TestFileStoreSaveIsAtomic pins the temp+rename discipline: after a save over an
// existing file the content is fully replaced and no staging file is left behind.
func TestFileStoreSaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	s := New[wardrobe](filepath.Join(dir, "wardrobe.yaml"))
	if err := s.Save(wardrobe{Coats: 1}); err != nil {
		t.Fatalf("first Save: %v", err)
	}
	if err := s.Save(wardrobe{Coats: 2}); err != nil {
		t.Fatalf("second Save: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "wardrobe.yaml" {
		t.Errorf("directory after saves = %v, want only wardrobe.yaml (no temp leftovers)", entries)
	}
	if got, _, _ := s.Load(); got.Coats != 2 {
		t.Errorf("reloaded coats = %d, want 2", got.Coats)
	}
}

// TestFileStoreSaveCreatesConfigDir pins first-use behavior: the config directory is
// created on demand, matching the six stores this replaces.
func TestFileStoreSaveCreatesConfigDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deeper", "w.yaml")
	if err := New[wardrobe](path).Save(wardrobe{Coats: 5}); err != nil {
		t.Fatalf("Save into missing dir: %v", err)
	}
	if got, found, err := New[wardrobe](path).Load(); err != nil || !found || got.Coats != 5 {
		t.Errorf("reload = (%+v, %v, %v), want coats 5", got, found, err)
	}
}

// TestFileStoreGoldenCompatibility pins on-disk stability: a YAML file written by the
// pre-#1651 store code (plain os.WriteFile of yamlcodec output) loads unchanged, and
// the FileStore writes the same bytes back (user configs must survive the refactor).
func TestFileStoreGoldenCompatibility(t *testing.T) {
	golden := "coats: 2\nlining: tweed\nhooks:\n    left: scarf\n"
	path := filepath.Join(t.TempDir(), "golden.yaml")
	if err := os.WriteFile(path, []byte(golden), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New[wardrobe](path)
	v, found, err := s.Load()
	if err != nil || !found || v.Coats != 2 || v.Lining != "tweed" || v.Hooks["left"] != "scarf" {
		t.Fatalf("golden load = (%+v, %v, %v)", v, found, err)
	}
	if err := s.Save(v); err != nil {
		t.Fatalf("re-save: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != golden {
		t.Errorf("re-saved bytes = %q, want the golden %q", raw, golden)
	}
}

func TestMemStoreReportsFoundAfterSave(t *testing.T) {
	s := NewMemStore[wardrobe]()
	if _, found, _ := s.Load(); found {
		t.Error("fresh MemStore should report not-found")
	}
	if err := s.Save(wardrobe{Coats: 4}); err != nil {
		t.Fatal(err)
	}
	got, found, _ := s.Load()
	if !found || got.Coats != 4 || s.Saved != 1 {
		t.Errorf("after Save: got=%+v found=%v Saved=%d, want coats 4, found, 1 save", got, found, s.Saved)
	}
}

func TestKeyedMemStoreIsolatesKeys(t *testing.T) {
	s := NewKeyedMemStore[wardrobe]()
	if err := s.Save("/a.obk", wardrobe{Coats: 1}); err != nil {
		t.Fatal(err)
	}
	if got, found, _ := s.Load("/a.obk"); !found || got.Coats != 1 {
		t.Errorf("Load(a) = (%+v, %v), want coats 1", got, found)
	}
	if _, found, _ := s.Load("/b.obk"); found {
		t.Error("unrelated key should report not-found")
	}
}
