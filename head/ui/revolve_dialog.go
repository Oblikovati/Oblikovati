//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati/app"
	"oblikovati/head/internal/native"
	"oblikovati/model/feature"
)

// The Revolve flow in the head: while the Revolve tool runs, a modeless options window
// (Inventor's Revolve property panel) lets the user choose the output operation, the
// axis of revolution and the swept angle (or a full revolution), then OK/Cancel. The
// picked region is outlined in the viewport by the tool's preview. The angle is shown
// in degrees.
var revolveUI = struct {
	angleDeg   float32
	centerline bool
	open       bool
}{angleDeg: 360}

// revolveAxes names each origin axis for the axis combo, paired with its work reference.
var revolveAxes = []struct {
	label string
	ref   feature.WorkRef
}{{"X Axis", feature.OriginXAxis}, {"Y Axis", feature.OriginYAxis}, {"Z Axis", feature.OriginZAxis}}

// drawRevolveDialog shows the revolve options window while the Revolve tool is active,
// syncing each control with the tool every frame; OK commits, Cancel aborts.
func drawRevolveDialog(s *app.Session) {
	rv := s.ActiveRevolve()
	if rv == nil {
		revolveUI.open = false
		return
	}
	if !revolveUI.open {
		revolveUI.angleDeg = seedRevolveAngle(rv)
		revolveUI.centerline = rv.UseCenterline()
		revolveUI.open = true
	}
	native.SetNextWindowSize(300, 270)
	if native.Begin("Revolve") {
		drawRevolveProfileText(rv)
		revolveOperationCombo(rv)
		native.Checkbox("About sketch centerline", &revolveUI.centerline)
		rv.SetUseCenterline(revolveUI.centerline)
		native.BeginDisabled(revolveUI.centerline) // the axis is the centerline now
		revolveAxisCombo(rv)
		native.EndDisabled()
		drawRevolveAngle(rv)
		drawRevolveButtons(s, rv)
	}
	native.End()
}

// seedRevolveAngle returns the dialog's initial angle in degrees from the tool (a full
// revolution shows 360).
func seedRevolveAngle(rv *app.RevolveTool) float32 {
	if rv.IsFullRevolution() {
		return 360
	}
	return float32(rv.Angle() * 180 / stdmath.Pi)
}

func drawRevolveProfileText(rv *app.RevolveTool) {
	if _, ok := rv.PickedProfile(); !ok {
		native.Text("Click a region to revolve")
	}
}

func revolveOperationCombo(rv *app.RevolveTool) {
	preview := "New Solid"
	for _, o := range extrudeOperations {
		if o.op == rv.Operation() {
			preview = o.label
		}
	}
	if native.BeginCombo("Output", preview) {
		for _, o := range extrudeOperations {
			if native.Selectable(o.label, o.op == rv.Operation()) {
				rv.SetOperation(o.op)
			}
		}
		native.EndCombo()
	}
}

func revolveAxisCombo(rv *app.RevolveTool) {
	preview := "Y Axis"
	for _, a := range revolveAxes {
		if a.ref == rv.Axis() {
			preview = a.label
		}
	}
	if native.BeginCombo("Axis", preview) {
		for _, a := range revolveAxes {
			if native.Selectable(a.label, a.ref == rv.Axis()) {
				rv.SetAxis(a.ref)
			}
		}
		native.EndCombo()
	}
}

// drawRevolveAngle offers a full-revolution checkbox and, when unchecked, an angle field
// in degrees written back to the tool as radians.
func drawRevolveAngle(rv *app.RevolveTool) {
	full := rv.IsFullRevolution()
	if native.Checkbox("Full revolution", &full) {
		if full {
			rv.SetFullRevolution()
		} else {
			rv.SetAngle(float64(revolveUI.angleDeg) * stdmath.Pi / 180)
		}
	}
	if full {
		return
	}
	native.Text("Angle (deg)")
	native.InputFloat("##revolve-angle", &revolveUI.angleDeg)
	rv.SetAngle(float64(revolveUI.angleDeg) * stdmath.Pi / 180)
}

func drawRevolveButtons(s *app.Session, rv *app.RevolveTool) {
	native.BeginDisabled(!rv.CanCommit())
	if native.Button("OK") {
		_ = s.OK() // a failed commit (e.g. open profile) keeps the tool open
	}
	native.EndDisabled()
	native.SameLine()
	if native.Button("Cancel") {
		s.CancelTool()
	}
}
