//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Full Round Fillet flow in the head (#694): while the tool runs, a modeless property panel
// shows three face-set selectors — Side 1, the Center face to round, and Side 2 — then OK/Cancel.
// There is no radius (it is derived as half the side-to-side distance).
var fullRoundUI = struct {
	seeded *app.FullRoundFilletTool // the tool the panel was bound to (nil = none)
}{}

// drawFullRoundFilletDialog shows the Full Round property panel while the tool is active — creating
// a full round or re-editing a committed one (the same panel serves both).
func drawFullRoundFilletDialog(s *app.Session) {
	f := s.ActiveFullRoundFillet()
	if f == nil {
		fullRoundUI.seeded = nil
		return
	}
	fullRoundUI.seeded = f
	native.SetNextWindowSizeOnce(340, 230)
	if native.Begin("Full Round Fillet") {
		drawFullRoundPanelBody(s, f)
	}
	native.End()
}

// drawFullRoundPanelBody draws the panel's sections (the Begin/End wrapper stays in
// drawFullRoundFilletDialog): the breadcrumb and the three face-set pickers.
func drawFullRoundPanelBody(s *app.Session, f *app.FullRoundFilletTool) {
	title := "Full Round Fillet"
	if name := f.EditingName(); name != "" {
		title = name // re-editing a committed full round: the breadcrumb names it
	}
	drawFeatureBreadcrumb(title, "")
	if propertySection("Input Geometry") {
		drawFaceSetRow("Side 1", "fullround-s1", f.Count1(), f.ActiveSet() == 0, f.ArmSide1, f.ClearSide1)
		drawFaceSetRow("Center face", "fullround-c", f.CountCenter(), f.ActiveSet() == 1, f.ArmCenter, f.ClearCenter)
		drawFaceSetRow("Side 2", "fullround-s2", f.Count2(), f.ActiveSet() == 2, f.ArmSide2, f.ClearSide2)
	}
	native.Separator()
	drawCommitCancelButtons(s, f.CanCommit())
}
