//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// The Coil flow in the head: while the Coil tool runs, a modeless property panel (the
// reference panel schema) drives the tool — the profile chip, the helix axis, the
// pitch and revolutions, and the boolean output — then OK/Cancel. The picked region is
// outlined by the tool's preview.
var coilUI = struct {
	pitch, revolutions float32
	open               bool
}{pitch: 1, revolutions: 3}

// drawCoilDialog shows the Coil property panel while the Coil tool is active, syncing
// every control with the tool each frame; OK commits, Cancel aborts.
func drawCoilDialog(s *app.Session) {
	c := s.ActiveCoil()
	if c == nil {
		coilUI.open = false
		return
	}
	refreshCoilUI(c)
	native.SetNextWindowSizeOnce(340, 340)
	if native.Begin("Coil") {
		drawFeatureBreadcrumb("Coil", c.SourceSketchName())
		drawCoilInputGeometry(c)
		drawCoilBehavior(s, c)
		drawCoilOutput(c)
		native.Separator()
		drawCommitCancelButtons(s, c.CanCommit())
	}
	native.End()
}

func refreshCoilUI(c *app.CoilTool) {
	if coilUI.open {
		return
	}
	coilUI.pitch = float32(c.Pitch())
	coilUI.revolutions = float32(c.Revolutions())
	coilUI.open = true
}

// drawCoilInputGeometry is the Input Geometry section: the required Profiles chip and
// the helix-axis combo.
func drawCoilInputGeometry(c *app.CoilTool) {
	if !propertySection("Input Geometry") {
		return
	}
	_, picked := c.PickedProfile()
	drawPickChipRow("Profiles", "coil-profiles", pickChipText(picked, "1 Profile", "Select Profile"),
		picked, "Click a region in the viewport to coil", c.ClearProfile)
	propertyRow("Axis")
	native.SetNextItemWidth(propertyFieldWidth)
	coilAxisCombo(c)
}

func coilAxisCombo(c *app.CoilTool) {
	preview := "Y Axis"
	for _, a := range revolveAxes {
		if a.ref == c.Axis() {
			preview = a.label
		}
	}
	if native.BeginCombo("##coil-axis", preview) {
		for _, a := range revolveAxes {
			if native.Selectable(a.label, a.ref == c.Axis()) {
				c.SetAxis(a.ref)
			}
		}
		native.EndCombo()
	}
}

// drawCoilBehavior is the Behavior section: the helix pitch and revolutions.
func drawCoilBehavior(s *app.Session, c *app.CoilTool) {
	if !propertySection("Behavior") {
		return
	}
	propertyFloatRow("Pitch", "coil-pitch", s.LengthUnitName(), &coilUI.pitch)
	c.SetPitch(float64(coilUI.pitch))
	propertyFloatRow("Revolutions", "coil-revolutions", "", &coilUI.revolutions)
	c.SetRevolutions(float64(coilUI.revolutions))
}

// drawCoilOutput is the Output section: the shared Boolean toggle row.
func drawCoilOutput(c *app.CoilTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("coil-boolean", c.Operation(), c.SetOperation)
}
