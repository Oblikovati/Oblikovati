//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"github.com/Oblikovati/oblikovati/app"
	"github.com/Oblikovati/oblikovati/head/internal/native"
)

// drawSketch3DSettings shows a small settings window while a 3D sketch is being edited
// (Inventor's 3D Sketch settings): sketch visibility, dimension display, and deferred
// updates. It edits the active sketch directly through its accessors (the same properties
// the sketch3d.setProperty wire method toggles).
func drawSketch3DSettings(s *app.Session) {
	sk := s.ActiveSketch3D()
	if sk == nil {
		return
	}
	if native.Begin("3D Sketch Settings") {
		visible := sk.Visible()
		if native.Checkbox("Visible", &visible) {
			sk.SetVisible(visible)
		}
		dims := sk.DimensionsVisible()
		if native.Checkbox("Show dimensions", &dims) {
			sk.SetDimensionsVisible(dims)
		}
		defer3d := sk.DeferUpdates()
		if native.Checkbox("Defer updates", &defer3d) {
			sk.SetDeferUpdates(defer3d)
		}
	}
	native.End()
}
