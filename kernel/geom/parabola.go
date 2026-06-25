// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// Parabola is an infinite 3D parabola (the conic a plane PARALLEL to a cone generator cuts from it —
// the boundary tilt between the elliptic and hyperbolic sections, M2 Phase-1 follow-up,
// Oblikovati/Oblikovati#1375). It lies in the plane through Vertex spanned by the orthonormal AxisDir
// (the axis of symmetry, toward which the curve opens) and CrossDir. Parameterized by the cross
// coordinate t (the standard y = x²/4f form, x along CrossDir, y along AxisDir):
//
//	P(t) = Vertex + t·CrossDir + (t²/(4·Focal))·AxisDir
//
// so t=0 is the vertex and |t|→∞ runs out the two arms. Like [Hyperbola] it is unbounded (Domain
// ±Inf, parameter t directly); the bounded edge a face stores is a [ParabolicArc]. Focal is the focal
// length (the distance from vertex to focus).
type Parabola struct {
	Vertex   math.Point3
	AxisDir  math.UnitVector3
	CrossDir math.UnitVector3
	Focal    float64
}

// NewParabola builds a parabola. axis and cross are normalized, and cross is re-orthogonalized against
// axis (its component along axis is dropped). Errors on a zero axis, a non-positive focal length, or
// axes that are parallel.
func NewParabola(vertex math.Point3, axis, cross math.Vector3, focal float64) (Parabola, error) {
	if focal <= 0 {
		return Parabola{}, fmt.Errorf("geom: parabola focal length %g must be positive", focal)
	}
	au, err := math.UnitVector3FromVector(axis)
	if err != nil {
		return Parabola{}, fmt.Errorf("geom: parabola axis %v is zero", axis)
	}
	cv := cross.Sub(au.AsVector().Scale(cross.Dot(au.AsVector())))
	cu, err := math.UnitVector3FromVector(cv)
	if err != nil {
		return Parabola{}, fmt.Errorf("geom: parabola cross axis %v is parallel to axis %v", cross, axis)
	}
	return Parabola{Vertex: vertex, AxisDir: au, CrossDir: cu, Focal: focal}, nil
}

// PointAt returns the parabola point at cross coordinate t (the parameter t is t).
func (p Parabola) PointAt(t float64) math.Point3 {
	return parabolaPoint(p.Vertex, p.AxisDir.AsVector(), p.CrossDir.AsVector(), p.Focal, t)
}

// TangentAt returns dP/dt = CrossDir + (t/(2·Focal))·AxisDir.
func (p Parabola) TangentAt(t float64) math.Vector3 {
	return parabolaTangent(p.AxisDir.AsVector(), p.CrossDir.AsVector(), p.Focal, t)
}

// Domain returns the full real line, as a Parabola is unbounded.
func (p Parabola) Domain() (lo, hi float64) { return stdmath.Inf(-1), stdmath.Inf(1) }

// Arc restricts the parabola to [t0, t1], the bounded curve stored on a trimmed face edge.
func (p Parabola) Arc(t0, t1 float64) ParabolicArc {
	return ParabolicArc{Vertex: p.Vertex, AxisDir: p.AxisDir, CrossDir: p.CrossDir, Focal: p.Focal, T0: t0, T1: t1}
}

// ParabolicArc is a bounded segment of a [Parabola], parameterized s∈[0,1] mapping to t = T0 +
// s·(T1−T0) — the edge curve a trimmed cone face stores (as LineSegment is to Line). T0 may exceed T1
// to run the arc in the opposite direction.
type ParabolicArc struct {
	Vertex   math.Point3
	AxisDir  math.UnitVector3
	CrossDir math.UnitVector3
	Focal    float64
	T0, T1   float64
}

// PointAt returns the point at parameter s∈[0,1].
func (p ParabolicArc) PointAt(s float64) math.Point3 {
	t := p.T0 + s*(p.T1-p.T0)
	return parabolaPoint(p.Vertex, p.AxisDir.AsVector(), p.CrossDir.AsVector(), p.Focal, t)
}

// TangentAt returns dP/ds (chain rule: dP/dt scaled by the t-span).
func (p ParabolicArc) TangentAt(s float64) math.Vector3 {
	t := p.T0 + s*(p.T1-p.T0)
	return parabolaTangent(p.AxisDir.AsVector(), p.CrossDir.AsVector(), p.Focal, t).Scale(math.Scalar(p.T1 - p.T0))
}

// Domain returns [0, 1].
func (p ParabolicArc) Domain() (lo, hi float64) { return 0, 1 }

// parabolaPoint evaluates Vertex + t·ĉ + (t²/(4f))·â.
func parabolaPoint(vertex math.Point3, axis, cross math.Vector3, focal, t float64) math.Point3 {
	return vertex.TranslateBy(cross.Scale(math.Scalar(t)).
		Add(axis.Scale(math.Scalar(t * t / (4 * focal)))))
}

// parabolaTangent returns dP/dt = ĉ + (t/(2f))·â.
func parabolaTangent(axis, cross math.Vector3, focal, t float64) math.Vector3 {
	return cross.Add(axis.Scale(math.Scalar(t / (2 * focal))))
}
