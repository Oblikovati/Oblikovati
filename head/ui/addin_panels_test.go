// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/app"
)

// TestPanelBufferReseedsOnDeclaredChange covers the editable-control text buffer: it seeds from
// the declared value, persists a user edit, but re-seeds in place when the add-in pushes a
// different value (populating a form from a loaded document) without clobbering an echoed edit.
func TestPanelBufferReseedsOnDeclaredChange(t *testing.T) {
	delete(panelEditBuffers, "w/poles")
	delete(panelDeclared, "w/poles")

	if got := bufString(panelBuffer("w/poles", "10")); got != "10" {
		t.Fatalf("seed = %q, want 10", got)
	}
	// Same declared value as the buffer already holds: no clobber.
	if got := bufString(panelBuffer("w/poles", "10")); got != "10" {
		t.Errorf("re-set same value = %q, want 10", got)
	}
	// Add-in pushes a different value → re-seed in place.
	if got := bufString(panelBuffer("w/poles", "12")); got != "12" {
		t.Errorf("after add-in pushed 12, buffer = %q, want 12", got)
	}
}

// TestSyncMaterialSelectionFollowsActiveDocument covers the Materials selector resync: switching
// the active document repoints the selectors at that part's assigned material.
func TestSyncMaterialSelectionFollowsActiveDocument(t *testing.T) {
	s := app.NewSession()
	mats := s.Materials().Materials()
	if len(mats) < 2 {
		t.Skip("need two materials")
	}
	matSelectionSynced = false
	selectedMaterialID = ""

	a, err := s.NewPart()
	if err != nil {
		t.Fatalf("new part A: %v", err)
	}
	if err := s.AssignMaterial("", mats[0].ID()); err != nil {
		t.Fatalf("assign A: %v", err)
	}
	syncMaterialSelection(s)
	if selectedMaterialID != mats[0].ID() {
		t.Errorf("after A, selected = %q, want %q", selectedMaterialID, mats[0].ID())
	}

	if _, err := s.NewPart(); err != nil { // part B active
		t.Fatalf("new part B: %v", err)
	}
	if err := s.AssignMaterial("", mats[1].ID()); err != nil {
		t.Fatalf("assign B: %v", err)
	}
	syncMaterialSelection(s)
	if selectedMaterialID != mats[1].ID() {
		t.Errorf("after switching to B, selected = %q, want %q (stale A)", selectedMaterialID, mats[1].ID())
	}
	_ = a
}
