// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/math"
	"oblikovati.org/solve/ad"
)

// This file is the bridge between a constraint's residual formula and the solver's
// exact Jacobian (Oblikovati/Oblikovati#1417). Each constraint writes its residual ONCE
// as a function over dual numbers (residualAD); adResiduals derives the float residual
// values from it (the value-only path), and adPartials derives the exact per-variable
// derivatives — neither perturbs the live geometry, replacing the solver's old
// finite-difference Jacobian on the sketch path.

// adFunc is a constraint's residual written over dual numbers: it receives one seeded
// [ad.Number] per variable, in the constraint's Variables() order, and returns one dual
// per residual, in Residuals() order.
type adFunc func(v []ad.Number) []ad.Number

// adFunc1 is a single-valued dual function — a dimension's measured quantity written over
// dual numbers (one seeded number per variable). Its derivative is the dimension's row of
// the Jacobian.
type adFunc1 func(v []ad.Number) ad.Number

// adMeasureValue evaluates a single-valued dual measure at the current variable values —
// the float quantity a dimension reports, derived from its dual measure so the value and
// its derivative never diverge.
func adMeasureValue(vars []*math.Scalar, f adFunc1) float64 {
	return f(ad.Consts(scalarValues(vars))).Val()
}

// adResiduals evaluates f at the current variable values as constants — the value-only
// path, which allocates no gradients.
func adResiduals(vars []*math.Scalar, f adFunc) []float64 {
	out := f(ad.Consts(scalarValues(vars)))
	res := make([]float64, len(out))
	for i, n := range out {
		res[i] = n.Val()
	}
	return res
}

// adPartials evaluates f with the variables seeded, returning ∂residualᵢ/∂varⱼ. A
// residual that does not depend on any variable yields a nil gradient, reported as a
// zero row of the right width.
func adPartials(vars []*math.Scalar, f adFunc) [][]float64 {
	out := f(ad.Seed(scalarValues(vars)))
	jac := make([][]float64, len(out))
	for i, n := range out {
		if g := n.Grad(); g != nil {
			jac[i] = g
		} else {
			jac[i] = make([]float64, len(vars))
		}
	}
	return jac
}

// scalarValues snapshots the current values behind a variable-pointer slice (a read; the
// live scalars are never written).
func scalarValues(vars []*math.Scalar) []float64 {
	vals := make([]float64, len(vars))
	for i, v := range vars {
		vals[i] = float64(*v)
	}
	return vals
}

// adZeroVec2 is the dual zero vector (a constant, no gradient).
func adZeroVec2() ad.Vec2 { return ad.V2(ad.Const(0), ad.Const(0)) }

// The orientation residuals below are normalised so their magnitude is in LENGTH units
// (perpendicular distance / projection) or DIMENSIONLESS (sine / cosine of an angle),
// never area (|line|·|offset|). An un-normalised cross/dot residual shrinks with the
// line's length, so a short segment yields a tiny Jacobian row that matrixRank drops —
// mis-reporting the constraint as redundant and falsifying the DOF / over-constrained
// signal (Oblikovati/Oblikovati#1418). Normalising keeps the Jacobian rows O(1)
// regardless of segment length, so classification matches the geometric truth and a long
// vs short geometrically-identical sketch classify the same.

// adSignedPerpDistance returns the signed perpendicular distance of offset w from a line
// of direction dir: (dir × w)/|dir|. A degenerate (zero-length) direction falls back to
// |w| (distance to the line's anchor), keeping the residual finite and in length units.
func adSignedPerpDistance(dir, w ad.Vec2) ad.Number {
	length := dir.Length()
	if length.Val() < math.DefaultTolerance {
		return w.Length()
	}
	return dir.Cross(w).Div(length)
}

// adSineAngle returns the sine of the angle between two directions, (d1 × d2)/(|d1||d2|) —
// the scale-invariant parallelism residual. A degenerate direction (no orientation to be
// parallel to) falls back to the raw cross product (which is itself ≈ 0).
func adSineAngle(d1, d2 ad.Vec2) ad.Number {
	denom := d1.Length().Mul(d2.Length())
	if denom.Val() < math.DefaultTolerance {
		return d1.Cross(d2)
	}
	return d1.Cross(d2).Div(denom)
}

// adCosAngle returns the cosine of the angle between two directions, (d1 · d2)/(|d1||d2|) —
// the scale-invariant perpendicularity residual (zero at a right angle). Degenerate
// directions fall back to the raw dot product.
func adCosAngle(d1, d2 ad.Vec2) ad.Number {
	denom := d1.Length().Mul(d2.Length())
	if denom.Val() < math.DefaultTolerance {
		return d1.Dot(d2)
	}
	return d1.Dot(d2).Div(denom)
}

// adProjectionAlong returns the component of w along dir, (w · dir)/|dir| — a length
// independent of dir's arbitrary representation length. A degenerate dir falls back to the
// raw dot product.
func adProjectionAlong(w, dir ad.Vec2) ad.Number {
	length := dir.Length()
	if length.Val() < math.DefaultTolerance {
		return w.Dot(dir)
	}
	return w.Dot(dir).Div(length)
}

// adPointSymmetry returns the two residuals that hold points a and b symmetric across the
// mirror line la→lb: the midpoint of a,b lies on the line (signed perpendicular distance
// zero) and the a→b segment is perpendicular to it (projection along the line zero). Both
// are normalised by the mirror line's length so they are independent of its arbitrary
// representation (#1418). Shared by the point-pair, line and circular symmetry constraints
// (#1574, #1870) so entity symmetry reuses the one grounded equation.
func adPointSymmetry(a, b, la, lb ad.Vec2) []ad.Number {
	dir := lb.Sub(la)
	mid := a.Add(b).Scale(0.5)
	return []ad.Number{adSignedPerpDistance(dir, mid.Sub(la)), adProjectionAlong(b.Sub(a), dir)}
}

// adUnit2 returns v scaled to unit length, or the zero vector when v is (near) zero —
// the dual twin of unit2, keeping a degenerate direction from producing NaNs.
func adUnit2(v ad.Vec2) ad.Vec2 {
	l := v.Length()
	if l.Val() < math.DefaultTolerance {
		return adZeroVec2()
	}
	return v.MulN(ad.Const(1).Div(l))
}

// adConstResiduals returns n constant-valued residuals — the dual form of a constraint's
// "cannot be satisfied" fallback (e.g. a smooth join whose endpoints are not curve ends).
func adConstResiduals(n int, val float64) []ad.Number {
	out := make([]ad.Number, n)
	for i := range out {
		out[i] = ad.Const(val)
	}
	return out
}

// adV3 reads a 3D point from three consecutive seeded variables starting at i.
func adV3(v []ad.Number, i int) ad.Vec3 { return ad.V3(v[i], v[i+1], v[i+2]) }

// adConstVec3 lifts a fixed (non-DOF) vector — a world axis or plane normal — into a
// constant dual vector.
func adConstVec3(n math.Vector3) ad.Vec3 {
	return ad.V3(ad.Const(float64(n.X)), ad.Const(float64(n.Y)), ad.Const(float64(n.Z)))
}

// adZeroVec3 is the dual zero vector (a constant, no gradient).
func adZeroVec3() ad.Vec3 { return ad.V3(ad.Const(0), ad.Const(0), ad.Const(0)) }

// adUnit3 returns v scaled to unit length, or the zero vector when v is (near) zero — the
// dual twin of unit3.
func adUnit3(v ad.Vec3) ad.Vec3 {
	l := v.Length()
	if l.Val() < math.DefaultTolerance {
		return adZeroVec3()
	}
	return v.MulN(ad.Const(1).Div(l))
}

// circularFrameAD returns a circular curve's center and radius as duals, read from the
// segment of the seeded variable row that holds the curve's circularVars (starting at
// off), plus the number of variables it consumed — so a constraint over two curves can
// place the second curve's frame after the first. It mirrors circularVars/CenterPoint/
// CurveRadius in dual arithmetic.
type circularAD interface {
	circularFrameAD(v []ad.Number, off int) (center ad.Vec2, radius ad.Number, consumed int)
}

// A circle's frame is its center and its stored radius DOF (three variables).
func (c *Circle) circularFrameAD(v []ad.Number, off int) (ad.Vec2, ad.Number, int) {
	return ad.V2(v[off], v[off+1]), v[off+2], 3
}

// An arc has no radius DOF: its radius is |center − start|, so the frame is computed from
// the center and start points (four variables).
func (a *Arc) circularFrameAD(v []ad.Number, off int) (ad.Vec2, ad.Number, int) {
	center := ad.V2(v[off], v[off+1])
	start := ad.V2(v[off+2], v[off+3])
	return center, start.Sub(center).Length(), 4
}
