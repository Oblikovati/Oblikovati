// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"

	"oblikovati.org/api/types"
)

// ToleranceType is the engineering-tolerance flavor. The type, its frozen ids,
// String, and ParseToleranceType live in the Apache-2.0 contract
// ([types.ToleranceType]); this alias keeps param spellings working (ADR-0018).
type ToleranceType = types.ToleranceType

// ModelValueType selects which value within the tolerance band the model
// consumes. Aliased from the contract; the historical param.Nominal /
// param.Upper / param.Lower / param.Median spellings are preserved below.
type ModelValueType = types.ModelValueType

const (
	Nominal = types.ModelValueNominal
	Upper   = types.ModelValueUpper
	Lower   = types.ModelValueLower
	Median  = types.ModelValueMedian
)

// ParameterDisplayFormat is how a parameter's numeric value is rendered
// (decimal, fractional, architectural). Aliased from the contract.
type ParameterDisplayFormat = types.ParameterDisplayFormat

const (
	DisplayFormatDecimal       = types.DisplayFormatDecimal
	DisplayFormatFractional    = types.DisplayFormatFractional
	DisplayFormatArchitectural = types.DisplayFormatArchitectural
)

// Tolerance is an engineering tolerance: a flavor plus the deviation band from the nominal
// value (Upper, Lower, database units). The canonical definition and [Tolerance.Kind] (which
// maps the zero Type onto [types.ToleranceDefault] so existing `t != Tolerance{}`
// has-explicit-tolerance checks keep working) live in the Apache-2.0 contract
// ([types.Tolerance]); this alias keeps param spellings working and lets the contract's
// Parameter.Tolerance() return it directly (ADR-0018, M39-F06). Which value within the band the
// model consumes is the parameter's [Parameter.ModelValueType], per the reference API's split
// between Tolerance.ToleranceType and Parameter.ModelValueType (Oblikovati#607).
type Tolerance = types.Tolerance

// SetToleranceDefault reverts the parameter to the standard/default tolerance
// (no explicit band; the model value follows the nominal).
func (p *Parameter) SetToleranceDefault() error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	p.tol = Tolerance{}
	return nil
}

// SetToleranceDeviation sets an asymmetric deviation band: upper and lower are
// deviations from nominal in database units, upper ≥ lower.
func (p *Parameter) SetToleranceDeviation(upper, lower float64) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	if upper < lower {
		return fmt.Errorf("param: deviation upper %g < lower %g for %q; want upper ≥ lower", upper, lower, p.name)
	}
	p.tol = Tolerance{Type: types.ToleranceDeviation, Upper: upper, Lower: lower}
	return nil
}

// SetToleranceSymmetric sets a symmetric ± band: band ≥ 0, in database units.
func (p *Parameter) SetToleranceSymmetric(band float64) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	if band < 0 {
		return fmt.Errorf("param: symmetric tolerance band %g for %q must be ≥ 0", band, p.name)
	}
	p.tol = Tolerance{Type: types.ToleranceSymmetric, Upper: band, Lower: -band}
	return nil
}

// SetToleranceLimits sets a limits tolerance from absolute limit values (in
// database units): the band is stored as deviations from the current nominal.
func (p *Parameter) SetToleranceLimits(upperLimit, lowerLimit float64) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	if upperLimit < lowerLimit {
		return fmt.Errorf("param: limits upper %g < lower %g for %q; want upper ≥ lower", upperLimit, lowerLimit, p.name)
	}
	nominal := p.value.Value
	p.tol = Tolerance{Type: types.ToleranceLimitsStacked, Upper: upperLimit - nominal, Lower: lowerLimit - nominal}
	return nil
}

// SetToleranceFits sets an ISO 286 limits-and-fits tolerance (#1848). The band is
// resolved from the nominal value and the class governing the dimensioned feature
// — the hole class when present, else the shaft class (hole-basis convention) —
// and both class strings are recorded for the fit annotation. The tolerance type
// is kLimitsFitsStacked. nominalMM must be > 0 and match the parameter's own
// nominal in millimetres; the caller passes it because the band scales with size.
func (p *Parameter) SetToleranceFits(hole, shaft string) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	if p.value.Unit != Length {
		return fmt.Errorf("param: fits tolerance for %q needs a length parameter (ISO limits-and-fits apply to linear sizes)", p.name)
	}
	governing := hole
	if governing == "" {
		governing = shaft
	}
	if governing == "" {
		return fmt.Errorf("param: fits tolerance for %q needs a hole or shaft class (e.g. hole \"H7\")", p.name)
	}
	upperMM, lowerMM, err := ISOFitBand(p.value.Value*millimetresPerDatabaseLength, governing)
	if err != nil {
		return err
	}
	// ISOFitBand works in millimetres; the stored band is in database units (cm).
	p.tol = Tolerance{
		Type:          types.ToleranceLimitsFitsStacked,
		Upper:         upperMM / millimetresPerDatabaseLength,
		Lower:         lowerMM / millimetresPerDatabaseLength,
		HoleTolerance: hole, ShaftTolerance: shaft,
	}
	return nil
}

// SetToleranceBasic marks the value as a basic (boxed) dimension: exact nominal,
// no tolerance band (kBasicTolerance, #1848).
func (p *Parameter) SetToleranceBasic() error {
	return p.setBandlessTolerance(types.ToleranceBasic)
}

// SetToleranceReference marks the value as a reference (parenthesized) dimension:
// informational, no tolerance band (kReferenceTolerance, #1848).
func (p *Parameter) SetToleranceReference() error {
	return p.setBandlessTolerance(types.ToleranceReference)
}

// setBandlessTolerance sets a tolerance flavor that carries no deviation band.
func (p *Parameter) setBandlessTolerance(t ToleranceType) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	p.tol = Tolerance{Type: t}
	return nil
}

// SetToleranceMinMax marks the value as a MIN or MAX tolerance (no band).
func (p *Parameter) SetToleranceMinMax(t ToleranceType) error {
	if err := p.requireNumericTolerance(); err != nil {
		return err
	}
	if t != types.ToleranceMin && t != types.ToleranceMax {
		return fmt.Errorf("param: SetToleranceMinMax(%s) for %q; want min or max", t, p.name)
	}
	p.tol = Tolerance{Type: t}
	return nil
}

// requireNumericTolerance rejects tolerance edits on text and true/false
// parameters, which carry no tolerance.
func (p *Parameter) requireNumericTolerance() error {
	if !p.IsNumeric() {
		return fmt.Errorf("param: %q is a %s parameter; only numeric parameters carry a tolerance", p.name, p.value.Unit)
	}
	return nil
}
