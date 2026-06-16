// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/model/style"
)

// TestAssignColorStyleToBody checks a body's color-style assignment round-trips and that an
// unknown style is rejected (M16-F02 #403/#408).
func TestAssignColorStyleToBody(t *testing.T) {
	s := NewSession()
	if err := s.AssignColorStyleToBody("BODYKEY", "Brass"); err != nil {
		t.Fatalf("assign Brass: %v", err)
	}
	if name, ok := s.BodyColorStyle("BODYKEY"); !ok || name != "Brass" {
		t.Errorf("BodyColorStyle = (%q, %v), want (Brass, true)", name, ok)
	}
	if err := s.AssignColorStyleToBody("BODYKEY", "Nope"); err == nil {
		t.Error("assigning an unknown style should error")
	}
	s.ClearBodyColorStyle("BODYKEY")
	if _, ok := s.BodyColorStyle("BODYKEY"); ok {
		t.Error("assignment should be gone after ClearBodyColorStyle")
	}
}

// TestStyleSurfaceUsesDiffuseAlbedo checks the style→surface conversion drives albedo from the
// diffuse color and roughness from shininess.
func TestStyleSurfaceUsesDiffuseAlbedo(t *testing.T) {
	cs := style.ColorStyle{Diffuse: types.NewColor(255, 0, 0), Shininess: 1, Opacity: 1}
	surf := styleSurface(cs)
	if surf.Albedo[0] != 1 || surf.Albedo[1] != 0 {
		t.Errorf("albedo = %v, want red", surf.Albedo)
	}
	if surf.Roughness != 0 {
		t.Errorf("roughness = %v, want 0 (shininess 1)", surf.Roughness)
	}
}
