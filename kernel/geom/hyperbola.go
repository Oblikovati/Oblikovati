// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Hyperbola is one branch of an infinite 3D hyperbola (the conic a plane parallel to a cone's
// axis cuts from it — M2 Phase-1 follow-up, Oblikovati/Oblikovati#1372). It lies in the plane
// through Center spanned by the orthonormal TransverseAxis (toward the branch's opening, the
// real axis) and ConjugateAxis (the imaginary axis). Parameterized by the hyperbolic angle θ:
//
//	P(θ) = Center + A·cosh(θ)·TransverseAxis + B·sinh(θ)·ConjugateAxis
//
// so θ=0 is the branch vertex (Center + A·TransverseAxis) and |θ|→∞ runs out the two arms. Like
// [Line] it is unbounded (Domain ±Inf, parameter t = θ directly); the bounded edge a face stores
// is a [HyperbolicArc]. A and B are the transverse and conjugate semi-axes.
type Hyperbola struct {
	Center         math.Point3
	TransverseAxis math.UnitVector3
	ConjugateAxis  math.UnitVector3
	A, B           float64
}

// NewHyperbola builds a hyperbola branch. transverse and conjugate are normalized, and conjugate
// is re-orthogonalized against transverse (its component along transverse is dropped). Errors on a
// zero axis, a non-positive semi-axis, or axes that are parallel.
func NewHyperbola(center math.Point3, transverse, conjugate math.Vector3, a, b float64) (Hyperbola, error) {
	if a <= 0 || b <= 0 {
		return Hyperbola{}, fmt.Errorf("geom: hyperbola semi-axes (%g, %g) must be positive", a, b)
	}
	tu, err := math.UnitVector3FromVector(transverse)
	if err != nil {
		return Hyperbola{}, fmt.Errorf("geom: hyperbola transverse axis %v is zero", transverse)
	}
	cv := conjugate.Sub(tu.AsVector().Scale(conjugate.Dot(tu.AsVector())))
	cu, err := math.UnitVector3FromVector(cv)
	if err != nil {
		return Hyperbola{}, fmt.Errorf("geom: hyperbola conjugate axis %v is parallel to transverse %v", conjugate, transverse)
	}
	return Hyperbola{Center: center, TransverseAxis: tu, ConjugateAxis: cu, A: a, B: b}, nil
}

// PointAt returns the branch point at hyperbolic angle θ (the parameter t is θ).
func (h Hyperbola) PointAt(theta float64) math.Point3 {
	return hyperbolaPoint(h.Center, h.TransverseAxis.AsVector(), h.ConjugateAxis.AsVector(), h.A, h.B, theta)
}

// TangentAt returns dP/dθ = A·sinh(θ)·TransverseAxis + B·cosh(θ)·ConjugateAxis.
func (h Hyperbola) TangentAt(theta float64) math.Vector3 {
	return hyperbolaTangent(h.TransverseAxis.AsVector(), h.ConjugateAxis.AsVector(), h.A, h.B, theta)
}

// Domain returns the full real line, as a Hyperbola branch is unbounded.
func (h Hyperbola) Domain() (lo, hi float64) { return stdmath.Inf(-1), stdmath.Inf(1) }

// Arc restricts the branch to [theta0, theta1], the bounded curve stored on a trimmed face edge.
func (h Hyperbola) Arc(theta0, theta1 float64) HyperbolicArc {
	return HyperbolicArc{Center: h.Center, TransverseAxis: h.TransverseAxis, ConjugateAxis: h.ConjugateAxis,
		A: h.A, B: h.B, Theta0: theta0, Theta1: theta1}
}

// HyperbolicArc is a bounded segment of a [Hyperbola] branch, parameterized t∈[0,1] mapping to
// θ = Theta0 + t·(Theta1−Theta0) — the edge curve a trimmed cone face stores (as LineSegment is to
// Line). Theta0 may exceed Theta1 to run the arc in the opposite direction.
type HyperbolicArc struct {
	Center         math.Point3
	TransverseAxis math.UnitVector3
	ConjugateAxis  math.UnitVector3
	A, B           float64
	Theta0, Theta1 float64
}

// PointAt returns the point at parameter t∈[0,1].
func (h HyperbolicArc) PointAt(t float64) math.Point3 {
	theta := h.Theta0 + t*(h.Theta1-h.Theta0)
	return hyperbolaPoint(h.Center, h.TransverseAxis.AsVector(), h.ConjugateAxis.AsVector(), h.A, h.B, theta)
}

// TangentAt returns dP/dt (chain rule: dP/dθ scaled by the θ-span).
func (h HyperbolicArc) TangentAt(t float64) math.Vector3 {
	theta := h.Theta0 + t*(h.Theta1-h.Theta0)
	return hyperbolaTangent(h.TransverseAxis.AsVector(), h.ConjugateAxis.AsVector(), h.A, h.B, theta).Scale(math.Scalar(h.Theta1 - h.Theta0))
}

// Domain returns [0, 1].
func (h HyperbolicArc) Domain() (lo, hi float64) { return 0, 1 }

// hyperbolaPoint evaluates Center + A·cosh(θ)·û + B·sinh(θ)·v̂.
func hyperbolaPoint(center math.Point3, transverse, conjugate math.Vector3, a, b, theta float64) math.Point3 {
	return center.TranslateBy(transverse.Scale(math.Scalar(a * stdmath.Cosh(theta))).
		Add(conjugate.Scale(math.Scalar(b * stdmath.Sinh(theta)))))
}

// hyperbolaTangent returns dP/dθ = A·sinh(θ)·û + B·cosh(θ)·v̂.
func hyperbolaTangent(transverse, conjugate math.Vector3, a, b, theta float64) math.Vector3 {
	return transverse.Scale(math.Scalar(a * stdmath.Sinh(theta))).
		Add(conjugate.Scale(math.Scalar(b * stdmath.Cosh(theta))))
}
