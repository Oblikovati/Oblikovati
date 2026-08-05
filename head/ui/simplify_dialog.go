//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Simplify flow in the head: while the tool runs, a modeless property panel shows the
// faces-to-remove chip and the fill-voids toggle, then OK/Cancel.

// drawSimplifyDialog shows the Simplify property panel while the tool is active.
func drawSimplifyDialog(s *app.Session) {
	sp := s.ActiveSimplify()
	if sp == nil {
		return
	}
	dialogSizeOnce(340, 210)
	if native.Begin("Simplify") {
		drawFeatureBreadcrumb("Simplify", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Remove", "simplify-faces", countChipText(sp.FaceCount(), "Face", "Select Faces"),
				sp.FaceCount() > 0, "Click the faces to remove; the openings heal", sp.ClearFaces)
		}
		if propertySection("Behavior") {
			propertyRow("")
			fill := sp.FillVoids()
			if native.Checkbox("Fill internal voids", &fill) {
				sp.SetFillVoids(fill)
			}
		}
		native.Separator()
		drawCommitCancelButtons(s, sp.CanCommit())
	}
	native.End()
}
