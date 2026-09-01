// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// TestResolveOverlayMeshTessellatesBody checks the body-overlay resolver (M16-F05 #641) finds a
// body by its reference key among the visible bodies and returns a non-empty tessellated mesh.
func TestResolveOverlayMeshTessellatesBody(t *testing.T) {
	t.Parallel()
	s, _ := extrudedBoxPart(t)
	bodies := s.VisibleBodies()
	if len(bodies) == 0 {
		t.Fatal("expected the extruded box body")
	}
	key := string(bodies[0].ReferenceKey())
	pos, _, idx, ok := s.resolveOverlayMesh(key, 0)
	if !ok || len(pos) == 0 || len(idx) == 0 {
		t.Fatalf("resolveOverlayMesh(%q) = (pos %d, idx %d, ok %v), want a mesh", key, len(pos), len(idx), ok)
	}
	if len(idx)%3 != 0 {
		t.Errorf("index count %d is not a multiple of 3 (triangles)", len(idx))
	}
}

// TestResolveOverlayMeshUnknownKey checks an unknown / empty key resolves to nothing.
func TestResolveOverlayMeshUnknownKey(t *testing.T) {
	t.Parallel()
	s, _ := extrudedBoxPart(t)
	if _, _, _, ok := s.resolveOverlayMesh("no-such-body", 0); ok {
		t.Error("an unknown body key should not resolve")
	}
	if _, _, _, ok := s.resolveOverlayMesh("", 0); ok {
		t.Error("an empty key should not resolve")
	}
}

// TestVisualPanelToggles covers the named-views / color-styles panel open+close setters.
func TestVisualPanelToggles(t *testing.T) {
	t.Parallel()
	s := NewSession()
	s.OpenNamedViewsPanel()
	s.CloseNamedViewsPanel()
	if s.NamedViewsPanelOpen() {
		t.Error("named-views panel should be closed")
	}
	s.OpenColorStylesPanel()
	s.CloseColorStylesPanel()
	if s.ColorStylesPanelOpen() {
		t.Error("color-styles panel should be closed")
	}
}
