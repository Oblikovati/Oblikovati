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
	if !displayGroundVisible(s) {
		t.Error("ground plane should be visible by default")
	}
	set := s.DisplaySettings(0)
	set.GroundPlane.Visible = false
	s.SetDisplaySettings(0, set)
	if displayGroundVisible(s) {
		t.Error("ground plane should be hidden after SetDisplaySettings")
	}
	if got := displayGroundColor(s); got == ([4]float32{}) {
		t.Error("ground color should be non-zero")
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
