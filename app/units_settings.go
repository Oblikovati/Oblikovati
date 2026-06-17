// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/api/types"
	"oblikovati.org/model/param"
)

// Document Settings ▸ Units dialog state and the write side of the document's
// units (Oblikovati/Oblikovati#146). The head opens the dialog from the Tools
// menu, edits a copy of DocumentUnits, and applies it through SetDocumentUnits —
// the same units object the API reads/writes and the .obk persists.

// OpenUnitsSettings / CloseUnitsSettings / UnitsSettingsOpen drive the dialog.
func (s *Session) OpenUnitsSettings()      { s.unitsSettingsOpen = true }
func (s *Session) CloseUnitsSettings()     { s.unitsSettingsOpen = false }
func (s *Session) UnitsSettingsOpen() bool { return s.unitsSettingsOpen }

// SetDocumentUnits applies an edited units object to the active part — the write
// counterpart of [Session.DocumentUnits]. Build it from DocumentUnits().Clone()
// so the edit does not touch the live document until stored. A no-op when there
// is no active part.
func (s *Session) SetDocumentUnits(u param.UnitsOfMeasure) {
	if part, err := activePart(s); err == nil {
		part.SetUnits(u)
	}
}

// LengthUnitOptions / AngleUnitOptions / MassUnitOptions / TimeUnitOptions are the
// unit names the Units dialog offers per category, so the head needs no unit
// vocabulary of its own.
func LengthUnitOptions() []string { return []string{"mm", "cm", "m", "in", "ft"} }
func AngleUnitOptions() []string  { return []string{"deg", "rad"} }
func MassUnitOptions() []string   { return []string{"kg", "g", "lb"} }
func TimeUnitOptions() []string   { return []string{"s", "ms", "min", "hr"} }

// Apply* mutate a units object from a UI control's current value, returning
// whether anything changed. They are the pure decision logic behind the Units
// dialog's combos and inputs (the head draws the control and feeds its value
// here), so the apply paths are testable without a GUI. An invalid/unchanged
// value is a no-op returning false.

// ApplyUnit sets a category's preferred unit by name.
func ApplyUnit(u *param.UnitsOfMeasure, cat param.Unit, name string) bool {
	if name == u.PreferredName(cat) {
		return false
	}
	return u.SetPreferred(cat, name) == nil
}

// ApplyLengthFormat sets the length display format from its wire spelling.
func ApplyLengthFormat(u *param.UnitsOfMeasure, spelling string) bool {
	f, ok := types.ParseParameterDisplayFormat(spelling)
	if !ok || f == u.LengthFormat() {
		return false
	}
	u.SetLengthFormat(f)
	return true
}

// ApplyLengthPrecision / ApplyAnglePrecision set the display decimal places.
func ApplyLengthPrecision(u *param.UnitsOfMeasure, places int) bool {
	if places == u.LengthPrecision() {
		return false
	}
	return u.SetLengthPrecision(places) == nil
}

func ApplyAnglePrecision(u *param.UnitsOfMeasure, places int) bool {
	if places == u.AnglePrecision() {
		return false
	}
	return u.SetAnglePrecision(places) == nil
}

// ApplyAngleDMS selects degrees-minutes-seconds (true) or decimal degrees.
func ApplyAngleDMS(u *param.UnitsOfMeasure, dms bool) bool {
	want := param.AngleDecimal
	if dms {
		want = param.AngleDMS
	}
	if want == u.AngleFormat() {
		return false
	}
	u.SetAngleFormat(want)
	return true
}
