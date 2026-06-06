//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"strconv"

	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Shell flow in the head: while the Shell tool runs, a modeless options window shows
// the picked removed-face count and the wall thickness (database units), then OK/Cancel.
var shellUI = struct {
	thickness float32
	open      bool
}{thickness: 1}

// drawShellDialog shows the shell options window while the Shell tool is active.
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
	native.SetNextWindowSize(300, 160)
	if native.Begin("Shell") {
		native.Text("Removed faces: " + strconv.Itoa(len(sh.Faces())) + " (click faces to open)")
		native.Text("Thickness (" + s.LengthUnitName() + ")")
		native.InputFloat("##shell-thickness", &shellUI.thickness)
		sh.SetThickness(float64(shellUI.thickness))
		drawCommitCancelButtons(s, sh.CanCommit())
	}
	native.End()
}
