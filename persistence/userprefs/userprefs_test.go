// SPDX-License-Identifier: GPL-2.0-only

package userprefs

import (
	"path/filepath"
	"testing"
)

func TestFileStoreRoundTrip(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "preferences.yaml"))
	if err := store.Save(Prefs{CompassHidden: true}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok, err := store.Load()
	if err != nil || !ok {
		t.Fatalf("Load: ok=%v err=%v", ok, err)
	}
	if !got.CompassHidden {
		t.Errorf("loaded CompassHidden = false, want true")
	}
}

func TestFileStoreMissingIsDefault(t *testing.T) {
	store := NewFileStore(filepath.Join(t.TempDir(), "none.yaml"))
	got, ok, err := store.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if ok {
		t.Error("missing file should report ok=false")
	}
	if got.CompassHidden { // zero value = compass shown
		t.Error("default Prefs should not hide the compass")
	}
}
