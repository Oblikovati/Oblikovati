// SPDX-License-Identifier: GPL-2.0-only

package geom

import "github.com/Oblikovati/oblikovati/math"

// This file holds the package-level geometric queries (the modern form of the
// COM GeometryUtilities / TransientGeometry intersection methods — plain
// functions, no object, per ADR-0006): the closed-form analytic queries used by
// snapping, constraints, and measurement. General numeric curve/surface
// intersection is a kernel numeric-phase concern (M06/M07), not implemented
// here.

// ClosestParamOnLine returns the line parameter t whose point is nearest p
// (the scalar projection of p onto the line).
func ClosestParamOnLine(l Line, p math.Point3) float64 {
	return l.Origin.VectorTo(p).Dot(l.Dir.AsVector())
}

// ClosestPointOnLine returns the point on the infinite line nearest p.
func ClosestPointOnLine(l Line, p math.Point3) math.Point3 {
	return l.PointAt(ClosestParamOnLine(l, p))
}

// DistancePointToLine returns the perpendicular distance from p to the line.
func DistancePointToLine(l Line, p math.Point3) float64 {
	return p.DistanceTo(ClosestPointOnLine(l, p))
}

// ClosestPointOnSegment returns the point on the bounded segment nearest p,
// clamping the projection to the segment's [0,1] parameter range.
func ClosestPointOnSegment(s LineSegment, p math.Point3) math.Point3 {
	d := s.StartPoint.VectorTo(s.EndPoint)
	len2 := d.LengthSquared()
	if len2 == 0 {
		return s.StartPoint
	}
	return s.PointAt(clamp01(s.StartPoint.VectorTo(p).Dot(d) / len2))
}

// DistancePointToSegment returns the distance from p to the nearest point of
// the segment.
func DistancePointToSegment(s LineSegment, p math.Point3) float64 {
	return p.DistanceTo(ClosestPointOnSegment(s, p))
}

// SignedDistanceToPlane returns the signed distance from p to the plane,
// positive on the side the plane normal points toward.
func SignedDistanceToPlane(pl Plane, p math.Point3) float64 {
	return pl.Origin.VectorTo(p).Dot(pl.Normal())
}

// ProjectPointToPlane returns the orthogonal projection of p onto the plane.
func ProjectPointToPlane(pl Plane, p math.Point3) math.Point3 {
	return p.TranslateBy(pl.Normal().Scale(-SignedDistanceToPlane(pl, p)))
}

// LinePlaneIntersection returns the point where the line meets the plane, and
// true. It returns false when the line is parallel to the plane (within tol on
// the direction·normal dot product); pass tol <= 0 for the default.
func LinePlaneIntersection(l Line, pl Plane, tol float64) (math.Point3, bool) {
	n := pl.Normal()
	denom := l.Dir.AsVector().Dot(n)
	if math.IsNearZero(denom, tol) {
		return math.Point3{}, false
	}
	t := l.Origin.VectorTo(pl.Origin).Dot(n) / denom
	return l.PointAt(t), true
}

// LineLineClosest returns the pair of nearest points (one on each line) and
// true. It returns false when the lines are parallel, where no unique nearest
// pair exists. Pass tol <= 0 for the default parallelism tolerance.
func LineLineClosest(a, b Line, tol float64) (onA, onB math.Point3, ok bool) {
	d1, d2 := a.Dir.AsVector(), b.Dir.AsVector()
	w0 := b.Origin.VectorTo(a.Origin)
	bb := d1.Dot(d2)
	denom := 1 - bb*bb // d1·d1 = d2·d2 = 1 (unit directions)
	if math.IsNearZero(denom, tol) {
		return math.Point3{}, math.Point3{}, false
	}
	d, e := d1.Dot(w0), d2.Dot(w0)
	sc := (bb*e - d) / denom
	tc := (e - bb*d) / denom
	return a.PointAt(sc), b.PointAt(tc), true
}

// LineLineIntersection returns the intersection point of two lines when they
// actually meet — i.e. their closest points coincide within tol (also the
// parallelism tolerance). Pass tol <= 0 for the default.
func LineLineIntersection(a, b Line, tol float64) (math.Point3, bool) {
	onA, onB, ok := LineLineClosest(a, b, tol)
	if !ok || !onA.IsEqualTo(onB, tol) {
		return math.Point3{}, false
	}
	return onA.Midpoint(onB), true
}

// Line2dIntersection returns the intersection point of two 2D lines, and true.
// It returns false when the lines are parallel (within tol on the direction
// cross product); pass tol <= 0 for the default.
func Line2dIntersection(a, b Line2d, tol float64) (math.Point2, bool) {
	da, db := a.Dir.AsVector(), b.Dir.AsVector()
	cross := da.Cross(db)
	if math.IsNearZero(cross, tol) {
		return math.Point2{}, false
	}
	s := a.Origin.VectorTo(b.Origin).Cross(db) / cross
	return a.PointAt(s), true
}
