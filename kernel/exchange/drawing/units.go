// SPDX-License-Identifier: GPL-2.0-only

package drawing

// $INSUNITS codes used across both codecs. The full set lives in metersPerInsunit; these
// named constants are the ones the writers default to (centimetre is the model's database
// unit) so call sites don't carry bare magic numbers.
const (
	INSUnitless    = 0
	INSInches      = 1
	INSFeet        = 2
	INSMillimetres = 4
	INSCentimetres = 5
	INSMetres      = 6
)

// metersPerInsunit maps the $INSUNITS code to metres per drawing unit. Code 0 is unitless
// (no intrinsic scale); unsupported/astronomical codes are omitted (ok=false), leaving the
// importer to fall back to the document's unit.
var metersPerInsunit = map[int]float64{
	1: 0.0254, 2: 0.3048, 3: 1609.344, 4: 0.001, 5: 0.01, 6: 1, 7: 1000,
	8: 0.0254e-6, 9: 0.0254e-3, 10: 0.9144, 11: 1e-10, 12: 1e-9, 13: 1e-6,
	14: 0.1, 15: 10, 16: 100, 17: 1e9,
	21: 0.30480061, 22: 0.0254000508, 23: 0.91440183, 24: 1609.347219,
}

// MetersPerUnit returns the length, in metres, of one drawing unit for the given $INSUNITS
// code, and whether the code carries a known unit. Unitless (0) and unknown codes return
// ok=false.
//
//	m, ok := drawing.MetersPerUnit(4) // 0.001, true (millimetres)
func MetersPerUnit(insunits int) (float64, bool) {
	m, ok := metersPerInsunit[insunits]
	return m, ok
}
