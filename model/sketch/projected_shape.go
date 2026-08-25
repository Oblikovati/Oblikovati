// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// A projected curve keeps its sampled polyline, but Inventor projects an analytic edge as analytic
// reference geometry — a line as a line, an arc as an arc, a circle as a circle — not the faceted
// chain a coplanar/parallel projection really is up to sampling. Detecting the shape from the
// projected 2D points is exact for those cases (a coplanar projection is the identity; an
// orthographic projection onto a parallel plane preserves the circle) and degrades to the polyline
// for an oblique projection, where a circle becomes an ellipse (#2158 follow-up). Reading only the
// 2D points keeps this independent of the source's analytic curve.

type projectedShapeKind uint8

const (
	shapeNone projectedShapeKind = iota
	shapeLine
	shapeArc
	shapeCircle
)

// projectedShape is the analytic form of a projected curve on the sketch plane, when it has one.
type projectedShape struct {
	kind         projectedShapeKind
	a, b         math.Point2 // line endpoints
	center       math.Point2 // arc/circle centre
	radius       float64
	start, sweep float64 // arc start angle and signed sweep (radians); unused for a circle
}

// projShapeTol is the relative residual within which the projected points must lie on the fitted
// line or circle for the shape to count as analytic — a coplanar/parallel projection fits to float
// error, so it is tight; an ellipse (oblique arc) fails it and falls back to the polyline.
const projShapeTol = 1e-6 // tol:calibrated — projected-shape line/circle fit residual

// projectedRenderSegments is how finely an analytic projected shape is sampled for drawing and
// hit-testing, so a projected arc reads as a smooth curve rather than the 16 source facets.
const projectedRenderSegments = 64

// fitProjectedShape classifies a projected polyline as a line, arc, circle, or (shapeNone) an
// arbitrary polyline, from its 2D points alone. A real projection samples one edge into many points
// along a smooth curve, so "every point on one circle" reliably means a circular arc (an ellipse —
// an obliquely projected arc — fails it and stays a polyline). Closure is read before de-duplicating
// so a full circle's coincident first/last point is not lost.
func fitProjectedShape(pts []math.Point2) projectedShape {
	closed := polylineReturnsToStart(pts)
	pts = dropDuplicateVertices(pts)
	n := len(pts)
	if n < 2 {
		return projectedShape{}
	}
	if !closed && collinearPolyline(pts) {
		return projectedShape{kind: shapeLine, a: pts[0], b: pts[n-1]}
	}
	if n < 3 {
		return projectedShape{}
	}
	center, radius, ok := fitCircleThrough(pts)
	if !ok || !allOnCircle(pts, center, radius) {
		return projectedShape{}
	}
	if closed {
		return projectedShape{kind: shapeCircle, center: center, radius: radius}
	}
	start, sweep := arcSpan(pts, center)
	return projectedShape{kind: shapeArc, center: center, radius: radius, start: start, sweep: sweep}
}

// collinearPolyline reports whether every vertex lies on the chord through the first and last, so a
// projected straight edge (however finely sampled) is one line.
func collinearPolyline(pts []math.Point2) bool {
	a, b := pts[0], pts[len(pts)-1]
	chord := float64(a.DistanceTo(b))
	if chord < 1e-12 {
		return false // a closed or degenerate chain is not a line
	}
	tol := projShapeTol * chord
	for _, p := range pts[1 : len(pts)-1] {
		if stdmath.Abs(perpDistanceToLine(a, b, p)) > tol {
			return false
		}
	}
	return true
}

// fitCircleThrough fits a circle to three well-spread points of the polyline (0, n/3, 2n/3), which
// avoids the near-coincident first/last pair of a closed loop.
func fitCircleThrough(pts []math.Point2) (math.Point2, float64, bool) {
	n := len(pts)
	center, radius, err := circumcircle(pts[0], pts[n/3], pts[2*n/3])
	if err != nil {
		return math.Point2{}, 0, false
	}
	return center, float64(radius), true
}

// allOnCircle reports whether every vertex is the fitted radius from the centre, within the fit
// tolerance scaled by the radius.
func allOnCircle(pts []math.Point2, center math.Point2, radius float64) bool {
	tol := projShapeTol * radius
	for _, p := range pts {
		if stdmath.Abs(float64(p.DistanceTo(center))-radius) > tol {
			return false
		}
	}
	return true
}

// arcSpan returns the start angle and signed sweep of the open arc through the polyline about
// centre, choosing the direction that passes through the polyline's middle sample.
func arcSpan(pts []math.Point2, center math.Point2) (start, sweep float64) {
	ang := func(p math.Point2) float64 {
		return stdmath.Atan2(float64(p.Y-center.Y), float64(p.X-center.X))
	}
	start = ang(pts[0])
	ccw := wrap2pi(ang(pts[len(pts)-1]) - start)
	if wrap2pi(ang(pts[len(pts)/2])-start) <= ccw {
		return start, ccw // the middle sample lies on the CCW path start→end
	}
	return start, ccw - 2*stdmath.Pi // otherwise the arc runs the other way (CW)
}

// polyline samples the analytic shape into points for drawing and hit-testing; nil for shapeNone.
func (s projectedShape) polyline() []math.Point2 {
	switch s.kind {
	case shapeLine:
		return []math.Point2{s.a, s.b}
	case shapeCircle:
		return sampleShapeArc(s.center, s.radius, 0, 2*stdmath.Pi)
	case shapeArc:
		return sampleShapeArc(s.center, s.radius, s.start, s.sweep)
	default:
		return nil
	}
}

// sampleShapeArc samples a circular arc into projectedRenderSegments segments.
func sampleShapeArc(center math.Point2, radius, start, sweep float64) []math.Point2 {
	pts := make([]math.Point2, projectedRenderSegments+1)
	for i := range pts {
		a := start + sweep*float64(i)/float64(projectedRenderSegments)
		pts[i] = math.P2(center.X+math.Scalar(radius*stdmath.Cos(a)), center.Y+math.Scalar(radius*stdmath.Sin(a)))
	}
	return pts
}

// perpDistanceToLine is the signed perpendicular distance from p to the infinite line through a→b.
func perpDistanceToLine(a, b, p math.Point2) float64 {
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	length := stdmath.Hypot(dx, dy)
	if length < 1e-12 {
		return float64(p.DistanceTo(a))
	}
	return (dx*float64(p.Y-a.Y) - dy*float64(p.X-a.X)) / length
}
