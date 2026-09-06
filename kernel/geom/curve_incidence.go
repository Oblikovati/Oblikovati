// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// CurveIncidence returns the conditions a point ALREADY ON the curve's host surface must satisfy to lie
// on the CURVE itself: each is zero on it and signed on either side. The list is empty for a curve with
// no such condition in closed form.
//
// It is what makes two curves that share a surface intersectable without walking both: substitute one
// curve's own parameterisation into the other's incidence and their meeting is a scalar root, certified
// against the geometry rather than found by sliding two polylines past each other.
//
//	for _, f := range geom.CurveIncidence(section) { ... } // f(p) == 0 on `section`, for p on the host
//
// A section curve's condition is its PLANE: on the host surface, lying in that plane is the same thing
// as lying on the section. A ruled-quadric crossing lies on TWO surfaces and names both, because which
// of them is the host — and so carries no information — depends on the chart asking.
func CurveIncidence(cv Curve3) []func(math.Point3) float64 {
	switch x := cv.(type) {
	case RuledQuadricArc:
		return ruledQuadricIncidence(x)
	case *RuledQuadricArc:
		return ruledQuadricIncidence(*x)
	case TrimmedCurve3:
		return CurveIncidence(x.Base)
	}
	if IsStraightCurve(cv) {
		return straightIncidence(cv)
	}
	cf, ok := AsConic(cv)
	if !ok {
		return nil
	}
	center, normal := cf.Center, cf.Major.Cross(cf.Minor)
	return []func(math.Point3) float64{
		func(p math.Point3) float64 { return float64(center.VectorTo(p).Dot(normal)) },
	}
}

// ruledQuadricIncidence is the crossing's two conditions: the quadric it carries, and the base surface
// it runs on when that surface has an implicit form of its own.
func ruledQuadricIncidence(a RuledQuadricArc) []func(math.Point3) float64 {
	out := []func(math.Point3) float64{a.Quad.ValueAt}
	if q, ok := a.Base.(ImplicitQuadric); ok {
		form := q.QuadricForm()
		out = append(out, form.ValueAt)
	}
	return out
}

// straightIncidence is a line's two conditions: the distances to two perpendicular planes through it.
// A straight curve is no section, so it had no incidence at all, and a chart's artificial SEAM — a
// ruling — could not be brought to the incidence solver: its crossings with an imprint were taken from
// the section-plane route, which answers only for planar sections and so never for a ruled∩quadric
// arc, and reported one point where a window loop crosses the seam twice (ADR-0061 stage 4).
func straightIncidence(cv Curve3) []func(math.Point3) float64 {
	lo, _ := cv.Domain()
	if stdmath.IsInf(lo, 0) {
		lo = 0
	}
	origin := cv.PointAt(lo)
	dir, err := math.UnitVector3FromVector(cv.TangentAt(lo))
	if err != nil {
		return nil
	}
	n1 := math.AnyPerpendicular(dir).AsVector()
	n2 := dir.AsVector().Cross(n1)
	return []func(math.Point3) float64{
		func(p math.Point3) float64 { return float64(origin.VectorTo(p).Dot(n1)) },
		func(p math.Point3) float64 { return float64(origin.VectorTo(p).Dot(n2)) },
	}
}
