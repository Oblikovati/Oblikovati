// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// curveHitTol is the angular/positional slack used when filtering a circle crossing onto
// an arc's sweep (so an intersection at an arc endpoint still counts).
const curveHitTol = 1e-7

// This file generalizes line trim/extend to cut against circles and arcs, not only other
// lines (the follow-up noted in edit_trim.go). The sketch entities are adapted to the
// kernel's analytic 2D types and the exact intersection primitives in kernel/geom do the
// geometry; arc crossings are circle crossings filtered to the arc's sweep.

// entitySegment2d adapts a sketch line to a kernel 2D segment.
func entitySegment2d(l *Line) geom.LineSegment2d {
	return geom.NewLineSegment2d(l.A.Position(), l.B.Position())
}

// entityLine2d adapts a sketch line to a kernel 2D *infinite* line (A along A→B). It
// errors on a degenerate (zero-length) line.
func entityLine2d(l *Line) (geom.Line2d, error) {
	return geom.NewLine2d(l.A.Position(), l.Direction())
}

// entityCircle2d adapts a sketch circle to a kernel 2D circle.
func entityCircle2d(c *Circle) geom.Circle2d {
	return geom.NewCircle2d(c.Center.Position(), float64(c.Radius))
}

// entityArc2d adapts a sketch arc to a kernel 2D arc, deriving the start angle and signed
// sweep (CCW positive) from its endpoints and winding flag.
func entityArc2d(a *Arc) geom.Arc2d {
	c := a.Center.Position()
	start := angleOfPoint(c, a.Start.Position())
	end := angleOfPoint(c, a.End.Position())
	sweep := wrap2pi(end - start)
	if !a.CounterClockwise {
		sweep = -wrap2pi(start - end)
	}
	return geom.NewArc2d(c, float64(a.Radius()), start, sweep)
}

// wrap2pi maps an angle delta into [0, 2π).
func wrap2pi(d float64) float64 {
	d = stdmath.Mod(d, 2*stdmath.Pi)
	if d < 0 {
		d += 2 * stdmath.Pi
	}
	return d
}

// segmentEntityHits returns the points where segment seg crosses entity e (a line, circle
// or arc); other entity kinds yield none.
func segmentEntityHits(seg geom.LineSegment2d, e Entity) []math.Point2 {
	switch g := e.(type) {
	case *Line:
		if p, _, _, ok := geom.Segment2dIntersection(seg, entitySegment2d(g), 0); ok {
			return []math.Point2{p}
		}
	case *Circle:
		return geom.SegmentCircle2dIntersection(seg, entityCircle2d(g), 0)
	case *Arc:
		return arcFilter(entityArc2d(g), geom.SegmentCircle2dIntersection(seg, arcCircle(g), 0))
	}
	return nil
}

// supportEntityHits returns the points where the infinite line crosses entity e — used by
// extend, which reaches past the line's endpoints to the nearest support crossing.
func supportEntityHits(line geom.Line2d, e Entity) []math.Point2 {
	switch g := e.(type) {
	case *Line:
		if other, err := entityLine2d(g); err == nil {
			if p, ok := geom.Line2dIntersection(line, other, 0); ok {
				return []math.Point2{p}
			}
		}
	case *Circle:
		return geom.LineCircle2dIntersection(line, entityCircle2d(g), 0)
	case *Arc:
		return arcFilter(entityArc2d(g), geom.LineCircle2dIntersection(line, arcCircle(g), 0))
	}
	return nil
}

// arcCircle is the full circle underlying an arc.
func arcCircle(a *Arc) geom.Circle2d {
	return geom.NewCircle2d(a.Center.Position(), float64(a.Radius()))
}

// arcFilter keeps only the points that lie within the arc's sweep.
func arcFilter(arc geom.Arc2d, pts []math.Point2) []math.Point2 {
	var out []math.Point2
	for _, p := range pts {
		if arc.ContainsPoint(p, curveHitTol) {
			out = append(out, p)
		}
	}
	return out
}

// lineEntityCrossings returns the parameters in (0,1) along l where it crosses any other
// line, circle or arc in the sketch. It supersedes the line-only lineCrossings.
func (s *Sketch) lineEntityCrossings(l *Line) []float64 {
	seg := entitySegment2d(l)
	var out []float64
	for _, e := range s.ents {
		if e == Entity(l) {
			continue
		}
		for _, p := range segmentEntityHits(seg, e) {
			if t := projectParamOnLine(l, p); t > 1e-9 && t < 1-1e-9 {
				out = append(out, t)
			}
		}
	}
	return out
}
