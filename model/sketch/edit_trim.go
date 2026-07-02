// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"
	"sort"

	"oblikovati.org/math"
)

// SplitLine splits a line at the point nearest pick into two collinear lines sharing a
// new midpoint, returning both. It errors if the pick is at (or beyond) an endpoint.
func (s *Sketch) SplitLine(l *Line, pick math.Point2) ([]Entity, error) {
	t := projectParamOnLine(l, pick)
	if t <= 1e-9 || t >= 1-1e-9 {
		return nil, fmt.Errorf("split: point projects to t=%.4g, not strictly inside the line", t)
	}
	mid := s.newPoint(l.A.Position().Lerp(l.B.Position(), t))
	second := s.lines.Add(mid, l.B)
	l.B = mid
	return []Entity{l, second}, nil
}

// TrimLine removes the segment of l containing pick, cutting at the nearest intersections
// with other lines (the most common case; curve intersections are a follow-up). Returns
// the surviving line(s). If pick lies before any intersection, an end stub is removed.
func (s *Sketch) TrimLine(l *Line, pick math.Point2) ([]Entity, error) {
	cuts := append([]float64{0, 1}, s.lineEntityCrossings(l)...)
	sort.Float64s(cuts)
	cuts = dedupeSorted(cuts)
	lo, hi, ok := bracketParam(cuts, projectParamOnLine(l, pick))
	if !ok {
		return nil, fmt.Errorf("trim: no segment found for the pick point")
	}
	return s.reshapeTrimmed(l, lo, hi), nil
}

// reshapeTrimmed rebuilds line l with the [lo, hi] segment removed.
func (s *Sketch) reshapeTrimmed(l *Line, lo, hi float64) []Entity {
	a, b := l.A.Position(), l.B.Position()
	switch {
	case lo <= 1e-9 && hi >= 1-1e-9: // whole line removed
		s.deleteEntity(l)
		return nil
	case lo <= 1e-9: // trim the front: keep [hi, 1]
		l.A = s.newPoint(a.Lerp(b, hi))
		return []Entity{l}
	case hi >= 1-1e-9: // trim the tail: keep [0, lo]
		l.B = s.newPoint(a.Lerp(b, lo))
		return []Entity{l}
	default: // interior gap: keep [0, lo] and [hi, 1]
		tail := s.lines.Add(s.newPoint(a.Lerp(b, hi)), s.newPoint(b))
		l.B = s.newPoint(a.Lerp(b, lo))
		return []Entity{l, tail}
	}
}

// ExtendLine lengthens l past the picked end (true ⇒ the B end) to the nearest crossing
// with another line's infinite support, returning l. It errors if nothing is reachable.
func (s *Sketch) ExtendLine(l *Line, atEnd bool) (*Line, error) {
	best, found := s.nearestExtension(l, atEnd)
	if !found {
		return nil, fmt.Errorf("extend: no line to extend to")
	}
	if atEnd {
		l.B = s.newPoint(best)
	} else {
		l.A = s.newPoint(best)
	}
	return l, nil
}

// nearestExtension returns the closest intersection of l's infinite support with another
// line, circle or arc, lying beyond the picked end. (Crossings are found via the kernel
// 2D intersection primitives in edit_trim_curves.go.)
func (s *Sketch) nearestExtension(l *Line, atEnd bool) (math.Point2, bool) {
	support, err := entityLine2d(l)
	if err != nil {
		return math.Point2{}, false
	}
	bestT, found := 0.0, false
	for _, e := range s.ents {
		if e == Entity(l) {
			continue
		}
		for _, p := range supportEntityHits(support, e) {
			t := projectParamOnLine(l, p)
			if pickBeyond(t, atEnd) && (!found || closerParam(t, bestT, atEnd)) {
				bestT, found = t, true
			}
		}
	}
	return l.A.Position().Lerp(l.B.Position(), bestT), found
}

// pickBeyond reports whether param t lies past the picked end of a [0,1] line.
func pickBeyond(t float64, atEnd bool) bool {
	if atEnd {
		return t > 1+1e-9
	}
	return t < -1e-9
}

// closerParam reports whether candidate is nearer the picked end than current.
func closerParam(candidate, current float64, atEnd bool) bool {
	if atEnd {
		return candidate < current
	}
	return candidate > current
}

// lineLineParams returns the parameters (t on l1, u on l2) of the support lines'
// intersection, or ok=false when parallel.
func lineLineParams(l1, l2 *Line) (float64, float64, bool) {
	a, b := l1.A.Position(), l1.B.Position()
	c, d := l2.A.Position(), l2.B.Position()
	r := b.VectorTo(a).Negate() // a→b
	sdir := d.VectorTo(c).Negate()
	denom := float64(r.X*sdir.Y - r.Y*sdir.X)
	if denom == 0 {
		return 0, 0, false
	}
	acx, acy := float64(c.X-a.X), float64(c.Y-a.Y)
	t := (acx*float64(sdir.Y) - acy*float64(sdir.X)) / denom
	u := (acx*float64(r.Y) - acy*float64(r.X)) / denom
	return t, u, true
}

// projectParamOnLine returns the parameter of pick projected onto the line a→b.
func projectParamOnLine(l *Line, pick math.Point2) float64 {
	a, b := l.A.Position(), l.B.Position()
	dx, dy := float64(b.X-a.X), float64(b.Y-a.Y)
	d2 := dx*dx + dy*dy
	if d2 == 0 {
		return 0
	}
	return (float64(pick.X-a.X)*dx + float64(pick.Y-a.Y)*dy) / d2
}

// bracketParam returns the adjacent cut params surrounding t (the picked segment).
func bracketParam(cuts []float64, t float64) (float64, float64, bool) {
	for i := 0; i+1 < len(cuts); i++ {
		if t >= cuts[i]-1e-9 && t <= cuts[i+1]+1e-9 {
			return cuts[i], cuts[i+1], true
		}
	}
	return 0, 0, false
}

// dedupeSorted removes near-equal neighbours from a sorted slice.
func dedupeSorted(xs []float64) []float64 {
	out := xs[:0]
	for i, x := range xs {
		if i == 0 || x-out[len(out)-1] > 1e-9 {
			out = append(out, x)
		}
	}
	return out
}
