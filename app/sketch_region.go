// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
	"oblikovati.org/renderer"
)

// Sketch box-select: in the sketch editor a rubber-band rectangle selects the 2D entities it
// covers. Each entity's outline (sketch.EntityOutline) is projected to screen through the sketch
// plane and the viewport camera, then classified — window (drag L→R) keeps entities fully
// enclosed by the box, crossing (R→L) also keeps entities the box merely touches. This is the
// sketch-env counterpart to RayPicker.PickRegion (#909).

// screenPt is a projected outline vertex in viewport pixels.
type screenPt struct{ x, y float64 }

// pickSketchRegion returns the active sketch's entities covered by the box-select rectangle.
func (s *Session) pickSketchRegion(x0, y0, x1, y1 float64, crossing bool) []Selectable {
	if s.activeSketch == nil {
		return nil
	}
	plane := s.activeSketch.Plane()
	rect := orderedRect(x0, y0, x1, y1)
	var hits []Selectable
	for _, e := range s.activeSketch.Entities() {
		pts := s.projectOutline(plane, sketch.EntityOutline(e))
		if len(pts) > 0 && regionCoversOutline(pts, rect, crossing) {
			hits = append(hits, SketchEntityHandle{Entity: e})
		}
	}
	return hits
}

// projectOutline maps an entity's sketch-space outline to viewport pixels through the sketch
// plane and camera, dropping vertices that fall at/behind the camera.
func (s *Session) projectOutline(plane sketch.Plane, outline []math.Point2) []screenPt {
	pts := make([]screenPt, 0, len(outline))
	for _, p := range outline {
		if sx, sy, ok := renderer.Project(s.camera, regionNear, regionFar, plane.ToModel(p)); ok {
			pts = append(pts, screenPt{sx, sy})
		}
	}
	return pts
}

// regionCoversOutline classifies a projected outline against the box: window (crossing=false)
// requires every vertex inside the rectangle; crossing requires any vertex inside OR any outline
// segment to cross a rectangle edge (so a long line passing through the box, all vertices outside,
// is still caught).
func regionCoversOutline(pts []screenPt, rect screenRect, crossing bool) bool {
	inside := 0
	for _, p := range pts {
		if rect.containsPoint(p.x, p.y) {
			inside++
		}
	}
	if !crossing {
		return inside == len(pts)
	}
	if inside > 0 {
		return true
	}
	for i := 1; i < len(pts); i++ {
		if segmentCrossesRect(pts[i-1], pts[i], rect) {
			return true
		}
	}
	return false
}

// segmentCrossesRect reports whether segment a–b intersects any edge of the rectangle (the
// endpoint-inside case is handled by the caller's vertex test).
func segmentCrossesRect(a, b screenPt, r screenRect) bool {
	corners := [4]screenPt{{r.minX, r.minY}, {r.maxX, r.minY}, {r.maxX, r.maxY}, {r.minX, r.maxY}}
	for i := range 4 {
		if segmentsIntersect(a, b, corners[i], corners[(i+1)%4]) {
			return true
		}
	}
	return false
}

// segmentsIntersect reports whether segments p1–p2 and p3–p4 properly cross, using the standard
// orientation (CCW) test.
func segmentsIntersect(p1, p2, p3, p4 screenPt) bool {
	d1 := orient(p3, p4, p1)
	d2 := orient(p3, p4, p2)
	d3 := orient(p1, p2, p3)
	d4 := orient(p1, p2, p4)
	return ((d1 > 0) != (d2 > 0)) && ((d3 > 0) != (d4 > 0))
}

// orient returns the signed area sign of triangle (a,b,c): >0 CCW, <0 CW, 0 collinear.
func orient(a, b, c screenPt) float64 {
	return (b.x-a.x)*(c.y-a.y) - (b.y-a.y)*(c.x-a.x)
}
