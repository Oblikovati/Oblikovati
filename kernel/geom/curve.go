// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Curve3 is a parametrically evaluable 3D curve. Implementations map a parameter
// t in their [Curve3.Domain] to a point and a tangent (the derivative dP/dt,
// not normalized — its magnitude carries the parameterization speed, which
// consumers may need; normalize when only direction matters).
type Curve3 interface {
	// PointAt returns the position at parameter t.
	PointAt(t float64) math.Point3
	// TangentAt returns the derivative dP/dt at parameter t.
	TangentAt(t float64) math.Vector3
	// Domain returns the valid parameter range [lo, hi]. Unbounded curves
	// (an infinite line) return ±Inf.
	Domain() (lo, hi float64)
}

// Curve2 is the 2D (sketch-space) analogue of [Curve3].
type Curve2 interface {
	PointAt(t float64) math.Point2
	TangentAt(t float64) math.Vector2
	Domain() (lo, hi float64)
}

// pointOnCircle returns center + r·(cos(a)·u + sin(a)·v), the point at angle a
// on a circle with the orthonormal in-plane basis (u, v).
func pointOnCircle(center math.Point3, u, v math.Vector3, r, a float64) math.Point3 {
	cos, sin := cosSin(a)
	return center.TranslateBy(u.Scale(r * cos).Add(v.Scale(r * sin)))
}

// circleTangent returns the unnormalized derivative of [pointOnCircle] with
// respect to a: r·(−sin(a)·u + cos(a)·v).
func circleTangent(u, v math.Vector3, r, a float64) math.Vector3 {
	cos, sin := cosSin(a)
	return u.Scale(-r * sin).Add(v.Scale(r * cos))
}
