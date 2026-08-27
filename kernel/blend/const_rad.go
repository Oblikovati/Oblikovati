// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// SectionArc is a constant-radius fillet's cross-section at one guide station: the circular arc of
// radius R centred at Center, sweeping Sweep radians from FootA (the contact on support 1, at v=0)
// through the exposed side to FootB (the contact on support 2, at v=1). It lies in the plane normal
// to the centre-curve tangent — the exact section plane, since the centre lies on both offset
// surfaces so the tangent is perpendicular to both support normals (ADR-0050 P4, method B).
type SectionArc struct {
	Center       math.Point3
	E1, E2       math.Vector3 // orthonormal in-plane basis; E1 points from Center to FootA
	R            float64
	Sweep        float64 // exposed-arc angle from the side-1 to the side-2 contact
	FootA, FootB math.Point3
}

// PointAt returns the section point at v∈[0,1]: Center + R(cos(vθ)E1 + sin(vθ)E2), so PointAt(0)=FootA
// and PointAt(1)=FootB.
func (s SectionArc) PointAt(v float64) math.Point3 {
	a := v * s.Sweep
	cos, sin := stdmath.Cos(a), stdmath.Sin(a)
	off := s.E1.Scale(math.Scalar(s.R * cos)).Add(s.E2.Scale(math.Scalar(s.R * sin)))
	return s.Center.TranslateBy(off)
}

// sectionAt builds the fillet section of radius r at a centre point c that lies on the blend's
// centre curve (the intersection of the two offset supports): it projects c onto each support for
// the two contact (foot) points, frames the arc in the plane normal to the centre-curve tangent,
// and selects the EXPOSED arc — the one whose midpoint lies outside the solid (inside==false). It
// returns ok=false when c is not equidistant r from both supports (c is off the centre curve).
func sectionAt(c math.Point3, a, b geom.Surface, r, tol float64, inside func(math.Point3) bool) (SectionArc, bool) {
	fa, fb := footOn(a, c), footOn(b, c)
	if stdmath.Abs(float64(fa.DistanceTo(c))-r) > tol || stdmath.Abs(float64(fb.DistanceTo(c))-r) > tol {
		return SectionArc{}, false
	}
	e1 := unitVec(c.VectorTo(fa))
	toB := c.VectorTo(fb)
	e2 := unitVec(toB.Add(e1.Scale(-toB.Dot(e1)))) // Gram-Schmidt: (c→fb) component ⟂ e1
	sweep := stdmath.Acos(math.Clamp(float64(e1.Dot(unitVec(toB))), -1, 1))
	arc := SectionArc{Center: c, E1: e1, E2: e2, R: r, Sweep: sweep, FootA: fa, FootB: fb}
	if inside(arc.PointAt(0.5)) { // the minor arc dips into the material: take the major arc instead
		arc.E2 = e2.Negate()
		arc.Sweep = 2*stdmath.Pi - sweep
	}
	return arc, true
}

// footOn returns the projection of c onto surface s (its nearest point for the plane/cylinder/sphere
// supports, via the surface's on-frame parameter inverse).
func footOn(s geom.Surface, c math.Point3) math.Point3 {
	u, v := s.ParamAt(c)
	return s.PointAt(u, v)
}

// unitVec normalizes v, returning it unchanged when it is zero length.
func unitVec(v math.Vector3) math.Vector3 {
	if l := float64(v.Length()); l > 0 {
		return v.Scale(math.Scalar(1 / l))
	}
	return v
}
