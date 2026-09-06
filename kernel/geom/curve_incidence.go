// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

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
