// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"strconv"
	"strings"
)

// User-authored expressions must be interpreted in the DOCUMENT'S display unit, never the raw
// database unit. A bare number typed into a dimension/parameter field means the displayed unit —
// "7" in a millimetre document is 7 mm (0.7 cm database), NOT 7 database units (7 cm = 70 mm).
// The expression grammar makes a bare number Unitless (expr_parser.go), and a Unitless value
// consumed by a Length/Angle parameter is taken as the database unit, so raw SetExpression on
// user input inflates every bare number by the db/display factor (10× for a mm document). Every
// path that receives a typed expression MUST pass it through QualifyAuthored first; raw
// SetExpression is reserved for deserialised or already-unit-bearing source.

// QualifyAuthored returns src rewritten so a bare/unitless value is expressed in units' preferred
// unit for cat, leaving an already-dimensioned value or a parameter-referencing formula untouched
// so it stays live. It is the units-aware seam every user-input path runs a typed expression
// through before SetExpression (see the package note above).
//
//	p.QualifyAuthored("7", Length, mmUnits)        // → "7 mm"      (0.7 cm database)
//	p.QualifyAuthored("3*2", Length, mmUnits)      // → "(3*2) * 1 mm"
//	p.QualifyAuthored("7 mm", Length, mmUnits)     // → "7 mm"      (already dimensioned)
//	p.QualifyAuthored("width/2", Length, mmUnits)  // → "width/2"   (formula, kept live)
func (p *Parameter) QualifyAuthored(src string, cat Unit, units UnitsOfMeasure) string {
	if cat == Unitless {
		return src // a unitless field takes the bare number as-is
	}
	unit := units.PreferredName(cat)
	if unit == "" {
		return src // category has no configured display unit — nothing to attach
	}
	q, err := p.evalAuthored(src)
	if err != nil || q.Unit != Unitless {
		// A parse error, an unresolved reference, or an already-dimensioned result: leave src
		// untouched so a formula stays live and the downstream parser reports any real error.
		return src
	}
	if isLoneNumber(src) {
		return strings.TrimSpace(src) + " " + unit
	}
	// A compound unitless expression ("3*2", "1/8"): multiply by one display unit so the whole
	// expression carries the category's dimension. Parenthesised to keep operator precedence.
	return "(" + strings.TrimSpace(src) + ") * 1 " + unit
}

// evalAuthored evaluates src to learn its dimension: against the owning parameter table (so
// formulas and references resolve) when the parameter belongs to one, else as a constant.
func (p *Parameter) evalAuthored(src string) (Quantity, error) {
	if p.owner != nil {
		return p.owner.EvaluateExpression(src)
	}
	e, err := Parse(src)
	if err != nil {
		return Quantity{}, err
	}
	return e.Eval(nil)
}

// isLoneNumber reports whether src is a single numeric literal with no unit or operator, so it can
// be qualified by appending the unit name directly ("7" → "7 mm") rather than the multiply form.
func isLoneNumber(src string) bool {
	_, err := strconv.ParseFloat(strings.TrimSpace(src), 64)
	return err == nil
}
