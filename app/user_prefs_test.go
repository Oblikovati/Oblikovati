// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/persistence/userprefs"
)

// TestShowCompassPersistsAndLoads checks the compass preference is global: toggling it
// writes through the store, and a fresh session with the same store loads it back.
func TestShowCompassPersistsAndLoads(t *testing.T) {
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
	s := NewSession()
	if !s.ShowCompass() {
		t.Fatal("default shown")
	}
	s.SetShowCompass(false)
	if s.ShowCompass() {
		t.Error("in-session toggle should still apply")
	}
}
