// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/compdef"
)

func TestMaterialsLibrarySeeded(t *testing.T) {
	s := NewSession()
	if len(s.Materials().Materials()) == 0 || len(s.Materials().Appearances()) == 0 {
		t.Fatal("session material library is empty; built-ins should be seeded")
	}
}

func TestSurfaceLookupNilWithoutActivePart(t *testing.T) {
	s := NewSession()
	if s.SurfaceLookup() != nil {
		t.Error("SurfaceLookup should be nil with no active part (renderer uses its default)")
	}
}

func TestSurfaceLookupResolvesAssignedAppearance(t *testing.T) {
	s := NewSession()
	if _, err := compdef.AddPart(s.Workspace(), "p.obk", true); err != nil {
		t.Fatalf("AddPart: %v", err)
	}
	if s.SurfaceLookup() == nil {
		t.Fatal("SurfaceLookup nil with an active part")
	}
	// The mapping from a library appearance to a renderer surface must carry the PBR
	// fields (the steel built-in is metallic).
	steel, _ := s.Materials().Appearance("steel")
	surf := appearanceSurface(steel)
	if surf.Albedo != steel.Albedo().Array() || surf.Metallic != steel.Metallic() {
		t.Errorf("appearanceSurface lost PBR fields: %+v", surf)
	}
}
