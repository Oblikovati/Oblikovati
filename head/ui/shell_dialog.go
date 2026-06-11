//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Shell flow in the head: while the Shell tool runs, a modeless property panel
// (the reference panel schema) shows the removed-faces chip and the wall thickness,
// then OK/Cancel.
var shellUI = struct {
	thickness float32
	open      bool
}{thickness: 1}

// drawShellDialog shows the Shell property panel while the Shell tool is active.
func drawShellDialog(s *app.Session) {
	sh := s.ActiveShell()
	if sh == nil {
		shellUI.open = false
		return
	}
	if !shellUI.open {
		shellUI.thickness = float32(sh.Thickness())
		shellUI.open = true
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Shell") {
		drawFeatureBreadcrumb("Shell", "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Remove", "shell-faces", countChipText(len(sh.Faces()), "Face", "Select Faces"),
				len(sh.Faces()) > 0, "Click the faces to open (they are removed; the rest walls in)", sh.ClearFaces)
		}
		if propertySection("Behavior") {
			propertyFloatRow("Thickness", "shell-thickness", s.LengthUnitName(), &shellUI.thickness)
			sh.SetThickness(float64(shellUI.thickness))
		}
		native.Separator()
		drawCommitCancelButtons(s, sh.CanCommit())
	}
	native.End()
}
