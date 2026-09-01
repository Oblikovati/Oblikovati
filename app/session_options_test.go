// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/app/options"
)

// FakeOptionsStore is a named in-memory options.Store.
type FakeOptionsStore struct {
	stored options.All
	saves  int
}

func (f *FakeOptionsStore) Load() (options.All, error) { return f.stored, nil }
func (f *FakeOptionsStore) Save(all options.All) error {
	f.stored = all
	f.saves++
	return nil
}

func TestUseOptionsStoreAppliesLiveGroups(t *testing.T) {
	t.Parallel()
	stored := options.Defaults()
	stored.Sketch.GridSpacingCm = 2.5
	stored.Sketch.SnapToGrid = false
	stored.Part.ChamferFlatCorners = false
	store := &FakeOptionsStore{stored: stored}

	s := NewSession()
	if err := s.UseOptionsStore(store); err != nil {
		t.Fatalf("UseOptionsStore: %v", err)
	}
	if s.Grid().SpacingModel() != 2.5 || s.Grid().SnapToGrid {
		t.Errorf("grid = spacing %v snap %v, want 2.5cm, no grid snap",
			s.Grid().SpacingModel(), s.Grid().SnapToGrid)
	}
	if s.ChamferFlatCorners() {
		t.Error("chamfer default not applied from store")
	}
}

func TestSetSketchOptionsAppliesAndPersists(t *testing.T) {
	t.Parallel()
	store := &FakeOptionsStore{stored: options.Defaults()}
	s := NewSession()
	if err := s.UseOptionsStore(store); err != nil {
		t.Fatalf("UseOptionsStore: %v", err)
	}

	o := s.Options().Sketch
	o.GridSpacingCm = 0.5
	o.GridMajorEvery = 10
	if err := s.SetSketchOptions(o); err != nil {
		t.Fatalf("SetSketchOptions: %v", err)
	}
	if s.Grid().SpacingModel() != 0.5 || s.Grid().MajorEvery != 10 {
		t.Errorf("grid not applied: spacing %v major %d", s.Grid().SpacingModel(), s.Grid().MajorEvery)
	}
	if store.saves != 1 || store.stored.Sketch.GridSpacingCm != 0.5 {
		t.Errorf("store = %+v after %d saves, want the new sketch group persisted", store.stored.Sketch, store.saves)
	}

	o.GridSpacingCm = -1
	if err := s.SetSketchOptions(o); err == nil {
		t.Error("a non-positive spacing should be rejected")
	}
}

func TestPersistLiveOptionsSnapshotsUIEdits(t *testing.T) {
	t.Parallel()
	store := &FakeOptionsStore{stored: options.Defaults()}
	s := NewSession()
	if err := s.UseOptionsStore(store); err != nil {
		t.Fatalf("UseOptionsStore: %v", err)
	}

	// The Preferences tabs mutate live state directly, then persist.
	s.Grid().Visible = false
	s.SetChamferFlatCorners(false)
	if err := s.PersistLiveOptions(); err != nil {
		t.Fatalf("PersistLiveOptions: %v", err)
	}
	if store.stored.Sketch.GridVisible || store.stored.Part.ChamferFlatCorners {
		t.Errorf("stored = %+v, want the live edits captured", store.stored)
	}
}

func TestSetGeneralOptionsStoresStartupAction(t *testing.T) {
	t.Parallel()
	store := &FakeOptionsStore{stored: options.Defaults()}
	s := NewSession()
	if err := s.UseOptionsStore(store); err != nil {
		t.Fatalf("UseOptionsStore: %v", err)
	}
	if err := s.SetGeneralOptions(options.General{StartupAction: types.StartupEmptyWorkspace}); err != nil {
		t.Fatalf("SetGeneralOptions: %v", err)
	}
	if store.stored.General.StartupAction != types.StartupEmptyWorkspace {
		t.Errorf("stored startup = %v, want empty workspace", store.stored.General.StartupAction)
	}
	if s.Options().General.StartupAction != types.StartupEmptyWorkspace {
		t.Error("live options not updated")
	}
}
