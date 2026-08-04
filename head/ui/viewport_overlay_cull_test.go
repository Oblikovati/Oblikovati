//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"testing"

	"oblikovati.org/scene"
)

// #2027: overlay anchors were placed at unclamped projected coordinates, so one belonging to
// off-screen geometry grew the ImGui content rect, made the Viewport window scrollable, and let
// it swallow the mouse wheel that should have been camera zoom.

func cullCam() scene.Camera { return scene.Camera{Width: 800, Height: 600} }

// TestOnScreenKeepsAnchorsInsideTheViewport: anything actually visible must still draw.
func TestOnScreenKeepsAnchorsInsideTheViewport(t *testing.T) {
	cam := cullCam()
	for _, p := range [][2]float64{{0, 0}, {400, 300}, {799, 599}, {800, 600}} {
		if !onScreen(p[0], p[1], cam) {
			t.Errorf("anchor %v inside an 800x600 viewport was culled", p)
		}
	}
}

// TestOnScreenCullsFarAnchors is the fix: the runaway coordinates that made the window
// scrollable are rejected.
func TestOnScreenCullsFarAnchors(t *testing.T) {
	cam := cullCam()
	for _, p := range [][2]float64{{5000, 300}, {400, 4000}, {-5000, 300}, {400, -900}} {
		if onScreen(p[0], p[1], cam) {
			t.Errorf("anchor %v far outside an 800x600 viewport was kept; it would extend the content rect", p)
		}
	}
}

// TestOnScreenKeepsEdgeStragglers: an anchor just past the edge still has its glyph partly
// inside, so culling must not clip labels at the boundary.
func TestOnScreenKeepsEdgeStragglers(t *testing.T) {
	cam := cullCam()
	if !onScreen(float64(cam.Width)+overlayCullMargin/2, 300, cam) {
		t.Error("an anchor just past the right edge was culled; its label is still partly visible")
	}
	if !onScreen(-overlayCullMargin/2, 300, cam) {
		t.Error("an anchor just past the left edge was culled; its label is still partly visible")
	}
}

// TestOnScreenScalesWithTheViewport: the bound is the camera's region, not a constant, so a
// large viewport does not cull anchors that are genuinely inside it.
func TestOnScreenScalesWithTheViewport(t *testing.T) {
	small, large := scene.Camera{Width: 200, Height: 200}, scene.Camera{Width: 4000, Height: 3000}
	if onScreen(3000, 2000, small) {
		t.Error("an anchor way outside a 200x200 viewport was kept")
	}
	if !onScreen(3000, 2000, large) {
		t.Error("an anchor inside a 4000x3000 viewport was culled")
	}
}
