//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati/app"
	"oblikovati/head/internal/native"
)

// The Coil flow in the head: while the Coil tool runs, a modeless options window lets
// the user choose the output operation, the helix axis, the pitch and the number of
// revolutions, then OK/Cancel. The picked region is outlined by the tool's preview.
var coilUI = struct {
	pitch, revolutions float32
	open               bool
}{pitch: 1, revolutions: 3}

// drawCoilDialog shows the coil options window while the Coil tool is active, syncing
// each control with the tool every frame; OK commits, Cancel aborts.
func drawCoilDialog(s *app.Session) {
	c := s.ActiveCoil()
	if c == nil {
		coilUI.open = false
		return
	}
	refreshCoilUI(c)
	native.SetNextWindowSize(300, 240)
	if native.Begin("Coil") {
		if _, ok := c.PickedProfile(); !ok {
			native.Text("Click a region to coil")
		}
		coilOperationCombo(c)
		coilAxisCombo(c)
		native.Text("Pitch (" + s.LengthUnitName() + ")")
		native.InputFloat("##coil-pitch", &coilUI.pitch)
		c.SetPitch(float64(coilUI.pitch))
		native.Text("Revolutions")
		native.InputFloat("##coil-revolutions", &coilUI.revolutions)
		c.SetRevolutions(float64(coilUI.revolutions))
		drawCoilButtons(s, c)
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

func coilOperationCombo(c *app.CoilTool) {
	preview := "New Solid"
	for _, o := range extrudeOperations {
		if o.op == c.Operation() {
			preview = o.label
		}
	}
	if native.BeginCombo("Output", preview) {
		for _, o := range extrudeOperations {
			if native.Selectable(o.label, o.op == c.Operation()) {
				c.SetOperation(o.op)
			}
		}
		native.EndCombo()
	}
}

func coilAxisCombo(c *app.CoilTool) {
	preview := "Y Axis"
	for _, a := range revolveAxes {
		if a.ref == c.Axis() {
			preview = a.label
		}
	}
	if native.BeginCombo("Axis", preview) {
		for _, a := range revolveAxes {
			if native.Selectable(a.label, a.ref == c.Axis()) {
				c.SetAxis(a.ref)
			}
		}
		native.EndCombo()
	}
}

func drawCoilButtons(s *app.Session, c *app.CoilTool) {
	native.BeginDisabled(!c.CanCommit())
	if native.Button("OK") {
		_ = s.OK()
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}
