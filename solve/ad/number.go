// SPDX-License-Identifier: GPL-2.0-only

// Package ad is a small forward-mode automatic-differentiation layer (vector
// duals) the constraint solver uses to obtain EXACT residual partial derivatives
// without finite differencing and without perturbing live geometry (Oblikovati/
// Oblikovati#1417). A [Number] carries a value and its gradient with respect to a
// fixed set of seeded variables; arithmetic propagates both, so evaluating a
// residual formula once over seeded numbers yields the residual AND the row of the
// Jacobian for free, to machine precision.
//
// It is the project-owned alternative to hand-coding (and re-deriving) a Jacobian
// per constraint — the approach SolveSpace/planegcs take by hand, done here by
// construction so a constraint's derivative can never drift from its residual.
//
//	v := ad.Seed([]float64{3, 4})        // two variables, seeded e0,e1
//	r := v[0].Hypot(v[1])                // r.Val()==5, r.Grad()=={0.6, 0.8}
package ad

import stdmath "math"

// Number is a forward-mode dual: a value and its gradient w.r.t. the seeded
// variables. A nil grad means a constant (all-zero gradient) — kept nil so the
// value-only path allocates nothing.
type Number struct {
	val  float64
	grad []float64
}

// Const returns a constant (zero-gradient) number.
func Const(v float64) Number { return Number{val: v} }

// Var returns variable i of an n-variable system seeded at value v: its gradient
// is the unit vector eᵢ. Panics on an out-of-range index so a mis-wired constraint
// fails loudly rather than silently producing a wrong derivative.
func Var(v float64, i, n int) Number {
	if i < 0 || i >= n {
		panic(adIndexError(i, n))
	}
	g := make([]float64, n)
	g[i] = 1
	return Number{val: v, grad: g}
}

// Seed returns the values as the variables of an len(vals)-variable system, each
// seeded with its own unit gradient — the input row a residual formula consumes.
func Seed(vals []float64) []Number {
	out := make([]Number, len(vals))
	for i, v := range vals {
		out[i] = Var(v, i, len(vals))
	}
	return out
}

// Consts returns the values as constants (no gradient) — the input row for a
// value-only (no-derivative) evaluation.
func Consts(vals []float64) []Number {
	out := make([]Number, len(vals))
	for i, v := range vals {
		out[i] = Number{val: v}
	}
	return out
}

// Val returns the number's value.
func (a Number) Val() float64 { return a.val }

// Grad returns the number's gradient (length = the seeded variable count), or nil
// for a constant.
func (a Number) Grad() []float64 { return a.grad }

// Add returns a+b.
func (a Number) Add(b Number) Number {
	return Number{val: a.val + b.val, grad: combine(a.grad, b.grad, 1, 1)}
}

// Sub returns a−b.
func (a Number) Sub(b Number) Number {
	return Number{val: a.val - b.val, grad: combine(a.grad, b.grad, 1, -1)}
}

// Mul returns a·b (product rule: d(ab)=b·da+a·db).
func (a Number) Mul(b Number) Number {
	return Number{val: a.val * b.val, grad: combine(a.grad, b.grad, b.val, a.val)}
}

// Div returns a/b (quotient rule). It does not guard b==0; a residual that can
// divide by zero must check its denominator before calling (as the float formulas
// already do for degenerate geometry).
func (a Number) Div(b Number) Number {
	inv := 1 / b.val
	return Number{val: a.val * inv, grad: combine(a.grad, b.grad, inv, -a.val*inv*inv)}
}

// Neg returns −a.
func (a Number) Neg() Number { return Number{val: -a.val, grad: combine(a.grad, nil, -1, 0)} }

// AddConst returns a+c (c constant); Scale returns a·c.
func (a Number) AddConst(c float64) Number { return Number{val: a.val + c, grad: a.grad} }
func (a Number) Scale(c float64) Number {
	return Number{val: a.val * c, grad: combine(a.grad, nil, c, 0)}
}

// Sqrt returns √a (derivative 1/(2√a)); the gradient is zero at a==0 where √ is not
// differentiable, which keeps a degenerate (coincident) term from producing NaNs.
func (a Number) Sqrt() Number {
	s := stdmath.Sqrt(a.val)
	d := 0.0
	if s != 0 {
		d = 0.5 / s
	}
	return Number{val: s, grad: combine(a.grad, nil, d, 0)}
}

// Hypot returns √(a²+b²) — the Euclidean length used by distances and radii. Its
// gradient is (a·da+b·db)/hypot, zero at the origin (the non-differentiable point).
func (a Number) Hypot(b Number) Number {
	h := stdmath.Hypot(a.val, b.val)
	if h == 0 {
		return Number{val: 0, grad: combine(a.grad, b.grad, 0, 0)}
	}
	return Number{val: h, grad: combine(a.grad, b.grad, a.val/h, b.val/h)}
}

// Abs returns |a|; its derivative is sign(a) (and zero at a==0, the kink). The
// tangent constraint and tangent-distance dimension take an absolute distance.
func (a Number) Abs() Number {
	switch {
	case a.val > 0:
		return a
	case a.val < 0:
		return a.Neg()
	default:
		return Number{val: 0, grad: combine(a.grad, nil, 0, 0)}
	}
}

// Atan2 returns atan2(a, b) treating a as y and b as x — the angle measure. Its
// gradient is (x·dy − y·dx)/(x²+y²); zero at the origin where the angle is
// undefined.
func (a Number) Atan2(b Number) Number {
	y, x := a.val, b.val
	den := x*x + y*y
	t := stdmath.Atan2(y, x)
	if den == 0 {
		return Number{val: t, grad: combine(a.grad, b.grad, 0, 0)}
	}
	return Number{val: t, grad: combine(a.grad, b.grad, x/den, -y/den)}
}

// Sin and Cos are the trigonometric primitives (used by 3D angle relations).
func (a Number) Sin() Number {
	return Number{val: stdmath.Sin(a.val), grad: combine(a.grad, nil, stdmath.Cos(a.val), 0)}
}
func (a Number) Cos() Number {
	return Number{val: stdmath.Cos(a.val), grad: combine(a.grad, nil, -stdmath.Sin(a.val), 0)}
}

// combine returns ca·ga + cb·gb, the chain-rule accumulation of two operands'
// gradients scaled by their partials. It is the single place gradient vectors are
// allocated, so a constant (nil) operand contributes nothing and the value-only
// path stays allocation-free.
func combine(ga, gb []float64, ca, cb float64) []float64 {
	n := len(ga)
	if len(gb) > n {
		n = len(gb)
	}
	if n == 0 {
		return nil
	}
	out := make([]float64, n)
	for i := range ga {
		out[i] += ca * ga[i]
	}
	for i := range gb {
		out[i] += cb * gb[i]
	}
	return out
}
