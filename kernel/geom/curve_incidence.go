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
	if on, ok := sectionIncidence(cv); ok {
		return on
	}
	if x, ok := cv.(TrimmedCurve3); ok {
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

// sectionIncidence answers for the intersector's own section forms — the curves it returns from a
// closed form, which are on two surfaces by construction and name both. Each is listed by value AND by
// pointer: an imprint may travel by identity so the arrangement's run-merge can compare curves with ==,
// and a form that answers for one and not the other reports no incidence for half its callers.
func sectionIncidence(cv Curve3) ([]func(math.Point3) float64, bool) {
	switch x := cv.(type) {
	case RuledQuadricArc:
		return ruledQuadricIncidence(x.Base, x.Quad), true
	case *RuledQuadricArc:
		return ruledQuadricIncidence(x.Base, x.Quad), true
	case RuledQuadricLoop:
		return ruledQuadricIncidence(x.Base, x.Quad), true
	case *RuledQuadricLoop:
		return ruledQuadricIncidence(x.Base, x.Quad), true
	case TorusQuadricArc:
		return torusQuadricIncidence(x.Torus, x.Quad), true
	case *TorusQuadricArc:
		return torusQuadricIncidence(x.Torus, x.Quad), true
	case TorusQuadricLoop:
		return torusQuadricIncidence(x.Torus, x.Quad), true
	case *TorusQuadricLoop:
		return torusQuadricIncidence(x.Torus, x.Quad), true
	}
	return nil, false
}

// ruledQuadricIncidence is a ruled∩quadric section's two conditions: the quadric it carries, and the
// base surface it runs on when that surface has an implicit form of its own. It serves the wrapping arc
// and the folded loop alike — they are one closed form over different windows, so they are ON the same
// two surfaces.
func ruledQuadricIncidence(base Surface, quad Quadric) []func(math.Point3) float64 {
	out := []func(math.Point3) float64{quad.ValueAt}
	if q, ok := base.(ImplicitQuadric); ok {
		form := q.QuadricForm()
		out = append(out, form.ValueAt)
	}
	return out
}

// torusQuadricIncidence is a torus∩quadric section's two conditions: the quadric it carries, and the
// TORUS it runs on. A torus is quartic and has no quadric form, so its condition is its own signed
// distance — an exact implicit function like any other, and the one thing a caller solving against this
// curve needs.
//
// A section curve WITHOUT its conditions is not merely slower to intersect: it cannot be intersected at
// all. curvePairMeets needs roots on BOTH curves and pairs them by distance, so a curve that reports no
// incidence yields no crossing — silently. That is how an axial drill through a ring lost the crossings
// between its bore seams and the wall chart's own seam, and came back as half a tube bridged by two
// rulings (ADR-0061 stage 5).
func torusQuadricIncidence(t Torus, quad Quadric) []func(math.Point3) float64 {
	return []func(math.Point3) float64{
		quad.ValueAt,
		func(p math.Point3) float64 { return float64(SignedDistanceToSurface(t, p)) },
	}
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
