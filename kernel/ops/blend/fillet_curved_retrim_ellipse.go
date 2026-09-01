// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Elliptic-arc survivor-edge splitting for the host retrim (the F4 elliptic-prism vein). The retrim
// segment machinery (fillet_curved_retrim_loop.go) handles STRAIGHT and CIRCULAR (Arc3d) survivor
// edges; an EllipticalCylinder host's cap edges are geom.EllipticalArc (the wall's cross-section at a
// cap plane), which the circular path treated as a chord — so the rolling-ball contact foot, which
// lies on the ellipse, was never found on the loop and the retrim declined. These helpers give the
// SAME operations for an elliptic arc — point→parameter inversion and sub-arc trimming — mirroring
// arcParam / subArcMajor so the elliptic host retrims (and its shared cap welds) exactly as a
// circular host does. Circular and straight edges are byte-identical (they never enter here).

// ellipticArcParam is a point's fractional parameter t∈(0,1) STRICTLY interior to elliptic arc ea, on
// the arc within tol (the sibling of arcParam for geom.EllipticalArc). Endpoints (t≤0 or t≥1) decline
// so a split point coincident with a vertex is not re-inserted.
func ellipticArcParam(ea geom.EllipticalArc, p math.Point3, tol float64) (float64, bool) {
	t, ok := ellipticArcFrac(ea, p, tol)
	if !ok || t <= 0 || t >= 1 {
		return 0, false
	}
	return t, true
}

// ellipticArcFrac is a point's fractional parameter t∈[0,1] on elliptic arc ea (endpoints included),
// ok only when p lies on the arc within tol and inside its sweep. The angle inversion rescales the
// projected components by the radii before atan2 (the SAME inversion geom.EllipticalCylinder.ParamAt /
// EllipticalArc use), then wrapToSweep folds the angle into the arc's own sweep so a point OUTSIDE the
// sweep (t>1) is rejected by the on-curve check rather than aliased onto the arc.
func ellipticArcFrac(ea geom.EllipticalArc, p math.Point3, tol float64) (float64, bool) {
	if ea.SweepAngle == 0 {
		return 0, false
	}
	d := ea.Center.VectorTo(p)
	minor := ea.Normal.Cross(ea.MajorAxis)
	a := stdmath.Atan2(float64(d.Dot(minor))/ea.MinorRadius, float64(d.Dot(ea.MajorAxis.AsVector()))/ea.MajorRadius)
	t := wrapToSweep(a-ea.StartAngle, ea.SweepAngle) / ea.SweepAngle
	if t < -ellipticFracEps || t > 1+ellipticFracEps || float64(ea.PointAt(t).DistanceTo(p)) > tol {
		return 0, false
	}
	return t, true
}

// ellipticFracEps is the parameter-space slack (dimensionless) that admits an arc's own endpoints
// (t≈0 / t≈1) through ellipticArcFrac despite floating-point angle noise; on-curve position is still
// gated by the model-relative point tolerance, so this never admits an off-arc point.
const ellipticFracEps = 1e-9

// ellipticSubArc is the sub-arc of ea between on-arc points from and to (the sibling of subSeg's Arc3d
// branch for geom.EllipticalArc): same centre/axes/radii, StartAngle at from's parameter, SweepAngle
// spanning from→to along ea's own sweep — so the trimmed cap edge stays an exact ellipse (never a
// chord) and both from and to fall on it by construction. Declines when either endpoint is off ea.
func ellipticSubArc(ea geom.EllipticalArc, from, to math.Point3) (geom.EllipticalArc, bool) {
	tf, okf := ellipticArcFrac(ea, from, ellipticSubArcTol(ea))
	tt, okt := ellipticArcFrac(ea, to, ellipticSubArcTol(ea))
	if !okf || !okt {
		return geom.EllipticalArc{}, false
	}
	return geom.EllipticalArc{
		Center: ea.Center, Normal: ea.Normal, MajorAxis: ea.MajorAxis,
		MajorRadius: ea.MajorRadius, MinorRadius: ea.MinorRadius,
		StartAngle: ea.StartAngle + tf*ea.SweepAngle, SweepAngle: (tt - tf) * ea.SweepAngle,
	}, true
}

// ellipticSubArcTol is the on-arc acceptance for the sub-arc endpoints, scaled to the ellipse size (a
// fraction of the major radius) so it is model-relative (ADR-0042) rather than a bare epsilon.
func ellipticSubArcTol(ea geom.EllipticalArc) float64 { return 1e-6 * (ea.MajorRadius + 1) }
