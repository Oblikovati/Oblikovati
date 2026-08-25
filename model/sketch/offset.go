// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/math"
)

// OffsetEntity creates a copy of a single line/circle/arc offset by signed distance d:
// a parallel line (offset to the left of A→B for d>0), or a concentric circle/arc of
// radius r+d. It errors for an unsupported entity or a non-positive resulting radius.
//
//	off, err := sk.OffsetEntity(line, 0.5)
func (s *Sketch) OffsetEntity(e Entity, d math.Scalar) (Entity, error) {
	switch v := e.(type) {
	case *Line:
		return s.offsetLine(v, d), nil
	case *Circle:
		return s.offsetCircle(v, d)
	case *Arc:
		return s.offsetArc(v, d)
	case *ProjectedCurve:
		return s.offsetProjectedCurve(v, d)
	default:
		return nil, fmt.Errorf("offset: unsupported entity %T (want line/circle/arc or a projected curve)", e)
	}
}

// projectedOffsetArcSegs is how finely a convex corner opened by the offset is rounded when
// offsetting a projected polyline (matching OffsetClosedLoop's own default fidelity).
const projectedOffsetArcSegs = 8

// offsetProjectedCurve offsets a projected reference curve — a sampled polyline, e.g. a projected
// face perimeter or model edge — by d as new sketch line geometry: a closed perimeter offsets as a
// closed loop (an inner offset for d<0, matching offsetCircle's sign), an open edge as an open
// chain. The projection is already faceted, so its offset is faceted too. This is the offset a user
// reaches by selecting a projected perimeter (#2158 follow-up); before it, OffsetEntity rejected the
// projected curve outright.
func (s *Sketch) offsetProjectedCurve(pc *ProjectedCurve, d math.Scalar) (Entity, error) {
	pts := pc.Points()
	if polylineReturnsToStart(pts) {
		if ents := s.OffsetClosedLoop(pts, float64(d), projectedOffsetArcSegs); len(ents) > 0 {
			return ents[0], nil
		}
		return nil, fmt.Errorf("offset: projected loop of %d points collapses when offset by %.4g", len(pts), float64(d))
	}
	off := offsetPolyline(pts, float64(d))
	if len(off) < 2 {
		return nil, fmt.Errorf("offset: projected curve of %d points cannot offset by %.4g", len(pts), float64(d))
	}
	return s.addOpenPolyline(off), nil
}

// addOpenPolyline adds an open chain of line segments through pts and returns its first segment (the
// representative entity OffsetEntity's contract expects; the offset tool ignores it and keeps all).
func (s *Sketch) addOpenPolyline(pts []math.Point2) Entity {
	nodes := make([]*Point, len(pts))
	for i, p := range pts {
		nodes[i] = s.points.Add(p)
	}
	var first Entity
	for i := 0; i+1 < len(nodes); i++ {
		if e := s.lines.Add(nodes[i], nodes[i+1]); first == nil {
			first = e
		}
	}
	return first
}

// offsetLine returns a line parallel to l, shifted by d along l's left normal.
func (s *Sketch) offsetLine(l *Line, d math.Scalar) *Line {
	u, ok := unitVec(l.A.Position().VectorTo(l.B.Position()))
	if !ok {
		return s.lines.AddByTwoPoints(l.A.Position(), l.B.Position()) // degenerate: copy in place
	}
	n := math.V2(-u.Y, u.X).Scale(float64(d))
	return s.lines.AddByTwoPoints(l.A.Position().TranslateBy(n), l.B.Position().TranslateBy(n))
}

// offsetCircle returns a concentric circle of radius r+d.
func (s *Sketch) offsetCircle(c *Circle, d math.Scalar) (*Circle, error) {
	r := c.Radius + d
	if r <= 0 {
		return nil, fmt.Errorf("offset: circle radius %.4g + %.4g ≤ 0", float64(c.Radius), float64(d))
	}
	return s.circles.AddByCenterRadius(c.Center.Position(), r), nil
}

// offsetArc returns a concentric arc of radius r+d (endpoints moved radially).
func (s *Sketch) offsetArc(a *Arc, d math.Scalar) (*Arc, error) {
	center := a.Center.Position()
	r := a.Radius()
	nr := r + d
	if nr <= 0 {
		return nil, fmt.Errorf("offset: arc radius %.4g + %.4g ≤ 0", float64(r), float64(d))
	}
	scale := float64(nr) / float64(r)
	start := center.TranslateBy(center.VectorTo(a.Start.Position()).Scale(scale))
	end := center.TranslateBy(center.VectorTo(a.End.Position()).Scale(scale))
	return s.arcs.AddByCenterStartEnd(center, start, end, a.CounterClockwise), nil
}
