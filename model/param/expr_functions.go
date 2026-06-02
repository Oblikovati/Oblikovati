// SPDX-License-Identifier: GPL-2.0-only

package param

import (
	"fmt"
	stdmath "math"
)

// builtin is a unit-aware expression function: it validates argument count and
// dimensions, then returns a dimensioned result.
type builtin func(args []Quantity) (Quantity, error)

// functions is the standard expression function library. Trig consumes angle
// (database unit: radians) and yields a unitless ratio; inverse trig does the
// reverse; sqrt halves dimension exponents; abs/min/max/floor/ceil/round
// preserve the unit; ln/log/exp require unitless arguments.
var functions = map[string]builtin{
	"sin":   trig(stdmath.Sin),
	"cos":   trig(stdmath.Cos),
	"tan":   trig(stdmath.Tan),
	"asin":  inverseTrig(stdmath.Asin),
	"acos":  inverseTrig(stdmath.Acos),
	"atan":  inverseTrig(stdmath.Atan),
	"sqrt":  sqrtFn,
	"abs":   unaryPreserving(stdmath.Abs),
	"floor": unaryPreserving(stdmath.Floor),
	"ceil":  unaryPreserving(stdmath.Ceil),
	"round": unaryPreserving(stdmath.Round),
	"ln":    unitlessFn(stdmath.Log),
	"log":   unitlessFn(stdmath.Log10),
	"exp":   unitlessFn(stdmath.Exp),
	"min":   reduceFn(stdmath.Min),
	"max":   reduceFn(stdmath.Max),
	"atan2": atan2Fn,
}

// trig wraps a sin/cos/tan function: one angle (or unitless) argument in, a
// unitless ratio out.
func trig(f func(float64) float64) builtin {
	return func(args []Quantity) (Quantity, error) {
		if err := requireArgs("trig", args, 1); err != nil {
			return Quantity{}, err
		}
		if u := args[0].Unit; u != Angle && u != Unitless {
			return Quantity{}, fmt.Errorf("param: trig function needs an angle, got %s", u)
		}
		return Scalar(f(args[0].Value)), nil
	}
}

// inverseTrig wraps asin/acos/atan: one unitless argument in, an angle out.
func inverseTrig(f func(float64) float64) builtin {
	return func(args []Quantity) (Quantity, error) {
		if err := requireArgs("inverse trig", args, 1); err != nil {
			return Quantity{}, err
		}
		if err := requireUnitless("inverse trig", args[0]); err != nil {
			return Quantity{}, err
		}
		return Q(f(args[0].Value), Angle), nil
	}
}

// unaryPreserving wraps abs/floor/ceil/round: the unit passes through unchanged.
func unaryPreserving(f func(float64) float64) builtin {
	return func(args []Quantity) (Quantity, error) {
		if err := requireArgs("function", args, 1); err != nil {
			return Quantity{}, err
		}
		return Quantity{f(args[0].Value), args[0].Unit}, nil
	}
}

// unitlessFn wraps ln/log/exp, which are defined only on unitless values.
func unitlessFn(f func(float64) float64) builtin {
	return func(args []Quantity) (Quantity, error) {
		if err := requireArgs("function", args, 1); err != nil {
			return Quantity{}, err
		}
		if err := requireUnitless("function", args[0]); err != nil {
			return Quantity{}, err
		}
		return Scalar(f(args[0].Value)), nil
	}
}

// reduceFn wraps min/max over two or more same-unit arguments.
func reduceFn(f func(a, b float64) float64) builtin {
	return func(args []Quantity) (Quantity, error) {
		if len(args) < 2 {
			return Quantity{}, fmt.Errorf("param: min/max needs >= 2 arguments, got %d", len(args))
		}
		acc := args[0]
		for _, a := range args[1:] {
			if a.Unit != acc.Unit {
				return Quantity{}, &DimensionError{Op: "min/max", Left: acc.Unit, Right: a.Unit}
			}
			acc.Value = f(acc.Value, a.Value)
		}
		return acc, nil
	}
}

// sqrtFn returns the square root, halving the dimension exponents (so
// sqrt(area)=length). It errors when an exponent is odd.
func sqrtFn(args []Quantity) (Quantity, error) {
	if err := requireArgs("sqrt", args, 1); err != nil {
		return Quantity{}, err
	}
	d, ok := dimensionOf(args[0].Unit)
	if !ok {
		return Quantity{}, fmt.Errorf("param: sqrt of non-arithmetic %s", args[0].Unit)
	}
	if d.l%2 != 0 || d.a%2 != 0 || d.m%2 != 0 || d.t%2 != 0 {
		return Quantity{}, fmt.Errorf("param: sqrt of %s has no representable unit", args[0].Unit)
	}
	unit, _ := unitForDimension(dimension{d.l / 2, d.a / 2, d.m / 2, d.t / 2})
	return Quantity{stdmath.Sqrt(args[0].Value), unit}, nil
}

// atan2Fn returns the angle of (y, x); both arguments must share a unit.
func atan2Fn(args []Quantity) (Quantity, error) {
	if err := requireArgs("atan2", args, 2); err != nil {
		return Quantity{}, err
	}
	if args[0].Unit != args[1].Unit {
		return Quantity{}, &DimensionError{Op: "atan2", Left: args[0].Unit, Right: args[1].Unit}
	}
	return Q(stdmath.Atan2(args[0].Value, args[1].Value), Angle), nil
}

func requireArgs(name string, args []Quantity, n int) error {
	if len(args) != n {
		return fmt.Errorf("param: %s takes %d argument(s), got %d", name, n, len(args))
	}
	return nil
}

func requireUnitless(name string, q Quantity) error {
	if q.Unit != Unitless {
		return fmt.Errorf("param: %s needs a unitless argument, got %s", name, q.Unit)
	}
	return nil
}
