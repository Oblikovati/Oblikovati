// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/persistence/userprefs"
)

// TestShowCompassPersistsAndLoads checks the compass preference is global: toggling it
// writes through the store, and a fresh session with the same store loads it back.
func TestShowCompassPersistsAndLoads(t *testing.T) {
	t.Parallel()
	store := userprefs.NewMemStore()

	s1 := NewSession()
	s1.SetUserPrefsStore(store)
	if !s1.ShowCompass() {
		t.Fatal("compass should default to shown")
	}
	s1.SetShowCompass(false)

	s2 := NewSession()
	s2.SetUserPrefsStore(store) // loads from the shared store
	if s2.ShowCompass() {
		t.Error("compass preference did not persist across sessions")
	}
}

// TestShowCompassNoStoreIsInSession ensures the toggle works without a store (no panic),
// just not persisted.
func TestShowCompassNoStoreIsInSession(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if !s.ShowCompass() {
		t.Fatal("default shown")
	}
	s.SetShowCompass(false)
	if s.ShowCompass() {
		t.Error("in-session toggle should still apply")
	}
}

// TestNavBarTogglePersists confirms the Navigation Bar show/hide toggle is a global preference
// that survives across sessions (#913 N25).
func TestNavBarTogglePersists(t *testing.T) {
	t.Parallel()
	store := userprefs.NewMemStore()
	s1 := NewSession()
	s1.SetUserPrefsStore(store)
	if !s1.ShowNavBar() {
		t.Fatal("the Navigation Bar should be shown by default")
	}
	s1.SetShowNavBar(false)

	s2 := NewSession()
	s2.SetUserPrefsStore(store)
	if s2.ShowNavBar() {
		t.Error("the Navigation Bar hidden state did not persist")
	}
}

// TestViewCubeTogglesPersist confirms the ViewCube show/hide and Lock-to-Selection toggles
// are now global preferences that survive across sessions.
func TestViewCubeTogglesPersist(t *testing.T) {
	t.Parallel()
	store := userprefs.NewMemStore()

	s1 := NewSession()
	s1.SetUserPrefsStore(store)
	s1.SetShowViewCube(false)
	s1.SetLockToSelection(true)

	s2 := NewSession()
	s2.SetUserPrefsStore(store)
	if s2.ShowViewCube() {
		t.Error("ViewCube show/hide did not persist")
	}
	if !s2.LockToSelection() {
		t.Error("Lock to Selection did not persist")
	}
}
