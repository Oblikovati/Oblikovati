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

// Tolerance is an engineering tolerance: a flavor plus the deviation band from
// the nominal value (Upper, Lower, database units). The zero Tolerance means
// "standard/default tolerance, no explicit band" — [Tolerance.Kind] maps the
// zero Type onto [types.ToleranceDefault] so existing `t != Tolerance{}`
// has-explicit-tolerance checks keep working. Which value within the band the
// model consumes is the parameter's [Parameter.ModelValueType], per the
// reference API's split between Tolerance.ToleranceType and
// Parameter.ModelValueType (Oblikovati#607).
type Tolerance struct {
	Type  ToleranceType
	Upper float64
	Lower float64
}

// Kind returns the tolerance flavor, mapping the zero value to ToleranceDefault.
func (t Tolerance) Kind() ToleranceType {
	if t.Type == 0 {
		return types.ToleranceDefault
	}
	return t.Type
}

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
