// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// ConicArcBetween restricts a conic to the arc running from a to b on the side `near` lies.
//
// It completes [ConicSubArc], which bounds only the OPEN conics. A CLOSED conic — a circle or a full
// ellipse — carries two arcs between any pair of its points, and the parameters alone cannot say
// which one a caller means; `near` chooses. Pass a point known to lie on the wanted side, such as the
// midpoint of the chord the arc replaces.
//
// The use this exists for is a SECTION EDGE. A face's boundary must lie on that face's surface, and a
// straight segment between two points of a cylinder or a cone does not — it is a chord through the
// solid, so the face it bounds is not valid geometry however small the sag. Cutting the edge from the
// neighbouring PLANAR face instead gives a curve lying exactly on both faces at once, and this bounds
// it to the piece the edge needs (Oblikovati/Oblikovati#3503).
//
// ok=false when the curve is not a conic this can bound, when an endpoint does not invert onto it, or
// when the two endpoints land on the same parameter (no arc is determined).
//
// Example:
//
//	sec, _ := geom.IntersectSurfacesAnalytic(wall, cap, res)
//	arc, ok := geom.ConicArcBetween(sec[0], a, b, a.Midpoint(b))
func ConicArcBetween(c Curve3, a, b, near math.Point3) (Curve3, bool) {
	ta, okA := ConicParamAt(c, a)
	tb, okB := ConicParamAt(c, b)
	if !okA || !okB || ta == tb {
		return nil, false
	}
	switch x := c.(type) {
	case Circle:
		return nearerArc(circleArcWay(x, ta, tb, true), circleArcWay(x, ta, tb, false), near), true
	case EllipseFull:
		return nearerArc(ellipseArcWay(x, ta, tb, true), ellipseArcWay(x, ta, tb, false), near), true
	}
	return ConicSubArc(c, ta, tb)
}

// circleArcWay is the arc from ta to tb taken the positive way about the normal when forward, and the
// other way round when not.
func circleArcWay(c Circle, ta, tb float64, forward bool) Curve3 {
	return Arc3d{
		Center: c.Center, Normal: c.Normal, RefDir: c.RefDir, Radius: c.Radius,
		StartAngle: twoPi * ta, SweepAngle: arcSweepWay(ta, tb, forward),
	}
}

// ellipseArcWay mirrors circleArcWay for a full ellipse. Both EllipseFull and EllipticalArc are
// written in the ECCENTRIC angle, so the sweep carries over with no reparameterisation.
func ellipseArcWay(e EllipseFull, ta, tb float64, forward bool) Curve3 {
	return EllipticalArc{
		Center: e.Center, Normal: e.Normal, MajorAxis: e.MajorAxis,
		MajorRadius: e.MajorRadius, MinorRadius: e.MinorRadius,
		StartAngle: twoPi * ta, SweepAngle: arcSweepWay(ta, tb, forward),
	}
}

// arcSweepWay is the signed sweep in RADIANS from ta to tb: positive about the normal when forward,
// its negative complement when not — the two arcs a closed conic offers between one pair of points.
func arcSweepWay(ta, tb float64, forward bool) float64 {
	fwd := wrapUnit(tb - ta)
	if forward {
		return twoPi * fwd
	}
	return twoPi * (fwd - 1)
}

// nearerArc returns whichever candidate passes closer to `near` at its own midpoint. The two arcs of
// a closed conic meet only at their shared ends, so their midpoints are as far apart as the curve
// allows and the comparison is never a near-tie for a point off to one side.
func nearerArc(a, b Curve3, near math.Point3) Curve3 {
	if near.DistanceTo(a.PointAt(0.5)) <= near.DistanceTo(b.PointAt(0.5)) {
		return a
	}
	return b
}
