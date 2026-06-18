//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Face Fillet flow in the head (#694): while the Face Fillet tool runs, a modeless property
// panel shows two face-set selectors (clicking one arms it as the target for viewport picks) and
// the blend radius, then OK/Cancel. It rounds every edge shared between the two face sets.
var faceFilletUI = struct {
	radius float32
	seeded *app.FaceFilletTool // the tool the fields were seeded from (nil = none)
}{radius: 1}

// drawFaceFilletDialog shows the Face Fillet property panel while the tool is active — creating a
// face fillet or re-editing a committed one (the same panel serves both).
func drawFaceFilletDialog(s *app.Session) {
	f := s.ActiveFaceFillet()
	if f == nil {
		faceFilletUI.seeded = nil
		return
	}
	if faceFilletUI.seeded != f {
		faceFilletUI.radius = float32(f.Radius())
		faceFilletUI.seeded = f
	}
	native.SetNextWindowSizeOnce(340, 240)
	if native.Begin("Face Fillet") {
		drawFaceFilletPanelBody(s, f)
	}
	native.End()
}

// drawFaceFilletPanelBody draws the panel's sections (the Begin/End wrapper stays in
// drawFaceFilletDialog): the breadcrumb, the two face-set pickers, and the radius row.
func drawFaceFilletPanelBody(s *app.Session, f *app.FaceFilletTool) {
	title := "Face Fillet"
	if name := f.EditingName(); name != "" {
		title = name // re-editing a committed face fillet: the breadcrumb names it
	}
	drawFeatureBreadcrumb(title, "")
	if propertySection("Input Geometry") {
		drawFaceSetRow("Face Set 1", "facefillet-a", f.CountA(), f.ActiveSet() == 0, f.ArmSetA, f.ClearSetA)
		drawFaceSetRow("Face Set 2", "facefillet-b", f.CountB(), f.ActiveSet() == 1, f.ArmSetB, f.ClearSetB)
	}
	if propertySection("Behavior") {
		lengthCmRow(s, "Radius", "facefillet-radius", &faceFilletUI.radius)
		f.SetRadius(float64(faceFilletUI.radius))
	}
	native.Separator()
	drawCommitCancelButtons(s, f.CanCommit())
}

// drawFaceSetRow draws one face-set selector: a count chip that arms the set as the pick target
// when clicked (accent while it is the active set), and a clear (×). The picking itself happens in
// the viewport — this row shows each set's count and switches which set the next pick extends.
func drawFaceSetRow(label, id string, count int, armed bool, onArm, onClear func()) {
	propertyRow(label)
	if armed {
		native.PushStyleColor("Button", accentColor)
		native.PushStyleColor("ButtonHovered", accentColor)
		native.PushStyleColor("ButtonActive", accentColor)
	}
	if native.Button(countChipText(count, "Face", "Select Faces") + "##" + id) {
		onArm()
	}
	if armed {
		native.PopStyleColor(3)
	}
	native.SameLine()
	if native.Button("×##" + id + "-clear") {
		onClear()
	}
}
