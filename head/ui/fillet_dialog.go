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
	seeded *app.FilletTool // the tool the fields were seeded from (nil = none)
}{radius: 1}

// drawFilletDialog shows the Fillet property panel while the Fillet tool is active —
// creating a fillet or re-editing a committed one (the same panel serves both).
func drawFilletDialog(s *app.Session) {
	f := s.ActiveFillet()
	if f == nil {
		filletUI.seeded = nil
		return
	}
	if filletUI.seeded != f {
		filletUI.radius = float32(f.Radius())
		filletUI.seeded = f
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Fillet") {
		title := "Fillet"
		if name := f.EditingName(); name != "" {
			title = name // re-editing a committed fillet: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Edges", "fillet-edges", countChipText(f.EdgeCount(), "Edge", "Select Edges"),
				f.EdgeCount() > 0, "Click convex edges in the viewport to round", f.ClearEdges)
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
