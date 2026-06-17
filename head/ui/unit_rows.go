// SPDX-License-Identifier: GPL-2.0-only

package ui

import "oblikovati.org/app"

// Unit-aware property rows (Oblikovati/Oblikovati#146): a length/angle field
// whose label AND value follow the document's preferred unit, replacing the
// hand-rolled rows that hardcoded "mm"/"deg".
//
// They are drop-in replacements that preserve each dialog's existing buffer
// convention — a length buffer in database centimetres, an angle buffer in
// degrees — and only adapt what the field SHOWS. The on-screen value is derived
// from the buffer every frame and written back ONLY when edited, so an untouched
// inch/radian field never accumulates round-trip error.

// lengthCmRow shows a length buffer held in database centimetres as a field in
// the document's length unit. cmBuf stays in cm, so the dialog's existing
// tool-write line keeps working unchanged.
func lengthCmRow(s *app.Session, label, id string, cmBuf *float32) {
	lengthCmRowHint(s, label, id, "", cmBuf)
}

// angleDegRow shows an angle buffer held in degrees as a field in the document's
// angle unit. degBuf stays in degrees (its tool boundary already expects them).
func angleDegRow(s *app.Session, label, id string, degBuf *float32) {
	angleDegRowHint(s, label, id, "", degBuf)
}

// lengthCmRowHint / angleDegRowHint are the same rows with a free-text hint
// appended after the unit label (e.g. " (+out / −in)" for a signed value).
func lengthCmRowHint(s *app.Session, label, id, hint string, cmBuf *float32) {
	disp := float32(s.LengthToDisplay(float64(*cmBuf)))
	if propertyFloatRow(label, id, s.LengthUnitName()+hint, &disp) {
		*cmBuf = float32(s.LengthFromDisplay(float64(disp)))
	}
}

func angleDegRowHint(s *app.Session, label, id, hint string, degBuf *float32) {
	disp := float32(s.AngleDegToDisplay(float64(*degBuf)))
	if propertyFloatRow(label, id, s.AngleUnitName()+hint, &disp) {
		*degBuf = float32(s.AngleDisplayToDeg(float64(disp)))
	}
}
