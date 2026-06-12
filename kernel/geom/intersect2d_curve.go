// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Intersections of an arbitrary parametric 2D curve ([Curve2]: a B-spline, an
// ellipse, …) with the analytic primitives, by sign-change bracketing over the
// curve's parameter and bisection refinement (M06-F12,
// Oblikovati/Oblikovati#627). The analytic↔analytic pairs keep their exact
// closed forms in intersect2d*.go; this is the fallback that makes every
// remaining pair honest instead of unsupported.

// curveIntersectSamples is the default bracketing density: fine enough that a
// sketch-scale curve cannot cross a primitive twice between consecutive
// samples (a spline span is itself only sampled 8× for region detection).
const curveIntersectSamples = 256

// curveZeroIterations bounds the bisection refinement; 60 halvings reduce any
// bracket below 1e-18 of the domain — far past float64 resolution.
const curveZeroIterations = 60

// SegmentCurve2dIntersection returns the points where segment seg crosses
// curve c, refined onto the true curve.
func SegmentCurve2dIntersection(seg LineSegment2d, c Curve2) []math.Point2 {
	dir := seg.StartPoint.VectorTo(seg.EndPoint)
	side := func(p math.Point2) float64 {
		return float64(dir.Cross(seg.StartPoint.VectorTo(p)))
	}
	var out []math.Point2
	for _, p := range curveSignedCrossings(c, side) {
		if t := projectOnSegment(seg, p); t >= 0 && t <= 1 {
			out = append(out, p)
		}
	}
	return out
}

// LineCurve2dIntersection returns the points where the infinite line crosses
// curve c, refined onto the true curve.
func LineCurve2dIntersection(l Line2d, c Curve2) []math.Point2 {
	side := func(p math.Point2) float64 {
		return float64(l.Dir.AsVector().Cross(l.Origin.VectorTo(p)))
	}
	return curveSignedCrossings(c, side)
}

// CircleCurve2dIntersection returns the points where circle cc crosses curve
// c, refined onto the true curve.
func CircleCurve2dIntersection(cc Circle2d, c Curve2) []math.Point2 {
	radial := func(p math.Point2) float64 {
		return float64(p.DistanceTo(cc.Center)) - cc.Radius
	}
	return curveSignedCrossings(c, radial)
}

// curveSignedCrossings locates every parameter where the signed field f
// changes sign along c and returns the refined crossing points. Tangential
// grazes (sign touch without crossing) are below the bracketing resolution by
// construction and are not reported.
func curveSignedCrossings(c Curve2, f func(math.Point2) float64) []math.Point2 {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || hi <= lo {
		return nil
	}
	var out []math.Point2
	prevT, prevF := lo, f(c.PointAt(lo))
	if prevF == 0 {
		out = append(out, c.PointAt(lo))
	}
	for i := 1; i <= curveIntersectSamples; i++ {
		t := lo + (hi-lo)*float64(i)/curveIntersectSamples
		ft := f(c.PointAt(t))
		switch {
		case ft == 0: // the sample landed exactly on the zero
			out = append(out, c.PointAt(t))
		case prevF == 0: // that zero was already reported; not a new crossing
		case (prevF > 0) != (ft > 0):
			out = append(out, c.PointAt(bisectCurveZero(c, f, prevT, t, prevF)))
		}
		prevT, prevF = t, ft
	}
	return out
}

// bisectCurveZero shrinks the sign-change bracket [a, b] of f∘c down to float
// resolution and returns the zero's parameter.
func bisectCurveZero(c Curve2, f func(math.Point2) float64, a, b, fa float64) float64 {
	for range [curveZeroIterations]int{} {
		mid := (a + b) / 2
		fm := f(c.PointAt(mid))
		if fm == 0 {
			return mid
		}
		if (fa > 0) == (fm > 0) {
			a, fa = mid, fm
		} else {
			b = mid
		}
	}
	return (a + b) / 2
}

// projectOnSegment returns p's parameter along seg (0 at the start, 1 at the
// end) by orthogonal projection; the crossing point already lies on the
// segment's support line, so this only bounds it to the segment.
func projectOnSegment(seg LineSegment2d, p math.Point2) float64 {
	d := seg.StartPoint.VectorTo(seg.EndPoint)
	len2 := float64(d.Dot(d))
	if len2 == 0 {
		return -1
	}
	return float64(seg.StartPoint.VectorTo(p).Dot(d)) / len2
}
