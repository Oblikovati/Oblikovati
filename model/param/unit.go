// SPDX-License-Identifier: GPL-2.0-only

package param

// Unit is the dimension category of a [Quantity] (contract: UnitsTypeEnum). The
// numeric members carry stable, explicit ids per the plan's convention — never
// renumber them, as they are persisted. Boolean and Text are degenerate
// (non-arithmetic) categories so a parameter value can also be a flag or label.
type Unit uint16

const (
	Unitless Unit = 0
	Length   Unit = 1 // database unit: centimetre
	Angle    Unit = 2 // database unit: radian
	Area     Unit = 3 // cm²
	Volume   Unit = 4 // cm³
	Mass     Unit = 5 // kilogram
	Time     Unit = 6 // second
	Boolean  Unit = 7
	Text     Unit = 8
)

// dimension is the exponent signature over the base dimensions length (L),
// angle (A), mass (M), and time (T). It lets the evaluator combine units
// correctly: Length·Length → L² → Area, Volume/Area → L → Length, and reject
// nonsense like Length + Angle.
type dimension struct {
	l, a, m, t int8
}

// dimensions maps each arithmetic Unit to its exponent signature. Boolean and
// Text are intentionally absent — they have no dimension and cannot take part
// in arithmetic.
var dimensions = map[Unit]dimension{
	Unitless: {0, 0, 0, 0},
	Length:   {1, 0, 0, 0},
	Area:     {2, 0, 0, 0},
	Volume:   {3, 0, 0, 0},
	Angle:    {0, 1, 0, 0},
	Mass:     {0, 0, 1, 0},
	Time:     {0, 0, 0, 1},
}

// dimensionOf returns the unit's exponent signature and whether the unit is
// arithmetic (false for Boolean/Text).
func dimensionOf(u Unit) (dimension, bool) {
	d, ok := dimensions[u]
	return d, ok
}

// unitForDimension returns the named Unit with the given signature, or false
// when no category matches (e.g. an L⁴ result has no name).
func unitForDimension(d dimension) (Unit, bool) {
	for u, dd := range dimensions {
		if dd == d {
			return u, true
		}
	}
	return 0, false
}

// IsArithmetic reports whether values of this unit can be combined with
// arithmetic operators (everything except Boolean and Text).
func (u Unit) IsArithmetic() bool {
	_, ok := dimensions[u]
	return ok
}

// unitNames gives each unit category a stable diagnostic name.
var unitNames = map[Unit]string{
	Unitless: "unitless", Length: "length", Angle: "angle", Area: "area",
	Volume: "volume", Mass: "mass", Time: "time", Boolean: "boolean", Text: "text",
}

// String returns the unit category name, for diagnostics and error messages.
func (u Unit) String() string {
	if name, ok := unitNames[u]; ok {
		return name
	}
	return "unit(?)"
}
