// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/material"
)

func TestMaterialsLibrarySeeded(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if len(s.Materials().Materials()) == 0 || len(s.Materials().Appearances()) == 0 {
		t.Fatal("session material library is empty; built-ins should be seeded")
	}
}

func TestSurfaceLookupNilWithoutActivePart(t *testing.T) {
	t.Parallel()
	s := NewSession()
	if s.SurfaceLookup() != nil {
		t.Error("SurfaceLookup should be nil with no active part (renderer uses its default)")
	}
}

// TestSurfaceLookupResolvesAssignedAppearance confirms the resolved renderer.Surface
// carries the assigned appearance's (gamma-encoded, since renderer.Surface.Albedo is
// sRGB per appearanceSurface's own convention) base color/metalness — not the part's
// unassigned default.
func TestSurfaceLookupResolvesAssignedAppearance(t *testing.T) {
	t.Parallel()
	s, def := extrudedBoxPart(t)
	body := def.SurfaceBodies().Item(0)

	custom, err := s.DuplicateAppearance(material.DefaultAppearanceID, "Test Red Metal")
	if err != nil {
		t.Fatalf("DuplicateAppearance: %v", err)
	}
	spec := custom.Spec()
	spec.Base.Color = material.Color3{R: 0.6, G: 0.1, B: 0.1}
	spec.Base.Metalness = 1
	s.UpdateAppearance(custom.ID(), spec)
	if err := s.AssignAppearance(ScopePart, "", custom.ID()); err != nil {
		t.Fatalf("AssignAppearance: %v", err)
	}

	lookup := s.SurfaceLookup()
	if lookup == nil {
		t.Fatal("SurfaceLookup nil with an active part")
	}
	surf := lookup(body)
	if surf.Metallic != 1 {
		t.Errorf("Metallic = %v, want 1 (from the assigned appearance, not the default)", surf.Metallic)
	}
	wantR := encodeSRGBChannel(0.6)
	if diff := surf.Albedo[0] - wantR; diff > 1e-4 || diff < -1e-4 {
		t.Errorf("Albedo.R = %v, want sRGB-encoded %v (linear 0.6)", surf.Albedo[0], wantR)
	}
}

// TestAssignAppearancePartScope: assigning at part scope succeeds and is readable back
// from the assignment store.
func TestAssignAppearancePartScope(t *testing.T) {
	t.Parallel()
	s, def := extrudedBoxPart(t)
	if err := s.AssignAppearance(ScopePart, "", material.DefaultAppearanceID); err != nil {
		t.Fatalf("AssignAppearance: %v", err)
	}
	if got := def.Assignments().PartAppearance(); got != material.DefaultAppearanceID {
		t.Errorf("PartAppearance() = %q, want %q", got, material.DefaultAppearanceID)
	}
}

// TestAssignAppearanceBodyAndFaceScope covers the body/face scopes.
func TestAssignAppearanceBodyAndFaceScope(t *testing.T) {
	t.Parallel()
	s, def := extrudedBoxPart(t)
	if err := s.AssignAppearance(ScopeBody, "bodykey", material.DefaultAppearanceID); err != nil {
		t.Fatalf("AssignAppearance(body): %v", err)
	}
	if err := s.AssignAppearance(ScopeFace, "facekey", material.DefaultAppearanceID); err != nil {
		t.Fatalf("AssignAppearance(face): %v", err)
	}
	if got := def.Assignments().BodyAppearances()["bodykey"]; got != material.DefaultAppearanceID {
		t.Errorf("BodyAppearances()[bodykey] = %q, want %q", got, material.DefaultAppearanceID)
	}
	if got := def.Assignments().FaceAppearances()["facekey"]; got != material.DefaultAppearanceID {
		t.Errorf("FaceAppearances()[facekey] = %q, want %q", got, material.DefaultAppearanceID)
	}
}

// TestAssignAppearanceEmbedsNonBuiltinCopy proves the document-embedding half of
// AssignAppearance: assigning a project-scoped (non-built-in) appearance copies it
// into the document's own asset set — the .obk stays portable even without the
// project library.
func TestAssignAppearanceEmbedsNonBuiltinCopy(t *testing.T) {
	t.Parallel()
	s, def := extrudedBoxPart(t)
	custom, err := s.DuplicateAppearance(material.DefaultAppearanceID, "My Copper")
	if err != nil {
		t.Fatalf("DuplicateAppearance: %v", err)
	}
	if err := s.AssignAppearance(ScopePart, "", custom.ID()); err != nil {
		t.Fatalf("AssignAppearance: %v", err)
	}
	embedded, ok := def.Assets().Appearance(custom.ID())
	if !ok {
		t.Fatal("assigning a non-built-in appearance did not embed a document copy")
	}
	if embedded.Source() != material.SourceDocument {
		t.Errorf("embedded copy source = %q, want document", embedded.Source())
	}
}

// TestAssignAppearanceRejectsUnknownIDAndScope: an unknown appearance id or an
// unrecognized scope must error, not silently no-op or panic.
func TestAssignAppearanceRejectsUnknownIDAndScope(t *testing.T) {
	t.Parallel()
	s, _ := extrudedBoxPart(t)
	if err := s.AssignAppearance(ScopePart, "", "does-not-exist"); err == nil {
		t.Error("AssignAppearance with an unknown id did not error")
	}
	if err := s.AssignAppearance("nonsense-scope", "", material.DefaultAppearanceID); err == nil {
		t.Error("AssignAppearance with an unknown scope did not error")
	}
}
