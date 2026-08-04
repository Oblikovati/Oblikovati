//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/clientgraphics"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/renderer"
	"oblikovati.org/scene"
)

// Add-in client/interaction graphics display (M05-F05): the geometry (meshes, heatmaps,
// lines, point glyphs) rides the normal viewport draw list, like the work-plane and sketch
// overlays; the text labels are 2D, drawn over the rendered image at each anchor's
// projected pixel — the same split the dimension overlay uses.

// clientGraphicsOverlay appends the live client/interaction graphics geometry to list and
// returns the text labels and image billboards to draw after the image is blitted.
func clientGraphicsOverlay(s *app.Session, cam scene.Camera, list renderer.DrawList) (renderer.DrawList, []clientgraphics.Label, []clientgraphics.ImageBillboard) {
	items, labels, images := s.Graphics().Build(cam)
	list.Items = append(list.Items, items...)
	return list, labels, images
}

// drawClientGraphicsLabels overlays each client-graphics label's text at its projected
// world anchor (the image has already been drawn at window-local cx,cy). Anchors behind the
// camera are skipped.
func drawClientGraphicsLabels(cx, cy float32, cam scene.Camera, labels []clientgraphics.Label) {
	for _, l := range labels {
		sx, sy, ok := renderer.Project(cam, viewportNear, viewportFar, l.Anchor)
		if !ok || !onScreen(sx, sy, cam) {
			continue // see onScreen: an off-screen anchor makes the viewport scrollable (#2027)
		}
		native.SetCursorPos(cx+float32(sx), cy+float32(sy))
		native.Text(l.Text)
	}
}
