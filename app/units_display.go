// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/model/param"
)

// Session-level conversions between a tool's database-unit values (cm for
// length, radians for angle) and the document's preferred display units — the
// boundary every tool dialog field crosses so its value and label follow the
// document's units (Oblikovati/Oblikovati#146). LengthUnitName/AngleUnitName
// (extrude_session.go) supply the matching labels.

// LengthToDisplay converts a database-unit length (cm) to the document's
// preferred length unit; LengthFromDisplay is its inverse.
func (s *Session) LengthToDisplay(cm float64) float64 {
	return s.DocumentUnits().ToPreferred(param.Q(cm, param.Length))
}

func (s *Session) LengthFromDisplay(value float64) float64 {
	return s.DocumentUnits().FromPreferred(value, param.Length).Value
}

// AngleToDisplay converts a database-unit angle (radians) to the document's
// preferred angle unit; AngleFromDisplay is its inverse.
func (s *Session) AngleToDisplay(radians float64) float64 {
	return s.DocumentUnits().ToPreferred(param.Q(radians, param.Angle))
}

func (s *Session) AngleFromDisplay(value float64) float64 {
	return s.DocumentUnits().FromPreferred(value, param.Angle).Value
}

// AngleDegToDisplay / AngleDisplayToDeg bridge dialogs that hold an angle in
// DEGREES (their on-screen buffer) to the document's preferred angle unit.
func (s *Session) AngleDegToDisplay(degrees float64) float64 {
	return s.AngleToDisplay(degrees * stdmath.Pi / 180)
}

func (s *Session) AngleDisplayToDeg(value float64) float64 {
	return s.AngleFromDisplay(value) * 180 / stdmath.Pi
}
