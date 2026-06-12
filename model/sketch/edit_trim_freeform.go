// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Trim/extend against free-form targets — ellipses, elliptical arcs and
// splines — closing the curve-pair classes left open by the analytic cases in
// edit_trim_curves.go (M06-F12, Oblikovati/Oblikovati#627; deferred-work
// registry #599). The targets are adapted to kernel [geom.Curve2] values and
// crossings come from the sign-bracketing intersector in
// kernel/geom/intersect2d_curve.go, refined onto the true curve.

// freeformCurve2 adapts an ellipse, elliptical arc or spline to a kernel
// parametric curve; ok is false for every other entity kind or for degenerate
// geometry that cannot be adapted.
func freeformCurve2(e Entity) (geom.Curve2, bool) {
	switch g := e.(type) {
	case *Ellipse:
		full, err := geom.NewEllipseFull2d(g.Center.Position(), g.MajorAxis,
			float64(g.MajorRadius), float64(g.MinorRadius))
		return full, err == nil
	case *EllipticalArc:
		arc, err := geom.NewEllipticalArc2d(g.Center.Position(), g.MajorAxis,
			float64(g.MajorRadius), float64(g.MinorRadius),
			float64(g.StartAngle), float64(g.EndAngle-g.StartAngle))
		return arc, err == nil
	case *Spline:
		return splineCurve2d(g)
	}
	return nil, false
}

// splineCurve2d builds the spline's true NURBS curve — the same construction
// the sampler uses, so trims land exactly on the rendered curve.
func splineCurve2d(sp *Spline) (geom.Curve2, bool) {
	pts := splinePositions(sp)
	if len(pts) < 3 {
		return nil, false
	}
	if sp.IsFitType() {
		curve, _, err := fitCurveFor(pts, sp.Closed, fitParameterization(sp.FitMethod))
		return curve, err == nil
	}
	curve, err := controlCurveFor(pts, sp.Closed)
	return curve, err == nil
}

// segmentFreeformHits returns the points where segment seg crosses a
// free-form entity; nil for other kinds.
func segmentFreeformHits(seg geom.LineSegment2d, e Entity) []math.Point2 {
	c, ok := freeformCurve2(e)
	if !ok {
		return nil
	}
	return geom.SegmentCurve2dIntersection(seg, c)
}

// supportFreeformHits returns the points where the infinite line crosses a
// free-form entity; nil for other kinds.
func supportFreeformHits(line geom.Line2d, e Entity) []math.Point2 {
	c, ok := freeformCurve2(e)
	if !ok {
		return nil
	}
	return geom.LineCurve2dIntersection(line, c)
}

// circleFreeformHits returns the points where circle cc crosses a free-form
// entity; nil for other kinds.
func circleFreeformHits(cc geom.Circle2d, e Entity) []math.Point2 {
	c, ok := freeformCurve2(e)
	if !ok {
		return nil
	}
	return geom.CircleCurve2dIntersection(cc, c)
}
