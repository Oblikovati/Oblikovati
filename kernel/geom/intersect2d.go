// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Segment2dIntersection is the one intersection primitive the sketch region detector
// needs: because every sketch curve (line, arc, spline, circle, ellipse) is faceted
// into straight segments before the planar arrangement is built (the same faceting the
// extrude prism uses), every curve–curve crossing reduces to a segment–segment
// crossing. This keeps the arrangement exact at the facet resolution and consistent
// with the downstream solid.
//
// It returns the crossing point and the parameters s,t∈[0,1] along a and b. ok is false
// when the segments are parallel (no single crossing) or do not actually overlap. Pass
// tol <= 0 for the default; tol widens the [0,1] acceptance so a touch at an endpoint
// (a T-junction) still counts.
func Segment2dIntersection(a, b LineSegment2d, tol float64) (pt math.Point2, s, t float64, ok bool) {
	tol = paramTol(tol)
	r := a.StartPoint.VectorTo(a.EndPoint)
	d := b.StartPoint.VectorTo(b.EndPoint)
	denom := r.Cross(d)
	if math.IsNearZero(denom, math.DefaultTolerance) {
		return math.Point2{}, 0, 0, false // parallel or degenerate
	}
	w := a.StartPoint.VectorTo(b.StartPoint)
	s = w.Cross(d) / denom
	t = w.Cross(r) / denom
	if s < -tol || s > 1+tol || t < -tol || t > 1+tol {
		return math.Point2{}, 0, 0, false
	}
	return a.PointAt(clampUnitParam(s)), clampUnitParam(s), clampUnitParam(t), true
}

// paramTol resolves a non-positive tolerance to a small default for the [0,1]
// parameter-range test (separate from the geometric DefaultTolerance).
func paramTol(tol float64) float64 {
	if tol <= 0 {
		return 1e-9
	}
	return tol
}

// clampUnitParam pins a parameter into [0,1] (a crossing reported just outside the
// range by tol resolves to the nearest endpoint).
func clampUnitParam(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
