// SPDX-License-Identifier: GPL-2.0-only

package keymap

import (
	"path/filepath"
	"testing"
)

func TestFileStoreMissingFileIsEmpty(t *testing.T) {
	s := NewFileStore(filepath.Join(t.TempDir(), "absent.yaml"))
	c, err := s.Load()
	if err != nil {
		t.Fatalf("Load of a missing file: %v", err)
	}
	if len(c.Chords) != 0 || len(c.Aliases) != 0 {
		t.Errorf("missing file should load the empty overlay, got %+v", c)
	}
}

func TestFileStoreRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keymap.yaml")
	s := NewFileStore(path)
	want := Customization{
		Chords:  map[string]string{"Feature.Extrude": "Ctrl+E", "edit.undo": "Ctrl+Z"},
		Aliases: map[string]string{"Feature.Hole": "HOL"},
	}
	if err := s.Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Chords["Feature.Extrude"] != "Ctrl+E" || got.Chords["edit.undo"] != "Ctrl+Z" || got.Aliases["Feature.Hole"] != "HOL" {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, want)
	}
}

func TestMemStoreSavesACopy(t *testing.T) {
	s := NewMemStore()
	c := Customization{Chords: map[string]string{"a": "A"}}
	if err := s.Save(c); err != nil {
		t.Fatalf("Save: %v", err)
	}
	c.Chords["a"] = "MUTATED" // mutating the caller's map must not affect the store
	got, _ := s.Load()
	if got.Chords["a"] != "A" {
		t.Errorf("MemStore should hold a copy, got %q after caller mutation", got.Chords["a"])
	}
	if s.Saved != 1 {
		t.Errorf("Saved = %d, want 1", s.Saved)
	}
}

func TestCloneIsIndependent(t *testing.T) {
	c := Customization{Chords: map[string]string{"a": "A"}, Aliases: map[string]string{"b": "B"}}
	clone := c.Clone()
	clone.Chords["a"] = "X"
	clone.Aliases["b"] = "Y"
	if c.Chords["a"] != "A" || c.Aliases["b"] != "B" {
		t.Error("Clone should not alias the source maps")
	}
}

func TestCloneOfEmptyIsNil(t *testing.T) {
	clone := Defaults().Clone()
	if clone.Chords != nil || clone.Aliases != nil {
		t.Errorf("clone of empty should keep nil maps (omitempty), got %+v", clone)
	}
}
