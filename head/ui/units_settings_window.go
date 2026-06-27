//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/model/param"
)

// drawUnitsSettingsWindow renders the Document Settings ▸ Units dialog while it is
// open (Oblikovati/Oblikovati#146): the document's length/angle/mass/time display
// units, the length/angle display precision, the length format (decimal /
// fractional / architectural) and the angle DMS toggle. Each edit writes the whole
// units value back through the session (the same surface the API uses and the .obk
// persists), so every open tool dialog and dimension immediately re-renders.
func drawUnitsSettingsBody(s *app.Session) {
	u := s.DocumentUnits().Clone()
	if editDocumentUnits(&u) {
		s.SetDocumentUnits(u)
	}
	native.Separator()
	if native.Button("Done") {
		s.CloseUnitsSettings()
	}
}

// editDocumentUnits draws the unit pickers, precision inputs and format controls
// into u, returning whether any changed. The controls only READ + render here; the
// apply decisions live in the app layer (app.Apply*), which run every frame (a
// no-op when the value is unchanged) so the dialog stays drift-free and the apply
// logic is unit-testable without a GUI.
func editDocumentUnits(u *param.UnitsOfMeasure) bool {
	changed := app.ApplyUnit(u, param.Length, unitCombo("Length", "units-length", u.PreferredName(param.Length), app.LengthUnitOptions()))
	changed = app.ApplyUnit(u, param.Angle, unitCombo("Angle", "units-angle", u.PreferredName(param.Angle), app.AngleUnitOptions())) || changed
	changed = app.ApplyUnit(u, param.Mass, unitCombo("Mass", "units-mass", u.PreferredName(param.Mass), app.MassUnitOptions())) || changed
	changed = app.ApplyUnit(u, param.Time, unitCombo("Time", "units-time", u.PreferredName(param.Time), app.TimeUnitOptions())) || changed
	native.Separator()
	changed = app.ApplyLengthFormat(u, unitCombo("Length format", "units-lenfmt", u.LengthFormat().String(), []string{"decimal", "fractional", "architectural"})) || changed
	lp, ap := precisionInputs(u)
	changed = app.ApplyLengthPrecision(u, lp) || changed
	changed = app.ApplyAnglePrecision(u, ap) || changed
	changed = app.ApplyAngleDMS(u, angleDMSToggle(u)) || changed
	return changed
}

// unitCombo draws a labelled combo over options and returns the user's selection
// (the unchanged current value when nothing is picked this frame).
func unitCombo(label, id, current string, options []string) string {
	selected := current
	native.Text(label)
	native.SameLine()
	if native.BeginCombo("##"+id, current) {
		for _, o := range options {
			if native.Selectable(o, o == current) {
				selected = o
			}
		}
		native.EndCombo()
	}
	return selected
}

// precisionInputs draws the length/angle precision fields and returns their values.
func precisionInputs(u *param.UnitsOfMeasure) (length, angle int) {
	lp := int32(u.LengthPrecision())
	native.InputInt("Length precision", &lp)
	ap := int32(u.AnglePrecision())
	native.InputInt("Angle precision", &ap)
	return int(lp), int(ap)
}

// angleDMSToggle draws the DMS checkbox and returns its state.
func angleDMSToggle(u *param.UnitsOfMeasure) bool {
	dms := u.AngleFormat() == param.AngleDMS
	native.Checkbox("Angle as DMS (deg/min/sec)", &dms)
	return dms
}
