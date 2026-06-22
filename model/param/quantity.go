// SPDX-License-Identifier: GPL-2.0-only

package param

import "fmt"

// Quantity is a dimensioned value: its Value is in WORKING (database) units for the
// given Unit. The working length unit is the centimetre by default, so historically a
// length Value was simply centimetres; under ADR-0042 Phase 2 a document may centre its
// working unit elsewhere (µm, km) for conditioning, in which case realCm = Value ×
// workingScale^L. Either way the conversion to a named user unit happens only at the
// parse/display boundary ([UnitsOfMeasure]); the kernel math is unit-agnostic and operates
// on the raw Value. Quantity is an immutable value type; arithmetic returns new quantities
// and enforces dimensional consistency.
type Quantity struct {
	Value float64
	Unit  Unit
}

// Q constructs a Quantity in database units.
func Q(value float64, u Unit) Quantity {
	return Quantity{Value: value, Unit: u}
}

// Scalar constructs a unitless quantity.
func Scalar(value float64) Quantity {
	return Quantity{Value: value, Unit: Unitless}
}

// Add returns q + o. It errors unless both operands share the same unit, naming
// the mismatch — adding a length to an angle is meaningless.
func (q Quantity) Add(o Quantity) (Quantity, error) {
	if q.Unit != o.Unit {
		return Quantity{}, &DimensionError{Op: "+", Left: q.Unit, Right: o.Unit}
	}
	return Quantity{q.Value + o.Value, q.Unit}, nil
}

// Sub returns q − o, with the same same-unit requirement as [Quantity.Add].
func (q Quantity) Sub(o Quantity) (Quantity, error) {
	if q.Unit != o.Unit {
		return Quantity{}, &DimensionError{Op: "-", Left: q.Unit, Right: o.Unit}
	}
	return Quantity{q.Value - o.Value, q.Unit}, nil
}

// Mul returns q · o, combining dimensions (Length·Length → Area). It errors
// when either operand is non-arithmetic or the product has no named unit.
func (q Quantity) Mul(o Quantity) (Quantity, error) {
	return q.combine(o, "*", func(a, b dimension) dimension {
		return dimension{a.l + b.l, a.a + b.a, a.m + b.m, a.t + b.t}
	}, q.Value*o.Value)
}

// Div returns q / o, subtracting dimensions (Volume/Area → Length). It errors
// on a zero divisor, a non-arithmetic operand, or an unnamed result dimension.
func (q Quantity) Div(o Quantity) (Quantity, error) {
	if o.Value == 0 {
		return Quantity{}, fmt.Errorf("param: division by zero (%g %s / 0 %s)", q.Value, q.Unit, o.Unit)
	}
	return q.combine(o, "/", func(a, b dimension) dimension {
		return dimension{a.l - b.l, a.a - b.a, a.m - b.m, a.t - b.t}
	}, q.Value/o.Value)
}

// combine applies a dimension-exponent operation and resolves the result back
// to a named unit, factoring the shared validation of Mul and Div.
func (q Quantity) combine(o Quantity, op string, exps func(a, b dimension) dimension, value float64) (Quantity, error) {
	da, okA := dimensionOf(q.Unit)
	db, okB := dimensionOf(o.Unit)
	if !okA || !okB {
		return Quantity{}, &DimensionError{Op: op, Left: q.Unit, Right: o.Unit}
	}
	unit, ok := unitForDimension(exps(da, db))
	if !ok {
		return Quantity{}, fmt.Errorf("param: %s of %s and %s has no representable unit", op, q.Unit, o.Unit)
	}
	return Quantity{value, unit}, nil
}

// Negate returns −q (unit preserved).
func (q Quantity) Negate() Quantity {
	return Quantity{-q.Value, q.Unit}
}

// DimensionError reports an operation between dimensionally incompatible
// quantities. It names the operator and both units, per the project's
// exception-message convention.
type DimensionError struct {
	Op          string
	Left, Right Unit
}

func (e *DimensionError) Error() string {
	return fmt.Sprintf("param: cannot apply %q to incompatible units %s and %s", e.Op, e.Left, e.Right)
}
