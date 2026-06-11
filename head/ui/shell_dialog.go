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
	seeded    *app.ShellTool // the tool the fields were seeded from (nil = none)
}{thickness: 1}

// drawShellDialog shows the Shell property panel while the Shell tool is active —
// creating a shell or re-editing a committed one (the same panel serves both).
func drawShellDialog(s *app.Session) {
	sh := s.ActiveShell()
	if sh == nil {
		shellUI.seeded = nil
		return
	}
	if shellUI.seeded != sh {
		shellUI.thickness = float32(sh.Thickness())
		shellUI.seeded = sh
	}
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Shell") {
		title := "Shell"
		if name := sh.EditingName(); name != "" {
			title = name // re-editing a committed shell: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, "")
		if propertySection("Input Geometry") {
			drawPickChipRow("Remove", "shell-faces", countChipText(sh.FaceCount(), "Face", "Select Faces"),
				sh.FaceCount() > 0, "Click the faces to open (they are removed; the rest walls in)", sh.ClearFaces)
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
