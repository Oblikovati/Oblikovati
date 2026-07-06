//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Revolve flow in the head: while the Revolve tool runs, a modeless property panel
// (the reference panel schema: Input Geometry / Behavior / Output sections over a
// label/control grid) drives the tool — the profile chip, the axis of revolution, the
// swept angle with its full-revolution toggle, and the boolean output — then OK/Cancel.
// The picked region is outlined in the viewport by the tool's preview. The angle is
// shown in degrees.
var revolveUI = struct {
	angleDeg   float32
	centerline bool
	seeded     *app.RevolveTool // the tool the fields were seeded from (nil = none)
}{angleDeg: 360}

// revolveAxes names each origin axis for the axis combo, paired with its work reference.
var revolveAxes = []struct {
	label string
	ref   feature.WorkRef
}{{"X Axis", feature.OriginXAxis}, {"Y Axis", feature.OriginYAxis}, {"Z Axis", feature.OriginZAxis}}

// drawRevolveDialog shows the Revolve property panel while the Revolve tool is active,
// syncing every control with the tool each frame; OK commits, Cancel aborts.
func drawRevolveDialog(s *app.Session) {
	rv := s.ActiveRevolve()
	if rv == nil {
		revolveUI.seeded = nil
		return
	}
	if revolveUI.seeded != rv {
		revolveUI.angleDeg = seedRevolveAngle(rv)
		revolveUI.centerline = rv.UseCenterline()
		revolveUI.seeded = rv
	}
	dialogSizeOnce(340, 360)
	if native.Begin("Revolve") {
		title := "Revolve"
		if name := rv.EditingName(); name != "" {
			title = name // re-editing a committed revolve: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, rv.SourceSketchName())
		drawRevolveInputGeometry(rv)
		drawRevolveBehavior(s, rv)
		drawRevolveOutput(rv)
		native.Separator()
		drawCommitCancelButtons(s, rv.CanCommit())
	}
	native.End()
}

// seedRevolveAngle returns the panel's initial angle in degrees from the tool (a full
// revolution shows 360).
func seedRevolveAngle(rv *app.RevolveTool) float32 {
	if rv.IsFullRevolution() {
		return 360
	}
	return float32(rv.Angle() * 180 / stdmath.Pi)
}

// drawRevolveInputGeometry is the Input Geometry section: the required Profiles chip
// and the Axis row (origin-axis combo, swapped for the sketch's centerline when that
// toggle is on).
func drawRevolveInputGeometry(rv *app.RevolveTool) {
	if !propertySection("Input Geometry") {
		return
	}
	propertyRow("Profiles")
	_, picked := rv.PickedProfile()
	if propertySelectorChip("revolve-profiles", pickChipText(picked, "1 Profile", "Select Profile"), picked, true) {
		rv.ClearProfile()
	}
	native.SetItemTooltip("Click a region in the viewport")
	propertyRow("Axis")
	drawRevolveAxisControls(rv)
}

// drawRevolveAxisControls renders the axis combo (greyed while the centerline drives
// the revolution) with the about-centerline toggle beneath it, aligned to the control
// column.
func drawRevolveAxisControls(rv *app.RevolveTool) {
	native.BeginDisabled(revolveUI.centerline)
	native.SetNextItemWidth(propertyFieldWidth)
	revolveAxisCombo(rv)
	native.EndDisabled()
	propertyRow("")
	native.Checkbox("About sketch centerline", &revolveUI.centerline)
	rv.SetUseCenterline(revolveUI.centerline)
}

func revolveAxisCombo(rv *app.RevolveTool) {
	preview := "Y Axis"
	for _, a := range revolveAxes {
		if a.ref == rv.Axis() {
			preview = a.label
		}
	}
	if native.BeginCombo("##revolve-axis", preview) {
		for _, a := range revolveAxes {
			if native.Selectable(a.label, a.ref == rv.Axis()) {
				rv.SetAxis(a.ref)
			}
		}
		native.EndCombo()
	}
}

// drawRevolveBehavior is the Behavior section: the Angle A field (greyed during a full
// revolution) with the full-revolution toggle beside it.
func drawRevolveBehavior(s *app.Session, rv *app.RevolveTool) {
	if !propertySection("Behavior") {
		return
	}
	propertyRow("Angle A")
	native.BeginDisabled(rv.IsFullRevolution())
	native.SetNextItemWidth(propertyFieldWidth)
	disp := float32(s.AngleDegToDisplay(float64(revolveUI.angleDeg)))
	if parameterField(s, "revolve-angle", s.AngleUnitName(), s.AnglePrecision(), paramAngle, &disp) {
		revolveUI.angleDeg = float32(s.AngleDisplayToDeg(float64(disp)))
	}
	if !rv.IsFullRevolution() {
		rv.SetAngle(float64(revolveUI.angleDeg) * stdmath.Pi / 180)
	}
	native.EndDisabled()
	native.SameLine()
	if drawPropertyToggle("revolve-full", "angle-full", "Full — revolve the whole 360°", rv.IsFullRevolution()) {
		toggleRevolveFull(rv)
	}
}

// toggleRevolveFull flips the full-revolution mode: leaving it restores the swept angle
// from the panel field (a non-positive field falls back to 360°).
func toggleRevolveFull(rv *app.RevolveTool) {
	if !rv.IsFullRevolution() {
		rv.SetFullRevolution()
		return
	}
	if revolveUI.angleDeg <= 0 {
		revolveUI.angleDeg = 360
	}
	rv.SetAngle(float64(revolveUI.angleDeg) * stdmath.Pi / 180)
}

// drawRevolveOutput is the Output section: the shared Boolean toggle row.
func drawRevolveOutput(rv *app.RevolveTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("revolve-boolean", rv.Operation(), rv.SetOperation)
}
