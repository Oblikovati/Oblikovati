//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/feature"
)

// The Coil flow in the head: while the Coil tool runs, a modeless property panel (the
// reference panel schema) drives the tool — the profile chip, the helix axis, the
// pitch and revolutions, and the boolean output — then OK/Cancel. The picked region is
// outlined by the tool's preview.
var coilUI = struct {
	pitch, revolutions float32
	// Variable-pitch rail (M06-F09, #624): the rows grid state and the
	// flat-end toggles + sweep angles (degrees in the UI).
	variable           bool
	rows               []coilRowUI
	startFlat, endFlat bool
	startTransition    float32
	startFlatAngle     float32
	endTransition      float32
	endFlatAngle       float32
	seeded             *app.CoilTool // the tool the fields were seeded from (nil = none)
}{pitch: 1, revolutions: 3}

// coilRowUI is one editable pitch row of the variable grid.
type coilRowUI struct {
	pitch      float32
	revolution float32
}

// drawCoilDialog shows the Coil property panel while the Coil tool is active, syncing
// every control with the tool each frame; OK commits, Cancel aborts.
func drawCoilDialog(s *app.Session) {
	c := s.ActiveCoil()
	if c == nil {
		coilUI.seeded = nil
		return
	}
	refreshCoilUI(c)
	native.SetNextWindowSizeOnce(340, 340)
	if native.Begin("Coil") {
		title := "Coil"
		if name := c.EditingName(); name != "" {
			title = name // re-editing a committed coil: the breadcrumb names it
		}
		drawFeatureBreadcrumb(title, c.SourceSketchName())
		drawCoilInputGeometry(c)
		drawCoilBehavior(s, c)
		drawCoilOutput(c)
		native.Separator()
		drawCommitCancelButtons(s, c.CanCommit())
	}
	native.End()
}

func refreshCoilUI(c *app.CoilTool) {
	if coilUI.seeded == c {
		return
	}
	coilUI.pitch = float32(c.Pitch())
	coilUI.revolutions = float32(c.Revolutions())
	coilUI.rows = coilUI.rows[:0]
	for _, r := range c.PitchRows() {
		coilUI.rows = append(coilUI.rows, coilRowUI{pitch: float32(r.Pitch), revolution: float32(r.Revolution)})
	}
	coilUI.variable = len(coilUI.rows) > 0
	seedCoilEnds(c)
	coilUI.seeded = c
}

// seedCoilEnds loads the tool's end conditions into the panel (degrees).
func seedCoilEnds(c *app.CoilTool) {
	start, end := c.StartEnd(), c.EndEnd()
	coilUI.startFlat, coilUI.endFlat = start.Flat, end.Flat
	coilUI.startTransition = float32(start.TransitionAngle * 180 / stdmath.Pi)
	coilUI.startFlatAngle = float32(start.FlatAngle * 180 / stdmath.Pi)
	coilUI.endTransition = float32(end.TransitionAngle * 180 / stdmath.Pi)
	coilUI.endFlatAngle = float32(end.FlatAngle * 180 / stdmath.Pi)
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

// drawCoilBehavior is the Behavior section: the helix pitch and revolutions,
// the variable-pitch rows grid, and the flat-end conditions (M06-F09, #624).
func drawCoilBehavior(s *app.Session, c *app.CoilTool) {
	if !propertySection("Behavior") {
		return
	}
	lengthCmRow(s, "Pitch", "coil-pitch", &coilUI.pitch)
	c.SetPitch(float64(coilUI.pitch))
	propertyFloatRow("Revolutions", "coil-revolutions", "", &coilUI.revolutions)
	c.SetRevolutions(float64(coilUI.revolutions))
	drawCoilPitchRows(s, c)
	drawCoilEnds(s, c)
}

// drawCoilPitchRows is the variable-pitch rows grid: toggle, per-row pitch +
// revolution inputs, add/remove row buttons.
func drawCoilPitchRows(s *app.Session, c *app.CoilTool) {
	if native.Checkbox("Variable pitch", &coilUI.variable) && coilUI.variable && len(coilUI.rows) == 0 {
		coilUI.rows = []coilRowUI{
			{pitch: coilUI.pitch, revolution: 0},
			{pitch: coilUI.pitch, revolution: coilUI.revolutions},
		}
	}
	if !coilUI.variable {
		c.SetPitchRows(nil)
		return
	}
	for i := range coilUI.rows {
		id := fmt.Sprintf("coil-row-%d", i)
		lengthCmRow(s, fmt.Sprintf("Row %d Pitch", i+1), id+"-pitch", &coilUI.rows[i].pitch)
		propertyFloatRow(fmt.Sprintf("Row %d Rev", i+1), id+"-rev", "", &coilUI.rows[i].revolution)
	}
	if native.Button("Add Row") {
		last := coilUI.rows[len(coilUI.rows)-1]
		coilUI.rows = append(coilUI.rows, coilRowUI{pitch: last.pitch, revolution: last.revolution + 1})
	}
	native.SameLine()
	if native.Button("Remove Row") && len(coilUI.rows) > 2 {
		coilUI.rows = coilUI.rows[:len(coilUI.rows)-1]
	}
	rows := make([]feature.CoilPitchRow, len(coilUI.rows))
	for i, r := range coilUI.rows {
		rows[i] = feature.CoilPitchRow{Pitch: float64(r.pitch), Revolution: float64(r.revolution)}
	}
	c.SetPitchRows(rows)
}

// drawCoilEnds is the flat start/end condition controls (angles in degrees).
func drawCoilEnds(s *app.Session, c *app.CoilTool) {
	native.Checkbox("Flat start", &coilUI.startFlat)
	if coilUI.startFlat {
		angleDegRow(s, "Start Transition", "coil-start-trans", &coilUI.startTransition)
		angleDegRow(s, "Start Flat", "coil-start-flat", &coilUI.startFlatAngle)
	}
	native.Checkbox("Flat end", &coilUI.endFlat)
	if coilUI.endFlat {
		angleDegRow(s, "End Transition", "coil-end-trans", &coilUI.endTransition)
		angleDegRow(s, "End Flat", "coil-end-flat", &coilUI.endFlatAngle)
	}
	c.SetEndConditions(coilEnd(coilUI.startFlat, coilUI.startTransition, coilUI.startFlatAngle),
		coilEnd(coilUI.endFlat, coilUI.endTransition, coilUI.endFlatAngle))
}

// coilEnd converts panel degrees into the feature's radian end condition.
func coilEnd(flat bool, transitionDeg, flatDeg float32) feature.CoilEndCondition {
	if !flat {
		return feature.CoilEndCondition{}
	}
	return feature.CoilEndCondition{
		Flat:            true,
		TransitionAngle: float64(transitionDeg) * stdmath.Pi / 180,
		FlatAngle:       float64(flatDeg) * stdmath.Pi / 180,
	}
}

// drawCoilOutput is the Output section: the shared Boolean toggle row.
func drawCoilOutput(c *app.CoilTool) {
	if !propertySection("Output") {
		return
	}
	drawBooleanPropertyRow("coil-boolean", c.Operation(), c.SetOperation)
}
