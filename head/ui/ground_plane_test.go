//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/renderer"
)

// TestDisplayGroundHelpers covers the display-settings ground visibility/color readers
// (M16-F07 #643): default visible, then toggled off, and a custom color round-trips.
func TestDisplayGroundHelpers(t *testing.T) {
	s := framedSession()
	if displayGroundVisible(s) {
		t.Error("ground plane should be hidden by default (#2042)")
	}
	set := s.DisplaySettings(0)
	set.GroundPlane.Visible = true
	s.SetDisplaySettings(0, set)
	if !displayGroundVisible(s) {
		t.Error("ground plane should be visible after SetDisplaySettings")
	}
	if got := displayGroundColor(s); got == ([4]float32{}) {
		t.Error("ground color should be non-zero")
	}
}

// TestWantGroundIsIndependentOfGroundShadows walks the two flags View ▸ Ground Plane and
// View ▸ Ground Shadows drive. Ground Plane alone decides whether the ground is drawn; Ground
// Shadows only decides whether it receives the cast shadow, so it must not veto the draw —
// gating on it made Ground Plane inert on a fresh part, where ground shadows start off (#2042).
func TestWantGroundIsIndependentOfGroundShadows(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		groundVisible, shadows bool
		want                   bool
	}{
		{"visible, shadows off — the fresh-part case", true, false, true},
		{"visible, shadows on", true, true, true},
		{"hidden, shadows off", false, false, false},
		{"hidden, shadows on", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := framedSession()
			set := s.DisplaySettings(0)
			set.GroundPlane.Visible = tc.groundVisible
			s.SetDisplaySettings(0, set)
			sh := s.ShadowSettings()
			sh.GroundShadows = tc.shadows
			s.SetShadowSettings(sh)
			if got := wantGround(s); got != tc.want {
				t.Errorf("wantGround = %v, want %v (groundPlane.visible=%v groundShadows=%v)",
					got, tc.want, tc.groundVisible, tc.shadows)
			}
		})
	}
}

// A wireframe style has no shaded surface to draw the ground on, so it stays suppressed however
// the two toggles are set.
func TestWantGroundSuppressedByWireframeStyle(t *testing.T) {
	s := framedSession()
	s.SetVisualStyle(renderer.Wireframe)
	if wantGround(s) {
		t.Error("the ground plane should not be drawn in a style that shades no faces")
	}
}

// TestGroundPlaneItemUsesColor checks the ground quad is built in the given display-settings
// color, sized around the model footprint.
func TestGroundPlaneItemUsesColor(t *testing.T) {
	red := [4]float32{1, 0, 0, 1}
	it := groundPlaneItem([3]float32{0, 0, 0}, [3]float32{2, 1, 2}, renderer.ShadeFlat, red)
	if it.Color != red {
		t.Errorf("ground color = %v, want red", it.Color)
	}
	if it.Primitive != renderer.Triangles || len(it.Indices) != 6 {
		t.Errorf("ground item = %+v, want a 2-triangle quad", it)
	}
}
