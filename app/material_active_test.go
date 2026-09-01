// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestActivePartMaterialIDTracksActiveDocument pins the accessor the Materials UI uses to keep
// its selectors in sync: it must report the ACTIVE part's assignment, so switching documents
// reports a different material (the doc-scoped-selection bug fix).
func TestActivePartMaterialIDTracksActiveDocument(t *testing.T) {
	t.Parallel()
	s := NewSession()
	mats := s.Materials().Materials()
	if len(mats) < 2 {
		t.Skip("material library needs at least two materials for this test")
	}
	matA, matB := mats[0].ID(), mats[1].ID()

	partA, err := s.NewPart()
	if err != nil {
		t.Fatalf("new part A: %v", err)
	}
	if err := s.AssignMaterial("", matA); err != nil {
		t.Fatalf("assign A: %v", err)
	}

	if _, err := s.NewPart(); err != nil { // part B becomes active
		t.Fatalf("new part B: %v", err)
	}
	if err := s.AssignMaterial("", matB); err != nil {
		t.Fatalf("assign B: %v", err)
	}
	if got := s.ActivePartMaterialID(); got != matB {
		t.Errorf("active (B) material = %q, want %q", got, matB)
	}

	if err := s.Workspace().SetActiveDocument(partA); err != nil {
		t.Fatalf("activate A: %v", err)
	}
	if got := s.ActivePartMaterialID(); got != matA {
		t.Errorf("after switching to A, material = %q, want %q (selector would show B's stale value)", got, matA)
	}
}

// TestActivePartAppearanceID covers the appearance accessor (and the no-active-part branch of
// both accessors) the Materials UI uses to keep its selectors synced.
func TestActivePartAppearanceID(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if got := s.ActivePartMaterialID(); got != "" {
		t.Errorf("no-part material = %q, want empty", got)
	}
	if got := s.ActivePartAppearanceID(); got != "" {
		t.Errorf("no-part appearance = %q, want empty", got)
	}
	if _, err := s.NewPart(); err != nil {
		t.Fatalf("new part: %v", err)
	}
	apprs := s.Materials().Appearances()
	if len(apprs) == 0 {
		t.Skip("no appearances in the seeded library")
	}
	id := apprs[0].ID()
	if err := s.AssignAppearance(ScopePart, "", id); err != nil {
		t.Fatalf("AssignAppearance: %v", err)
	}
	if got := s.ActivePartAppearanceID(); got != id {
		t.Errorf("active part appearance = %q, want %q", got, id)
	}
}
