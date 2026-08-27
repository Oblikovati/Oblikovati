// SPDX-License-Identifier: GPL-2.0-only

package sketch

import "oblikovati.org/math"

// Closest point on an entity, for any entity kind.
//
// It works off [EntityPolyline], the same traversal the renderer and the exporters sample, so a
// curve kind added later is handled without registering it anywhere — the property the constraint
// marker placement is built on.

// ClosestPointOnEntity returns the point of e nearest target, and false for an entity with no
// curve to speak of (a text box, an image). The result lies on the entity's sampled outline, so
// for a curved entity it is exact to the sampling density rather than analytically exact — which
// is what a screen annotation needs.
//
//	at, ok := ClosestPointOnEntity(circle, math.P2(10, 0)) // the near side of the circle
func ClosestPointOnEntity(e Entity, target math.Point2) (math.Point2, bool) {
	pts, closed := EntityPolyline(e)
	if len(pts) == 0 {
		return math.Point2{}, false
	}
	if len(pts) == 1 {
		return pts[0], true
	}
	return closestOnPolyline(pts, closed, target), true
}

// closestOnPolyline returns the point of the polyline nearest target, walking every segment —
// including the closing one when the outline is closed, without which the nearest point on a
// circle could never fall in its final span.
func closestOnPolyline(pts []math.Point2, closed bool, target math.Point2) math.Point2 {
	best, bestD := pts[0], target.DistanceSquaredTo(pts[0])
	for i := 0; i+1 < len(pts); i++ {
		best, bestD = nearerOnSegment(pts[i], pts[i+1], target, best, bestD)
	}
	if closed {
		best, _ = nearerOnSegment(pts[len(pts)-1], pts[0], target, best, bestD)
	}
	return best
}

// nearerOnSegment returns whichever of the current best and the segment's closest point is nearer
// target, with its squared distance.
func nearerOnSegment(a, b, target, best math.Point2, bestD float64) (math.Point2, float64) {
	p := closestOnSegment(a, b, target)
	if d := target.DistanceSquaredTo(p); d < bestD {
		return p, d
	}
	return best, bestD
}

// closestOnSegment returns the point of segment a–b nearest target: the projection of target onto
// the segment's line, clamped to the segment so the result never runs off the end.
func closestOnSegment(a, b, target math.Point2) math.Point2 {
	seg := a.VectorTo(b)
	lenSq := seg.LengthSquared()
	if lenSq == 0 {
		return a // a degenerate segment is just its own endpoint
	}
	return a.Lerp(b, math.Clamp01(a.VectorTo(target).Dot(seg)/lenSq))
}
