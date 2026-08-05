//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Unwrap flow in the head: while the tool runs, a modeless property panel shows the picked
// face chip, then OK/Cancel. There is no parameter — the development is determined by the face.

// drawUnwrapDialog shows the Unwrap property panel while the tool is active.
func drawUnwrapDialog(s *app.Session) {
	u := s.ActiveUnwrap()
	if u == nil {
		return
	}
	dialogSizeOnce(340, 190)
	if native.Begin("Unwrap") {
		drawFeatureBreadcrumb("Unwrap", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Face", "unwrap-face", pickChipText(u.FacePicked(), "1 Face", "Select Face"),
				u.FacePicked(), "Click the cylindrical face to flatten", u.ClearFace)
			native.Text("The flat development is appended as a sheet body.")
		}
		native.Separator()
		drawCommitCancelButtons(s, u.CanCommit())
	}
	native.End()
}
