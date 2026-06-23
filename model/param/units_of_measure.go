// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	stdmath "math"
	"strconv"
	"strings"

	"oblikovati.org/api/types"
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
	// Length (database unit: cm). Sub-millimetre metric units down to picometres are accepted by
	// the parser; "µm" and its ASCII spelling "um" are the same micrometre.
	"pm": {Length, 1e-10}, "nm": {Length, 1e-7}, "um": {Length, 1e-4}, "µm": {Length, 1e-4},
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
//
// Beyond the per-category preferred unit, it carries the length/angle display
// precision (decimal places) and the length display format (decimal /
// fractional / architectural). These drive presentation only; the rich
// rendering for fractional/architectural/DMS is built on top of these in #146.
type UnitsOfMeasure struct {
	prefs           map[Unit]string // category → preferred user-unit name
	lengthPrecision int             // length display precision (decimal places / fraction subdivision exponent)
	anglePrecision  int             // angle display precision (decimal places)
	lengthFormat    types.ParameterDisplayFormat
	angleFormat     AngleFormat
	// workingScale is the centimetre size of one stored (working/database) length unit —
	// the key to ADR-0042 Phase 2. A [Quantity]'s Value is in WORKING units, not always
	// centimetres: realCm = workingValue × workingScale^L (L = the unit's length exponent,
	// 1 for Length, 2 for Area, 3 for Volume). Default 1.0 ⇒ working unit IS the centimetre,
	// so every existing document and stored value is unchanged. Centring it on the document's
	// length unit (e.g. 1e-4 for µm, 1e5 for km) keeps coordinates O(1) so sub-µm and km-scale
	// parts stay well-conditioned and out of the primitive-construction underflow floor.
	workingScale float64
}

// AngleFormat is how an angle is rendered for display: as decimal degrees or as
// degrees-minutes-seconds (DMS). Length has the richer
// [types.ParameterDisplayFormat]; angle needs only this binary choice.
type AngleFormat uint8

const (
	// AngleDecimal renders an angle as decimal degrees (e.g. "30.5 deg").
	AngleDecimal AngleFormat = iota
	// AngleDMS renders an angle in degrees-minutes-seconds (e.g. "30° 30' 0\"").
	AngleDMS
)

// DefaultUnitsOfMeasure returns metric defaults (mm, degrees, kg, seconds) with
// three length / two angle display decimals and decimal length formatting.
func DefaultUnitsOfMeasure() UnitsOfMeasure {
	return UnitsOfMeasure{
		prefs: map[Unit]string{
			Length: "mm", Angle: "deg", Area: "mm^2", Volume: "mm^3", Mass: "kg", Time: "s",
		},
		lengthPrecision: 3,
		anglePrecision:  2,
		lengthFormat:    types.DisplayFormatDecimal,
		angleFormat:     AngleDecimal,
		workingScale:    1, // working unit = centimetre (back-compatible default)
	}
}

// Clone returns an independent copy whose preference edits do not touch the
// receiver's shared map — the safe basis for building an updated units object
// before storing it back on a document.
func (m UnitsOfMeasure) Clone() UnitsOfMeasure {
	prefs := make(map[Unit]string, len(m.prefs))
	for k, v := range m.prefs {
		prefs[k] = v
	}
	m.prefs = prefs
	return m
}

// LengthPrecision / AnglePrecision are the display decimal places for lengths
// and angles.
func (m UnitsOfMeasure) LengthPrecision() int { return m.lengthPrecision }
func (m UnitsOfMeasure) AnglePrecision() int  { return m.anglePrecision }

// LengthFormat is how lengths are rendered (decimal / fractional / architectural).
func (m UnitsOfMeasure) LengthFormat() types.ParameterDisplayFormat { return m.lengthFormat }

// AngleFormat is how angles are rendered (decimal degrees / DMS).
func (m UnitsOfMeasure) AngleFormat() AngleFormat { return m.angleFormat }

// SetAngleFormat sets the angle display format (decimal degrees / DMS).
func (m *UnitsOfMeasure) SetAngleFormat(f AngleFormat) { m.angleFormat = f }

// SetLengthPrecision / SetAnglePrecision set the display decimal places; a
// negative count is rejected naming the offending value.
func (m *UnitsOfMeasure) SetLengthPrecision(places int) error {
	return setPrecision(&m.lengthPrecision, places)
}

func (m *UnitsOfMeasure) SetAnglePrecision(places int) error {
	return setPrecision(&m.anglePrecision, places)
}

// setPrecision validates and assigns a display-precision decimal-place count.
func setPrecision(dst *int, places int) error {
	if places < 0 {
		return fmt.Errorf("param: display precision %d is negative; want ≥ 0", places)
	}
	*dst = places
	return nil
}

// SetLengthFormat sets the length display format (decimal / fractional /
// architectural).
func (m *UnitsOfMeasure) SetLengthFormat(f types.ParameterDisplayFormat) {
	m.lengthFormat = f
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

// WorkingScale is the centimetre size of one stored (working/database) length unit
// (ADR-0042 Phase 2). 1.0 means working coordinates are centimetres — the default.
func (m UnitsOfMeasure) WorkingScale() float64 { return m.effectiveWorkingScale() }

// effectiveWorkingScale guards a zero/negative or absent (older-struct) working scale,
// treating it as the centimetre default so a value is never divided by zero.
func (m UnitsOfMeasure) effectiveWorkingScale() float64 {
	if m.workingScale <= 0 {
		return 1
	}
	return m.workingScale
}

// WithWorkingScale returns a copy whose working unit is cmPerUnit centimetres — the
// general lever behind [UnitsOfMeasure.CenteredOnLength]. A non-positive value is
// rejected (the working unit must have a positive size).
func (m UnitsOfMeasure) WithWorkingScale(cmPerUnit float64) (UnitsOfMeasure, error) {
	if cmPerUnit <= 0 {
		return m, fmt.Errorf("param: working scale %g is not positive (cm per working unit)", cmPerUnit)
	}
	out := m.Clone()
	out.workingScale = cmPerUnit
	return out, nil
}

// SetWorkingScale sets the working unit to cmPerUnit centimetres in place (the mutating
// counterpart of [UnitsOfMeasure.WithWorkingScale]) — used by the .obk restore path. A
// non-positive value is rejected.
func (m *UnitsOfMeasure) SetWorkingScale(cmPerUnit float64) error {
	if cmPerUnit <= 0 {
		return fmt.Errorf("param: working scale %g is not positive (cm per working unit)", cmPerUnit)
	}
	m.workingScale = cmPerUnit
	return nil
}

// CenteredOnLength returns a copy whose working unit equals the named length unit, so a
// length authored in that unit is stored as an O(1) coordinate (ADR-0042 Phase 2): a µm
// document keeps sub-micron features out of the primitive-construction underflow floor,
// a km document avoids needlessly tight tolerances. The name must be a registered length
// unit.
func (m UnitsOfMeasure) CenteredOnLength(name string) (UnitsOfMeasure, error) {
	def, ok := lookupUnit(name)
	if !ok || def.category != Length {
		return m, fmt.Errorf("param: %q is not a registered length unit", name)
	}
	return m.WithWorkingScale(def.factor)
}

// safeWorkingBand brackets the centimetre size of length units that store well at the centimetre
// working scale (ADR-0042 Phase 2). Units inside it (mm … ft) keep centimetre coordinates — so
// existing documents are unchanged — while units outside (µm/nm/pm and km and larger) get a
// working unit matching the document unit, keeping coordinates O(1) and clear of the
// underflow/precision extremes.
const (
	safeWorkingBandLoCm = 1e-3
	safeWorkingBandHiCm = 1e4
)

// RecommendedWorkingScale returns the working scale (centimetres per working unit) to centre a
// fresh document on the named length unit: the unit's own size when it is extreme enough that
// centimetre storage would lose conditioning, else 1.0 (the centimetre default). ok is false for
// an unknown or non-length unit. This is the policy the document layer applies when a new
// document's length unit is chosen.
func RecommendedWorkingScale(lengthUnitName string) (scaleCm float64, ok bool) {
	def, found := lookupUnit(lengthUnitName)
	if !found || def.category != Length {
		return 1, false
	}
	if def.factor < safeWorkingBandLoCm || def.factor > safeWorkingBandHiCm {
		return def.factor, true
	}
	return 1, true
}

// wsFactor is workingScale raised to the unit's length exponent (Length¹, Area², Volume³,
// 0 otherwise) — the multiplier converting a working-unit Value to centimetres. An explicit
// switch keeps the common exponents exact (no Pow round-off).
func (m UnitsOfMeasure) wsFactor(u Unit) float64 {
	ws := m.effectiveWorkingScale()
	switch dimensions[u].l {
	case 0:
		return 1
	case 1:
		return ws
	case 2:
		return ws * ws
	case 3:
		return ws * ws * ws
	default:
		return stdmath.Pow(ws, float64(dimensions[u].l))
	}
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
		return Quantity{value * factor / m.wsFactor(defaultCat), defaultCat}, nil
	}
	def, ok := lookupUnit(unitName)
	if !ok {
		return Quantity{}, fmt.Errorf("param: unknown unit %q in %q", unitName, s)
	}
	return Quantity{value * def.factor / m.wsFactor(def.category), def.category}, nil
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
	return strconv.FormatFloat(q.Value*m.wsFactor(q.Unit)/factor, 'g', -1, 64)
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
	return q.Value * m.wsFactor(q.Unit) / factor
}

// FromPreferred builds a database-unit [Quantity] from a value given in category's
// preferred unit — the inverse of [UnitsOfMeasure.ToPreferred].
func (m UnitsOfMeasure) FromPreferred(value float64, category Unit) Quantity {
	factor, _ := m.preferredFactor(category)
	return Quantity{value * factor / m.wsFactor(category), category}
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
