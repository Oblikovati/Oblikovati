// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/material"
)

// TestAssignOpenPBRAppearancePartScope mirrors TestPartAppearanceResolvesThroughLiveAssignPath
// for the separate OpenPBRAppearance chain (M45-F05 PBI-350, ADR-0053): assigning the
// built-in default at part scope succeeds and is readable back from the assignment
// store. TestSurfaceLookupResolvesAssignedOpenPBRAppearance below covers render-time
// resolution (SurfaceLookup consuming the assignment, #2150) — this test only proves
// the assignment machinery itself.
func TestAssignOpenPBRAppearancePartScope(t *testing.T) {
	s, def := extrudedBoxPart(t)
	if err := s.AssignOpenPBRAppearance(ScopePart, "", material.DefaultOpenPBRAppearanceID); err != nil {
		t.Fatalf("AssignOpenPBRAppearance: %v", err)
	}
	if got := def.Assignments().PartOpenPBRAppearance(); got != material.DefaultOpenPBRAppearanceID {
		t.Errorf("PartOpenPBRAppearance() = %q, want %q", got, material.DefaultOpenPBRAppearanceID)
	}
}

// TestAssignOpenPBRAppearanceBodyAndFaceScope covers the body/face scopes, mirroring
// the legacy AssignAppearance's scope switch.
func TestAssignOpenPBRAppearanceBodyAndFaceScope(t *testing.T) {
	s, def := extrudedBoxPart(t)
	if err := s.AssignOpenPBRAppearance(ScopeBody, "bodykey", material.DefaultOpenPBRAppearanceID); err != nil {
		t.Fatalf("AssignOpenPBRAppearance(body): %v", err)
	}
	if err := s.AssignOpenPBRAppearance(ScopeFace, "facekey", material.DefaultOpenPBRAppearanceID); err != nil {
		t.Fatalf("AssignOpenPBRAppearance(face): %v", err)
	}
	if got := def.Assignments().BodyOpenPBRAppearances()["bodykey"]; got != material.DefaultOpenPBRAppearanceID {
		t.Errorf("BodyOpenPBRAppearances()[bodykey] = %q, want %q", got, material.DefaultOpenPBRAppearanceID)
	}
	if got := def.Assignments().FaceOpenPBRAppearances()["facekey"]; got != material.DefaultOpenPBRAppearanceID {
		t.Errorf("FaceOpenPBRAppearances()[facekey] = %q, want %q", got, material.DefaultOpenPBRAppearanceID)
	}
}

// TestAssignOpenPBRAppearanceEmbedsNonBuiltinCopy proves the document-embedding half of
// AssignOpenPBRAppearance: assigning a project-scoped (non-built-in) appearance copies
// it into the document's own asset set, exactly as AssignAppearance already does for
// the legacy chain — the .obk stays portable even without the project library.
func TestAssignOpenPBRAppearanceEmbedsNonBuiltinCopy(t *testing.T) {
	s, def := extrudedBoxPart(t)
	custom, err := s.DuplicateOpenPBRAppearance(material.DefaultOpenPBRAppearanceID, "My Copper")
	if err != nil {
		t.Fatalf("DuplicateOpenPBRAppearance: %v", err)
	}
	if err := s.AssignOpenPBRAppearance(ScopePart, "", custom.ID()); err != nil {
		t.Fatalf("AssignOpenPBRAppearance: %v", err)
	}
	embedded, ok := def.Assets().OpenPBRAppearance(custom.ID())
	if !ok {
		t.Fatal("assigning a non-built-in OpenPBR appearance did not embed a document copy")
	}
	if embedded.Source() != material.SourceDocument {
		t.Errorf("embedded copy source = %q, want document", embedded.Source())
	}
}

// TestAssignOpenPBRAppearanceRejectsUnknownIDAndScope mirrors AssignAppearance's error
// handling: an unknown appearance id or an unrecognized scope must error, not silently
// no-op or panic.
func TestAssignOpenPBRAppearanceRejectsUnknownIDAndScope(t *testing.T) {
	s, _ := extrudedBoxPart(t)
	if err := s.AssignOpenPBRAppearance(ScopePart, "", "does-not-exist"); err == nil {
		t.Error("AssignOpenPBRAppearance with an unknown id did not error")
	}
	if err := s.AssignOpenPBRAppearance("nonsense-scope", "", material.DefaultOpenPBRAppearanceID); err == nil {
		t.Error("AssignOpenPBRAppearance with an unknown scope did not error")
	}
}

// TestSurfaceLookupResolvesAssignedOpenPBRAppearance is the render-wiring half PBI-350's
// own doc comment deferred (#2150, found live-testing PBI-354): assigning an
// OpenPBRAppearance must change what SurfaceLookup — and so BOTH the raster and
// Realistic-mode renderers — actually draws, not just what the assignment store
// records. Confirms the resolved renderer.Surface carries the assigned appearance's
// (gamma-encoded, since renderer.Surface.Albedo is sRGB per appearanceSurface's own
// convention) base color/metalness — not the part's unassigned/legacy default.
func TestSurfaceLookupResolvesAssignedOpenPBRAppearance(t *testing.T) {
	s, def := extrudedBoxPart(t)
	body := def.SurfaceBodies().Item(0)

	custom, err := s.DuplicateOpenPBRAppearance(material.DefaultOpenPBRAppearanceID, "Test Red Metal")
	if err != nil {
		t.Fatalf("DuplicateOpenPBRAppearance: %v", err)
	}
	spec := custom.Spec()
	spec.Base.Color = material.Color3{R: 0.6, G: 0.1, B: 0.1}
	spec.Base.Metalness = 1
	s.UpdateOpenPBRAppearance(custom.ID(), spec)
	if err := s.AssignOpenPBRAppearance(ScopePart, "", custom.ID()); err != nil {
		t.Fatalf("AssignOpenPBRAppearance: %v", err)
	}

	lookup := s.SurfaceLookup()
	if lookup == nil {
		t.Fatal("SurfaceLookup nil with an active part")
	}
	surf := lookup(body)
	if surf.Metallic != 1 {
		t.Errorf("Metallic = %v, want 1 (from the assigned OpenPBR appearance, not the legacy default)", surf.Metallic)
	}
	wantR := encodeSRGBChannel(0.6)
	if diff := surf.Albedo[0] - wantR; diff > 1e-4 || diff < -1e-4 {
		t.Errorf("Albedo.R = %v, want sRGB-encoded %v (linear 0.6)", surf.Albedo[0], wantR)
	}
}
