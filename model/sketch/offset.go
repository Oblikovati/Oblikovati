// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati/math"
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
	default:
		return nil, fmt.Errorf("offset: unsupported entity %T (want line/circle/arc)", e)
	}
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
