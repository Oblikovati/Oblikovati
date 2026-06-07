// SPDX-License-Identifier: GPL-2.0-only

package viewstate

import (
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTripAndPreservesOtherDocs(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "view-state.yaml"))

	a := ViewState{
		Views:  []ViewFrame{{Name: "Front", Eye: [3]float64{0, 0, 10}, Up: [3]float64{0, 1, 0}, FOV: 0.8}},
		Active: 0, Layout: 4, SplitX: 0.3, SplitY: 0.6,
	}
	if err := store.Save("/a.obk", a); err != nil {
		t.Fatalf("Save a: %v", err)
	}
	if err := store.Save("/b.obk", ViewState{Layout: 1}); err != nil {
		t.Fatalf("Save b: %v", err)
	}

	got, ok, err := store.Load("/a.obk")
	if err != nil || !ok {
		t.Fatalf("Load a: ok=%v err=%v", ok, err)
	}
	if got.Layout != 4 || got.SplitX != 0.3 || len(got.Views) != 1 || got.Views[0].Name != "Front" {
		t.Errorf("loaded a = %+v, want layout 4 splitX 0.3 one view 'Front'", got)
	}
	// Saving a did not clobber b.
	if b, ok, _ := store.Load("/b.obk"); !ok || b.Layout != 1 {
		t.Errorf("b not preserved: ok=%v layout=%d", ok, b.Layout)
	}
}

func TestFileStoreMissingIsNotFound(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "nope.yaml"))
	if _, ok, err := store.Load("/x.obk"); ok || err != nil {
		t.Errorf("missing file: ok=%v err=%v, want not-found, no error", ok, err)
	}
}
