// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/math"
)

// snapPixels is the screen-space radius (in pixels) within which a sketch click snaps
// to existing geometry or the grid.
const snapPixels = 12

// SnapKind classifies what a cursor snapped to, so the viewport can draw the matching
// glyph (Inventor's snap markers) and the user can see the otherwise-1px snap point.
type SnapKind int

const (
	// SnapNone: no snap; the raw plane point is used.
	SnapNone SnapKind = iota
	// SnapPoint: an existing endpoint/vertex (a square glyph).
	SnapPoint
	// SnapMidpoint: the midpoint of a line (a triangle glyph).
	SnapMidpoint
	// SnapOnCurve: a point on a line/circle edge (a cross glyph).
	SnapOnCurve
	// SnapGrid: a grid intersection.
	SnapGrid
)

// SnapResult is a snapped point plus what it snapped to.
type SnapResult struct {
	Point math.Point2
	Kind  SnapKind
}

// sketchClickPoint maps a viewport pixel to a snapped point in the active sketch plane —
// what a geometry-tool click places (latching onto existing geometry/grid).
func (s *Session) sketchClickPoint(px, py float64) (math.Point2, bool) {
	raw, ok := screenToSketch(s, px, py)
	if !ok {
		return math.Point2{}, false
	}
	return s.snapAt(raw).Point, true
}

// SnapAt maps a viewport pixel to its snap (point + kind), reporting false when nothing
// snapped — the viewport uses it to draw the snap glyph under the cursor.
func (s *Session) SnapAt(px, py float64) (SnapResult, bool) {
	raw, ok := screenToSketch(s, px, py)
	if !ok {
		return SnapResult{}, false
	}
	r := s.snapAt(raw)
	return r, r.Kind != SnapNone
}

// snapAt snaps a plane point by priority: existing endpoint, then line midpoint, then a
// point on a curve, then the grid — honoring the snap preferences.
func (s *Session) snapAt(p math.Point2) SnapResult {
	tol := s.snapTolerance()
	if tol <= 0 || s.activeSketch == nil {
		return SnapResult{Point: p}
	}
	if s.Grid().SnapToPoints {
		if q, ok := s.nearestSketchPoint(p, tol); ok {
			return SnapResult{q, SnapPoint}
		}
		if q, ok := s.nearestMidpoint(p, tol); ok {
			return SnapResult{q, SnapMidpoint}
		}
		if q, ok := s.nearestOnCurve(p, tol); ok {
			return SnapResult{q, SnapOnCurve}
		}
	}
	if q, ok := s.nearestGridPoint(p, tol); ok {
		return SnapResult{q, SnapGrid}
	}
	return SnapResult{Point: p}
}

// snapTolerance is the world-space snap radius (snapPixels scaled by the current zoom).
func (s *Session) snapTolerance() float64 { return snapPixels * s.camera.WorldPerPixel() }

// nearestSketchPoint returns the existing sketch point closest to p within tol (and the
// origin, always a snap target).
func (s *Session) nearestSketchPoint(p math.Point2, tol float64) (math.Point2, bool) {
	best, found := math.Point2{}, false
	bestD := tol
	consider := func(c math.Point2) {
		if d := p.DistanceTo(c); d <= bestD {
			best, bestD, found = c, d, true
		}
	}
	consider(math.P2(0, 0)) // the sketch origin is always snappable
	for _, pt := range s.activeSketch.AllPoints() {
		consider(pt.Position())
	}
	return best, found
}

// nearestMidpoint returns the closest line midpoint within tol.
func (s *Session) nearestMidpoint(p math.Point2, tol float64) (math.Point2, bool) {
	best, found := math.Point2{}, false
	bestD := tol
	lines := s.activeSketch.Lines()
	for i := 0; i < lines.Count(); i++ {
		l := lines.Item(i)
		mid := l.A.Position().Midpoint(l.B.Position())
		if d := p.DistanceTo(mid); d <= bestD {
			best, bestD, found = mid, d, true
		}
	}
	return best, found
}

// nearestOnCurve returns the closest point on a line segment, circle, or arc outline
// within tol — the "snap to edge" target. Arcs snap to their full circle, matching how
// selection (nearestEntityCurve) measures arc distance.
func (s *Session) nearestOnCurve(p math.Point2, tol float64) (math.Point2, bool) {
	best, found := math.Point2{}, false
	bestD := tol
	consider := func(q math.Point2) {
		if d := p.DistanceTo(q); d <= bestD {
			best, bestD, found = q, d, true
		}
	}
	sk := s.activeSketch
	for i := 0; i < sk.Lines().Count(); i++ {
		l := sk.Lines().Item(i)
		consider(segmentClosestPoint(p, l.A.Position(), l.B.Position()))
	}
	for i := 0; i < sk.Circles().Count(); i++ {
		c := sk.Circles().Item(i)
		consider(circleClosestPoint(p, c.Center.Position(), c.Radius))
	}
	for i := 0; i < sk.Arcs().Count(); i++ {
		a := sk.Arcs().Item(i)
		consider(circleClosestPoint(p, a.Center.Position(), a.Radius()))
	}
	return best, found
}

// nearestGridPoint returns the grid intersection closest to p within tol.
func (s *Session) nearestGridPoint(p math.Point2, tol float64) (math.Point2, bool) {
	g := s.Grid()
	if !g.SnapToGrid {
		return math.Point2{}, false
	}
	step := g.SpacingModel()
	if step <= 0 {
		return math.Point2{}, false
	}
	q := math.P2(stdmath.Round(p.X/step)*step, stdmath.Round(p.Y/step)*step)
	if p.DistanceTo(q) <= tol {
		return q, true
	}
	return math.Point2{}, false
}

// segmentClosestPoint returns the point on segment a–b nearest p.
func segmentClosestPoint(p, a, b math.Point2) math.Point2 {
	ab := a.VectorTo(b)
	lenSq := ab.Dot(ab)
	if lenSq < math.DefaultTolerance {
		return a
	}
	t := stdmath.Max(0, stdmath.Min(1, a.VectorTo(p).Dot(ab)/lenSq))
	return a.TranslateBy(ab.Scale(t))
}

// circleOutlineDistance returns the unsigned distance from p to a circle's outline (used
// to pick circles and arcs, which both measure distance to their full circle).
func circleOutlineDistance(p, center math.Point2, r float64) float64 {
	return stdmath.Abs(p.DistanceTo(center) - r)
}

// circleClosestPoint returns the point on a circle's outline nearest p (center when p is
// at the center).
func circleClosestPoint(p, center math.Point2, r float64) math.Point2 {
	dir := center.VectorTo(p)
	if dir.Length() < math.DefaultTolerance {
		return math.P2(center.X+r, center.Y)
	}
	u := dir.Scale(r / dir.Length())
	return center.TranslateBy(u)
}
