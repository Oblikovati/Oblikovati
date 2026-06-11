//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Fillet flow in the head: while the Fillet tool runs, a modeless property panel
// (the reference panel schema) shows the picked-edges chip and the blend radius, then
// OK/Cancel.
var filletUI = struct {
	radius float32
	open   bool
}{radius: 1}

// drawFilletDialog shows the Fillet property panel while the Fillet tool is active.
func drawFilletDialog(s *app.Session) {
	f := s.ActiveFillet()
	if f == nil {
		filletUI.open = false
		return
	}
	if !filletUI.open {
		filletUI.radius = float32(f.Radius())
		filletUI.open = true
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Fillet") {
		drawFeatureBreadcrumb("Fillet", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Edges", "fillet-edges", countChipText(len(f.Edges()), "Edge", "Select Edges"),
				len(f.Edges()) > 0, "Click convex edges in the viewport to round", f.ClearEdges)
		}
		if propertySection("Behavior") {
			propertyFloatRow("Radius", "fillet-radius", s.LengthUnitName(), &filletUI.radius)
			f.SetRadius(float64(filletUI.radius))
		}
		native.Separator()
		drawCommitCancelButtons(s, f.CanCommit())
	}
	native.End()
}
