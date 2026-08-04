//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/scene"

// Overlay widgets (dimension labels, client-graphics labels and billboards) are positioned with
// SetCursorPos at a PROJECTED coordinate. renderer.Project only reports whether a point is in
// front of the camera — it says nothing about whether it landed on screen — so an overlay
// belonging to off-screen geometry was placed far outside the panel.
//
// ImGui grows a window's content rect to contain whatever is drawn in it, so one label at
// x=+5000 made the Viewport window scrollable. It then showed a scrollbar and took ownership of
// the mouse wheel, which stopped zooming the camera and started scrolling the window instead,
// stranding the viewport image outside the visible area. That is #2027, reported as "when zoomed
// in too much a vertical scroll bar appears and it starts to scroll the viewport not zoom".
//
// Culling is also the cheaper path: off-screen labels were being laid out and drawn every frame.

// overlayCullMargin (pixels) keeps an anchor whose glyph or billboard still overlaps the panel
// edge, so nothing pops out at the boundary. It is generous enough for a label's own width.
const overlayCullMargin = 128.0

// onScreen reports whether a projected anchor is close enough to the viewport to be worth
// drawing. cam carries the region's pixel size, which is what the anchor is relative to.
//
//	if !onScreen(sx, sy, cam) { continue } // skip: it would extend the ImGui content rect
func onScreen(sx, sy float64, cam scene.Camera) bool {
	w, h := float64(cam.Width), float64(cam.Height)
	if sx < -overlayCullMargin || sy < -overlayCullMargin {
		return false
	}
	return sx <= w+overlayCullMargin && sy <= h+overlayCullMargin
}
