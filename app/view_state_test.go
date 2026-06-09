// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/persistence/viewstate"
)

// TestViewStateSavedAndRestoredViaStore checks that the per-user view-state store captures
// a document's cameras/layout on save and restores them — so view state lives outside the
// .obk yet survives reopen.
func TestViewStateSavedAndRestoredViaStore(t *testing.T) {
	store := viewstate.NewMemStore()

	// Save: a document with two views, a quad layout, and a moved camera.
	s1 := NewSession()
	s1.SetViewStateStore(store)
	a := addPart(t, s1, "model.obk")
	activate(t, s1, a)
	if err := s1.SetViewLayout(types.LayoutFour); err != nil { // creates the 4 views
		t.Fatalf("SetViewLayout: %v", err)
	}
	_ = s1.ActivateView(0)
	cam := s1.Camera()
	cam.Eye = math.P3(42, 7, 9)
	s1.SetCamera(cam)
	s1.saveViewState(a) // simulate the on-save hook (key = a's path)

	if _, ok, _ := store.Load("model.obk"); !ok {
		t.Fatal("store has no entry after save")
	}

	// Restore into a fresh session/document with the same path.
	s2 := NewSession()
	s2.SetViewStateStore(store)
	b := addPart(t, s2, "model.obk")
	activate(t, s2, b)
	s2.loadViewState(b)

	if b.Views().Count() != 4 || b.Views().Layout() != types.LayoutFour {
		t.Fatalf("restored views=%d layout=%v, want 4 / four", b.Views().Count(), b.Views().Layout())
	}
	if got := b.Views().All()[0].Eye; got != math.P3(42, 7, 9) {
		t.Errorf("restored view0 eye = %v, want (42,7,9)", got)
	}
}

// TestViewStateNoStoreIsNoOp ensures save/load are safe when no store is installed.
func TestViewStateNoStoreIsNoOp(t *testing.T) {
	s := NewSession()
	a := addPart(t, s, "x.obk")
	activate(t, s, a)
	s.saveViewState(a) // must not panic without a store
	s.loadViewState(a)
}
