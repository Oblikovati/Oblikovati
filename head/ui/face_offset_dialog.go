//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Offset Face flow in the head: while the tool runs, a modeless options window shows
// the picked-face count and the (signed) offset distance, then OK/Cancel.
var faceOffsetUI = struct {
	distance float32
	open     bool
}{distance: 1}

// drawFaceOffsetDialog shows the offset-face options window while the tool is active.
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
	native.SetNextWindowSize(300, 160)
	if native.Begin("Offset Face") {
		native.Text("Faces: " + strconv.Itoa(len(o.Faces())) + " (click faces to offset)")
		native.Text("Distance (" + s.LengthUnitName() + ", +out / −in)")
		native.InputFloat("##face-offset-distance", &faceOffsetUI.distance)
		o.SetDistance(float64(faceOffsetUI.distance))
		drawCommitCancelButtons(s, o.CanCommit())
	}
	native.End()
}
