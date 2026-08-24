// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/material"
)

// TestAssignOpenPBRAppearancePartScope mirrors TestPartAppearanceResolvesThroughLiveAssignPath
// for the separate OpenPBRAppearance chain (M45-F05 PBI-350, ADR-0053): assigning the
// built-in default at part scope succeeds and is readable back from the assignment
// store. Render-time resolution (SurfaceLookup consuming an OpenPBRAppearance) is out
// of this PBI's scope — see realistic_render.go's doc comment — so this only proves the
// assignment machinery itself.
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
