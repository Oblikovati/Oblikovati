// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

const twoPi = 2 * stdmath.Pi

// cosSin returns cos(a) and sin(a) together, the common pair for evaluating
// circular and elliptical geometry.
func cosSin(a float64) (cos, sin float64) {
	return stdmath.Cos(a), stdmath.Sin(a)
}

// clamp01 constrains t to [0, 1], used to project a line parameter onto a
// bounded segment.
func clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// wrapPositive maps an angle delta into the half-open range (0, 2π], used to
// turn a raw end−start difference into a positive (counter-clockwise) sweep.
func wrapPositive(delta float64) float64 {
	d := stdmath.Mod(delta, twoPi)
	if d <= 0 {
		d += twoPi
	}
	return d
}

// wrap2pi maps any angle into the half-open range [0, 2π) — the canonical form
// for an inverted angular parameter (see ParamAt).
func wrap2pi(a float64) float64 {
	d := stdmath.Mod(a, twoPi)
	if d < 0 {
		d += twoPi
	}
	return d
}

// clampUnit constrains x to [−1, 1] so it is a valid argument to asin/acos after
// the rounding of a normalized-direction component.
func clampUnit(x float64) float64 {
	if x > 1 {
		return 1
	}
	if x < -1 {
		return -1
	}
	return x
}

// clampTo constrains x to [lo, hi].
func clampTo(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// signedArea2 returns twice the signed area of triangle (a, b, c). Positive
// means the three points wind counter-clockwise.
func signedArea2(a, b, c math.Point2) float64 {
	return a.VectorTo(b).Cross(a.VectorTo(c))
}

// perpendicularUnit returns some unit vector perpendicular to u. It crosses u
// with whichever world axis is least aligned with it, so the result is never
// degenerate. Used to pick a default angle-zero reference for a circle whose
// caller supplied only a plane normal.
func perpendicularUnit(u math.UnitVector3) math.UnitVector3 {
	seed := math.V3(1, 0, 0)
	if stdmath.Abs(u.X()) > 0.9 {
		seed = math.V3(0, 1, 0)
	}
	return u.Cross(seed.AsUnit()).AsUnit()
}

// axisFrame returns two in-plane unit vectors (ref, binormal) perpendicular to
// axis and to each other, giving a right-handed frame (ref, binormal, axis) for
// parameterizing surfaces of revolution.
func axisFrame(axis math.UnitVector3) (ref, binormal math.Vector3) {
	r := perpendicularUnit(axis)
	return r.AsVector(), axis.Cross(r)
}

// unitOrZero returns v normalized, or the zero vector when v is too short to
// have a direction (e.g. a surface normal at a pole or apex). Surfaces return
// this rather than erroring, because evaluation at a degenerate parameter is a
// valid query with a degenerate answer.
func unitOrZero(v math.Vector3) math.Vector3 {
	u, err := math.UnitVector3FromVector(v)
	if err != nil {
		return math.Vector3{}
	}
	return u.AsVector()
}

// circumcenter2d returns the center of the circle through three points and
// true, or the zero point and false when the points are collinear (no finite
// circle exists).
func circumcenter2d(a, b, c math.Point2) (math.Point2, bool) {
	d := 2 * signedArea2(a, b, c)
	if d <= math.DefaultTolerance && d >= -math.DefaultTolerance {
		return math.Point2{}, false
	}
	a2, b2, c2 := a.AsVector().LengthSquared(), b.AsVector().LengthSquared(), c.AsVector().LengthSquared()
	ux := (a2*(b.Y-c.Y) + b2*(c.Y-a.Y) + c2*(a.Y-b.Y)) / d
	uy := (a2*(c.X-b.X) + b2*(a.X-c.X) + c2*(b.X-a.X)) / d
	return math.P2(ux, uy), true
}
