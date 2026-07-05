//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/sketch"
)

// drawSketch3DSettings shows a small settings panel while a 3D sketch is being edited:
// sketch visibility, dimension display, and deferred updates, on the property-panel
// schema (a breadcrumb + a Behavior section of toggle rows). It edits the active sketch
// directly through its accessors (the same properties the sketch3d.setProperty wire
// method toggles); the settings are live, so there is no OK/Cancel.
func drawSketch3DSettings(s *app.Session) {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return
	}
	dialogSizeOnce(300, 190)
	if native.Begin("3D Sketch Settings") {
		drawFeatureBreadcrumb("3D Sketch", sk.Name())
		if propertySection("Behavior") {
			drawSketch3DToggles(sk)
		}
	}
	native.End()
}

// drawSketch3DToggles renders the three live setting rows.
func drawSketch3DToggles(sk *sketch.Sketch3D) {
	propertyRow("")
	visible := sk.Visible()
	if native.Checkbox("Visible", &visible) {
		sk.SetVisible(visible)
	}
	propertyRow("")
	dims := sk.DimensionsVisible()
	if native.Checkbox("Show dimensions", &dims) {
		sk.SetDimensionsVisible(dims)
	}
	propertyRow("")
	defer3d := sk.DeferUpdates()
	if native.Checkbox("Defer updates", &defer3d) {
		sk.SetDeferUpdates(defer3d)
	}
}
