// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	stdmath "math"
	"strconv"
	"strings"
)

// unitDef defines a named user unit: its dimension category and the factor that
// converts a value expressed in it to database units (dbValue = userValue·factor).
type unitDef struct {
	category Unit
	factor   float64
}

// namedUnits is the registry of user-facing unit names (the parse/display
// vocabulary). Database units are the factor-1 member of each category: cm for
// length, radian for angle, and so on.
var namedUnits = map[string]unitDef{
	// Length (database unit: cm).
	"mm": {Length, 0.1}, "cm": {Length, 1}, "m": {Length, 100}, "km": {Length, 100000},
	"in": {Length, 2.54}, "ft": {Length, 30.48},
	// Angle (database unit: radian).
	"rad": {Angle, 1}, "deg": {Angle, stdmath.Pi / 180},
	// Area / Volume (database units: cm², cm³).
	"mm^2": {Area, 0.01}, "cm^2": {Area, 1}, "m^2": {Area, 10000},
	"mm^3": {Volume, 0.001}, "cm^3": {Volume, 1}, "m^3": {Volume, 1e6}, "l": {Volume, 1000},
	// Mass / Time.
	"kg": {Mass, 1}, "g": {Mass, 0.001}, "lb": {Mass, 0.45359237},
	"s": {Time, 1}, "ms": {Time, 0.001}, "min": {Time, 60}, "hr": {Time, 3600},
}

// lookupUnit returns the definition for a user unit name.
func lookupUnit(name string) (unitDef, bool) {
	d, ok := namedUnits[name]
	return d, ok
}

// UnitsOfMeasure holds a document's display-unit preferences and performs the
// conversion/formatting at the boundary (contract: UnitsOfMeasure). Stored
// values never change when preferences do — only their presentation.
type UnitsOfMeasure struct {
	prefs map[Unit]string // category → preferred user-unit name
}

// DefaultUnitsOfMeasure returns metric defaults (mm, degrees, kg, seconds).
func DefaultUnitsOfMeasure() UnitsOfMeasure {
	return UnitsOfMeasure{prefs: map[Unit]string{
		Length: "mm", Angle: "deg", Area: "mm^2", Volume: "mm^3", Mass: "kg", Time: "s",
	}}
}

// SetPreferred sets the display unit for a category; the name must be registered
// and belong to that category.
func (m UnitsOfMeasure) SetPreferred(category Unit, name string) error {
	def, ok := lookupUnit(name)
	if !ok || def.category != category {
		return fmt.Errorf("param: %q is not a registered %s unit", name, category)
	}
	m.prefs[category] = name
	return nil
}

// preferredFactor returns the db-conversion factor and name of the preferred
// unit for a category (factor 1 / "" for unitless and unconfigured categories).
func (m UnitsOfMeasure) preferredFactor(category Unit) (factor float64, name string) {
	name, ok := m.prefs[category]
	if !ok {
		return 1, ""
	}
	return namedUnits[name].factor, name
}

// Parse converts a user string ("25 mm", "30 deg", "5") to a database-unit
// [Quantity]. A bare number is interpreted in the preferred unit of defaultCat;
// an explicit unit suffix overrides both the factor and the category. This is
// the GetValueFromString equivalent.
func (m UnitsOfMeasure) Parse(s string, defaultCat Unit) (Quantity, error) {
	numText, unitName := splitNumberUnit(s)
	value, err := strconv.ParseFloat(numText, 64)
	if err != nil {
		return Quantity{}, fmt.Errorf("param: cannot parse number in %q: %w", s, err)
	}
	if unitName == "" {
		factor, _ := m.preferredFactor(defaultCat)
		return Quantity{value * factor, defaultCat}, nil
	}
	def, ok := lookupUnit(unitName)
	if !ok {
		return Quantity{}, fmt.Errorf("param: unknown unit %q in %q", unitName, s)
	}
	return Quantity{value * def.factor, def.category}, nil
}

// Format renders a database-unit quantity in its category's preferred unit, with
// the unit name appended (e.g. Q(1, Length) → "10 mm"). The GetStringFromValue
// equivalent. Round-tripping through [UnitsOfMeasure.Parse] is lossless.
func (m UnitsOfMeasure) Format(q Quantity) string {
	value, name := m.FormatValue(q), m.unitName(q.Unit)
	if name == "" {
		return value
	}
	return value + " " + name
}

// FormatValue renders just the numeric part of q in its preferred unit, using
// the shortest exact decimal so parse/format round-trips without loss.
func (m UnitsOfMeasure) FormatValue(q Quantity) string {
	factor, _ := m.preferredFactor(q.Unit)
	return strconv.FormatFloat(q.Value/factor, 'g', -1, 64)
}

// unitName returns the preferred display-unit name for a category.
func (m UnitsOfMeasure) unitName(category Unit) string {
	_, name := m.preferredFactor(category)
	return name
}

// PreferredName returns the preferred display-unit name for a category (e.g. "mm").
func (m UnitsOfMeasure) PreferredName(category Unit) string { return m.unitName(category) }

// ToPreferred returns q's numeric value expressed in its category's preferred unit —
// the value a UI edits (vs the database-unit value stored in the model).
func (m UnitsOfMeasure) ToPreferred(q Quantity) float64 {
	factor, _ := m.preferredFactor(q.Unit)
	return q.Value / factor
}

// FromPreferred builds a database-unit [Quantity] from a value given in category's
// preferred unit — the inverse of [UnitsOfMeasure.ToPreferred].
func (m UnitsOfMeasure) FromPreferred(value float64, category Unit) Quantity {
	factor, _ := m.preferredFactor(category)
	return Quantity{value * factor, category}
}

// splitNumberUnit separates the leading numeric literal from a trailing unit
// name, tolerating optional whitespace between them.
func splitNumberUnit(s string) (numText, unitName string) {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) && isNumberByte(s[end], end, s) {
		end++
	}
	return s[:end], strings.TrimSpace(s[end:])
}

// isNumberByte reports whether the byte at index i can be part of a float
// literal: digits and dot anywhere, sign only leading or after an exponent.
func isNumberByte(b byte, i int, s string) bool {
	switch {
	case b >= '0' && b <= '9', b == '.':
		return true
	case b == 'e' || b == 'E':
		return i > 0
	case b == '+' || b == '-':
		return i == 0 || s[i-1] == 'e' || s[i-1] == 'E'
	default:
		return false
	}
}
