// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Trimming a circle or arc as the target (not just a cutting curve). A circle trimmed at
// its crossings becomes the complementary arc; an arc trimmed at its crossings keeps the
// remaining arc(s). Crossing geometry reuses the kernel 2D intersection primitives.

// circleEntityHits returns the points where circle cc crosses every entity except exclude.
func (s *Sketch) circleEntityHits(cc geom.Circle2d, exclude Entity) []math.Point2 {
	var out []math.Point2
	for _, e := range s.ents {
		if e == exclude {
			continue
		}
		switch g := e.(type) {
		case *Line:
			out = append(out, geom.SegmentCircle2dIntersection(entitySegment2d(g), cc, 0)...)
		case *Circle:
			out = append(out, geom.Circle2dCircle2dIntersection(cc, entityCircle2d(g), 0)...)
		case *Arc:
			out = append(out, arcFilter(entityArc2d(g), geom.Circle2dCircle2dIntersection(cc, arcCircle(g), 0))...)
		}
	}
	return out
}

// TrimCircle removes the picked arc of the circle (the gap between the two crossings
// bracketing the pick) and replaces the circle with the complementary arc. It needs at
// least two crossings; otherwise the closed circle cannot be opened.
func (s *Sketch) TrimCircle(c *Circle, pick math.Point2) ([]Entity, error) {
	center := c.Center.Position()
	var angs []float64
	for _, p := range s.circleEntityHits(entityCircle2d(c), Entity(c)) {
		angs = append(angs, wrap2pi(angleOfPoint(center, p)))
	}
	sort.Float64s(angs)
	angs = dedupeSorted(angs)
	if len(angs) < 2 {
		return nil, fmt.Errorf("trim: a circle needs at least two crossings to open, found %d", len(angs))
	}
	lo, hi := circularBracket(angs, wrap2pi(angleOfPoint(center, pick)))
	r := c.Radius
	// Removed gap is lo→hi CCW; the kept arc is its complement, hi→lo CCW.
	arc := s.arcs.AddByCenterStartEnd(center, pointAtAngle(center, r, hi), pointAtAngle(center, r, lo), true)
	s.deleteEntity(c)
	return []Entity{arc}, nil
}

// TrimArc removes the picked sub-arc (between adjacent crossings, with the arc's own ends
// as boundaries) and keeps the remaining arc(s) — one when an end stub is removed, two
// when an interior bite is taken.
func (s *Sketch) TrimArc(a *Arc, pick math.Point2) ([]Entity, error) {
	arc := entityArc2d(a)
	cuts := []float64{0, 1}
	for _, p := range s.circleEntityHits(arcCircle(a), Entity(a)) {
		if t, ok := arcParamOf(arc, p); ok && t > 1e-9 && t < 1-1e-9 {
			cuts = append(cuts, t)
		}
	}
	sort.Float64s(cuts)
	cuts = dedupeSorted(cuts)
	pt, _ := arcParamOf(arc, pick)
	lo, hi, ok := bracketParam(cuts, pt)
	if !ok {
		return nil, fmt.Errorf("trim: no arc segment found for the pick point (t=%.4g)", pt)
	}
	return s.reshapeTrimmedArc(a, arc, lo, hi), nil
}

// reshapeTrimmedArc rebuilds arc a with the [lo, hi] parameter span removed.
func (s *Sketch) reshapeTrimmedArc(a *Arc, arc geom.Arc2d, lo, hi float64) []Entity {
	switch {
	case lo <= 1e-9 && hi >= 1-1e-9: // whole arc removed
		s.deleteEntity(a)
		return nil
	case lo <= 1e-9: // trim the front: keep [hi, 1]
		a.Start = s.newPoint(arc.PointAt(hi))
		return []Entity{a}
	case hi >= 1-1e-9: // trim the tail: keep [0, lo]
		a.End = s.newPoint(arc.PointAt(lo))
		return []Entity{a}
	default: // interior bite: keep [0, lo] and [hi, 1]
		end := a.End.Position()
		tail := s.arcs.AddByCenterStartEnd(a.Center.Position(), arc.PointAt(hi), end, a.CounterClockwise)
		a.End = s.newPoint(arc.PointAt(lo))
		return []Entity{a, tail}
	}
}

// arcParamOf returns the parameter t∈[0,1] of point p along the arc (by angle), and
// whether it lies within the arc's sweep.
func arcParamOf(arc geom.Arc2d, p math.Point2) (float64, bool) {
	rel := angleOfPoint(arc.Center, p) - arc.StartAngle
	if arc.SweepAngle < 0 {
		rel = -rel
	}
	t := wrap2pi(rel) / stdmath.Abs(arc.SweepAngle)
	return t, t <= 1+1e-7
}

// circularBracket returns the adjacent pair of (sorted, unique, [0,2π)) crossing angles
// whose counter-clockwise gap contains pa — the segment the user picked to remove.
func circularBracket(angs []float64, pa float64) (lo, hi float64) {
	for i := 0; i+1 < len(angs); i++ {
		if pa >= angs[i] && pa < angs[i+1] {
			return angs[i], angs[i+1]
		}
	}
	return angs[len(angs)-1], angs[0] // the wrap gap through 0
}

// pointAtAngle returns the point at angle ang on a circle of the given center and radius.
func pointAtAngle(center math.Point2, r math.Scalar, ang float64) math.Point2 {
	return math.P2(center.X+r*math.Scalar(stdmath.Cos(ang)), center.Y+r*math.Scalar(stdmath.Sin(ang)))
}
