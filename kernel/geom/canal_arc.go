// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/math"
)

// The u-cross-section of a constant-radius rolling-ball corner blend at a spine (ball-center)
// point m is the EXACT radius-r circular arc between the two host-contact feet fa, fb — centered
// at m, in the plane of {fa, fb, m}, on the cavity (minor-arc) side. Because both feet sit on the
// ball's characteristic circle (each at distance r from m) the plane {fa, fb, m} IS the arc plane,
// so no spine differentiation is needed (canal-corner-math.md §2; do Carmo §3-4). It is a rational
// quadratic with shoulder weight w = cos(½·∠(fa, m, fb)) (Piegl & Tiller §7) — exactly OCCT's
// u-rational-quadratic (result5-poles.txt row-2 weights 0.707→0.667 = cos β).

const (
	// halfPi is the arc half-angle ceiling: at β = π/2 the feet are antipodal and the rational
	// quadratic degenerates (weight cos β → 0, parabola/hyperbola blow-up, pitfall 3).
	halfPi = stdmath.Pi / 2
	// arcMinHalfAngle floors/ceils the arc half-angle: outside (arcMinHalfAngle, halfPi−arcMinHalfAngle)
	// the feet+center are collinear (grazing β→0 or antipodal β→π/2) and there is no radius arc.
	arcMinHalfAngle = 1e-6 // tol:angular — collinear-feet grazing/antipodal guard
)

// crossSectionArc builds the EXACT radius-`radius` rational-quadratic arc fa → shoulder → fb of a
// rolling-ball cross-section: centered at ball-center m, in the plane of {fa, fb, m}, on the cavity
// side, weight cos(½·∠(fa,m,fb)). It errors (carrying the half-angle) when fa, fb, m are collinear
// (grazing/antipodal) — there is no radius arc plane. The returned Curve3 is the same geometry as
// the loft's v=0/v=1 end isolines, so a blend face's boundary edge welds to it (blend_ruled_arc.go).
//
//	arc, err := crossSectionArc(cPrime, wallFoot, cylFoot, 5)
func crossSectionArc(m, fa, fb math.Point3, radius float64) (Curve3, error) {
	shoulder, weight, err := arcControls(m, fa, fb, radius)
	if err != nil {
		return nil, err
	}
	arc, err := NewConicSectionCurve(fa, shoulder, fb, weight)
	if err != nil {
		return nil, err
	}
	return arc, nil
}

// arcControls returns the rational-quadratic middle control point (shoulder) and its weight for
// the radius-`radius` arc fa → fb about center m. The shoulder is the intersection of the arc
// tangents at fa and fb, which for a circular arc lies on the feet bisector at distance
// radius/weight from m; since the two radii m→fa, m→fb subtend 2β the bisector of the two unit
// radii has length 2·cos β = 2·weight, giving shoulder = m + bisector·radius/(2·weight²). It is the
// one construction shared by [crossSectionArc] and the loft's per-station columns.
func arcControls(m, fa, fb math.Point3, radius float64) (shoulder math.Point3, weight float64, err error) {
	da, db := m.VectorTo(fa), m.VectorTo(fb)
	la, lb := float64(da.Length()), float64(db.Length())
	if la == 0 || lb == 0 {
		return math.Point3{}, 0, fmt.Errorf(
			"crossSectionArc: foot coincides with center m=%v (|m→fa|=%g, |m→fb|=%g), expected both = radius %g",
			m, la, lb, radius)
	}
	half := 0.5 * stdmath.Acos(clampUnit(float64(da.Dot(db))/(la*lb)))
	if half <= arcMinHalfAngle || half >= halfPi-arcMinHalfAngle {
		return math.Point3{}, 0, fmt.Errorf(
			"crossSectionArc: feet+center near-collinear (arc half-angle %g rad outside (%g, %g)): "+
				"grazing/antipodal, no radius-%g arc plane", half, arcMinHalfAngle, halfPi-arcMinHalfAngle, radius)
	}
	weight = stdmath.Cos(half)
	bis := da.Scale(1 / la).Add(db.Scale(1 / lb)) // feet bisector → the minor-arc (cavity) side, |bis| = 2·weight
	return m.TranslateBy(bis.Scale(radius / (2 * weight * weight))), weight, nil
}

// clampUnit clamps x to [-1, 1] so a round-off-inflated dot/‖·‖ ratio cannot make Acos return NaN.
func clampUnit(x float64) float64 {
	return stdmath.Max(-1, stdmath.Min(1, x))
}
