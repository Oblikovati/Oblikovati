// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The ELLIPSE arm of the curved-survivor rim carry. fillet_survivor_rim.go re-derives a survivor
// wall's retained rim from the parent curve's OWN parameters, but every helper it uses
// (projectOntoArcCircle / arcFrac / subArcOnParent) is written for a CIRCLE, so a rim that is a
// geom.EllipticalArc — the cap rim of an elliptic prism / elliptical cylinder (F7's
// `ellipse 15 10` + `prism 2 0 10`) — fell through to nil and shipped a straight CHORD across its own
// face: 89.44 off the wall, the corpus's largest per-face gross error (40793 vs DRAWEXE).
//
// The parameter algebra is NOT the arc-length algebra: an ellipse's parameter is the ECCENTRIC ANGLE,
// and arc length along it is an incomplete elliptic integral of the second kind. That is exactly why
// the circle helpers cannot be reused as-is. The three operations the carry actually needs, however,
// are all eccentric-angle operations and stay closed-form, because the ellipse is the image of the unit
// circle under the linear map D = diag(MajorRadius, MinorRadius) in the (MajorAxis, minorAxis) frame:
//
//	inversion   θ(p) = atan2((p−C)·m̂ / b, (p−C)·M̂ / a)     — ellipticArcFrac, already in the kernel
//	sub-span    same centre/axes/radii, StartAngle=θ(from), SweepAngle=θ(to)−θ(from)  — ellipticSubArc
//	projection  evaluate the ellipse at θ(p)                — projectOntoEllipse, below
//
// D scales AREA uniformly by ab, so the circular-segment area ½R²(Δθ−sinΔθ) becomes ½ab(Δθ−sinΔθ)
// under the same map — but this file does not need it: an elliptic rim is carried only when both
// retained-span endpoints already lie ON the parent (ellipseSpanIsExact), which is the case in which
// the sub-span IS the wall's true boundary and no chord-vs-arc area argument can justify degrading it.

// projectOntoEllipse drops p onto the ellipse of ea (same centre/plane/axes/radii) along the ellipse's
// own parameter: it inverts p to an eccentric angle θ and evaluates the ellipse there. This is the
// EllipticalArc sibling of projectOntoArcCircle — which likewise projects RADIALLY (not to the closest
// point), so a point already on the parent is returned to machine precision and a point pulled slightly
// off it lands at the parameter the sub-span algebra will use. It is computed WITHOUT trigonometry:
// (cosθ, sinθ) is the unit-circle preimage (u,v)/|(u,v)| of p under D⁻¹, so evaluating the ellipse there
// is one normalize plus the two axis scalings. A p on the ellipse's axis (both projected components
// zero) has no defined angle and is returned unchanged, a degeneracy a rim tangent point never reaches.
func projectOntoEllipse(ea geom.EllipticalArc, p math.Point3) math.Point3 {
	minor := ea.Normal.Cross(ea.MajorAxis)
	d := ea.Center.VectorTo(p)
	u := float64(d.Dot(ea.MajorAxis.AsVector())) / ea.MajorRadius
	v := float64(d.Dot(minor)) / ea.MinorRadius
	norm := stdmath.Hypot(u, v)
	if norm == 0 {
		return p
	}
	along := ea.MajorAxis.AsVector().Scale(math.Scalar(ea.MajorRadius * u / norm))
	across := minor.Scale(math.Scalar(ea.MinorRadius * v / norm))
	return ea.Center.TranslateBy(along.Add(across))
}

// ellipseSpanIsExact reports whether both retained-span endpoints already lie ON the parent ellipse,
// within 1e-9 of its major radius (the rim's own scale, so it is scale-invariant without threading a
// tol.Resolution — the same test rimSpanIsExact makes for a circular rim). True when the loop point IS a
// point of the parent rim: a fillet tangent point that the corner solve placed on the wall's own rim
// curve. False for a corner whose section cap is still off the rim, where the parent's sub-span is not
// the wall's boundary and the base chord is kept.
func ellipseSpanIsExact(ea geom.EllipticalArc, a, b math.Point3) bool {
	tol := 1e-9 * ea.MajorRadius
	return a.DistanceTo(projectOntoEllipse(ea, a)) <= tol &&
		b.DistanceTo(projectOntoEllipse(ea, b)) <= tol
}

// retainedEllipticRimCurve is the sub-arc to carry for an ELLIPTIC survivor rim between the loop
// segment's own endpoints, or nil (the base straight chord, the pre-fix behaviour) when the span is not
// exactly on the parent or the ellipse's own parameter inversion declines an endpoint. Built from the
// parent's OWN parameters (ellipticSubArc), so a MAJOR retained span stays major — a three-point re-fit
// has no ellipse analogue at all, which is the other half of why the circle helpers could not be reused.
func retainedEllipticRimCurve(ea geom.EllipticalArc, p0, p1 math.Point3) geom.Curve3 {
	if !ellipseSpanIsExact(ea, p0, p1) {
		return nil
	}
	sub, ok := ellipticSubArc(ea, projectOntoEllipse(ea, p0), projectOntoEllipse(ea, p1))
	if !ok {
		return nil
	}
	return sub
}

// ellipseSpansItsSegment reports whether ea already runs from p0 to p1, within 1e-9 of its major radius
// (the EllipticalArc sibling of arcSpansItsSegment).
func ellipseSpansItsSegment(ea geom.EllipticalArc, p0, p1 math.Point3) bool {
	lo, hi := ea.Domain()
	tol := 1e-9 * ea.MajorRadius
	return ea.PointAt(lo).DistanceTo(p0) <= tol && ea.PointAt(hi).DistanceTo(p1) <= tol
}
