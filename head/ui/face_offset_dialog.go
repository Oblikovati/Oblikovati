//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Offset Face flow in the head: while the tool runs, a modeless property panel
// (the reference panel schema) shows the picked-faces chip and the signed offset
// distance, then OK/Cancel.
var faceOffsetUI = struct {
	distance float32
	open     bool
}{distance: 1}

// drawFaceOffsetDialog shows the Offset Face property panel while the tool is active.
func drawFaceOffsetDialog(s *app.Session) {
	o := s.ActiveFaceOffset()
	if o == nil {
		faceOffsetUI.open = false
		return
	}
	if !faceOffsetUI.open {
		faceOffsetUI.distance = float32(o.Distance())
		faceOffsetUI.open = true
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Offset Face") {
		drawFeatureBreadcrumb("Offset Face", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Faces", "face-offset-faces", countChipText(len(o.Faces()), "Face", "Select Faces"),
				len(o.Faces()) > 0, "Click faces in the viewport to offset", o.ClearFaces)
		}
		if propertySection("Behavior") {
			lengthCmRowHint(s, "Distance", "face-offset-distance", " (+out / −in)", &faceOffsetUI.distance)
			o.SetDistance(float64(faceOffsetUI.distance))
			if i := propertyComboRow("Approximation", "face-offset-approx", app.ApproximationOptions(), o.ApproximationIndex()); i >= 0 {
				o.SetApproximationIndex(i)
			}
		}
		native.Separator()
		drawCommitCancelButtons(s, o.CanCommit())
	}
	native.End()
}
