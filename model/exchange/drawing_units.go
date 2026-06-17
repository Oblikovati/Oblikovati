// SPDX-License-Identifier: GPL-2.0-only

package exchange

import (
	"oblikovati.org/kernel/exchange/drawing"
	"oblikovati.org/model/param"
)

// Sketch DXF/DWG export units (Oblikovati/Oblikovati#146). The kernel works in
// centimetres; a sketch export writes the document's preferred length unit — it
// declares the matching $INSUNITS code and scales coordinates from centimetres
// into that unit. The import side already scales $INSUNITS → centimetres
// (sketch_from_drawing.go), so a round-trip preserves size.

// insCodeForUnit maps a document length-unit name to its $INSUNITS code; an
// unrecognised unit falls back to centimetres (the database unit, scale 1).
var insCodeForUnit = map[string]int{
	"mm": drawing.INSMillimetres, "cm": drawing.INSCentimetres, "m": drawing.INSMetres,
	"in": drawing.INSInches, "ft": drawing.INSFeet,
}

// documentDrawingUnit returns the $INSUNITS code and the centimetre→unit scale for
// a document's preferred length unit, so an export labels and sizes coordinates
// in that unit.
func documentDrawingUnit(u param.UnitsOfMeasure) (insUnits int, scale float64) {
	ins, ok := insCodeForUnit[u.PreferredName(param.Length)]
	if !ok {
		return drawing.INSCentimetres, 1
	}
	// ToPreferred(1 cm) is the value of one database unit in the preferred unit —
	// exactly the centimetre→unit coordinate scale.
	return ins, u.ToPreferred(param.Q(1, param.Length))
}
