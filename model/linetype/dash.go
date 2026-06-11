// SPDX-License-Identifier: GPL-2.0-only

package linetype

import "oblikovati.org/math"

// dotLength is the pen-down length a pattern dot (0) renders as — long enough to
// survive rasterization at typical sketch zoom, short enough to read as a dot.
const dotLength = 0.02 // cm

// minCycle guards against degenerate patterns whose elements sum to ~zero, which
// would otherwise loop forever along an edge.
const minCycle = 1e-6

// DashPolyline splits a polyline into the pen-down segments of a .lin pattern,
// flowing the pattern continuously across vertices. A nil/degenerate pattern
// returns nil — the caller draws the polyline solid.
//
//	segs := linetype.DashPolyline(pts, false, linetype.Builtin(types.SketchLineCenter))
func DashPolyline(pts []math.Point2, closed bool, pattern []float64) [][2]math.Point2 {
	steps, cycle := penSteps(pattern)
	if len(pts) < 2 || cycle < minCycle {
		return nil
	}
	if closed {
		pts = append(append([]math.Point2{}, pts...), pts[0])
	}
	segs := [][2]math.Point2{}
	cur := dashCursor{steps: steps, rem: steps[0].length}
	for i := 1; i < len(pts); i++ {
		segs = cur.walkEdge(segs, pts[i-1], pts[i])
	}
	return segs
}

// penStep is one normalized pattern element: a pen-down or pen-up run.
type penStep struct {
	length float64
	down   bool
}

// penSteps normalizes a .lin pattern (dash/gap/dot) into pen runs and the total
// cycle length.
func penSteps(pattern []float64) ([]penStep, float64) {
	steps := make([]penStep, 0, len(pattern))
	cycle := 0.0
	for _, v := range pattern {
		s := penStep{length: v, down: true}
		switch {
		case v == 0:
			s.length = dotLength
		case v < 0:
			s.length, s.down = -v, false
		}
		steps = append(steps, s)
		cycle += s.length
	}
	return steps, cycle
}

// dashCursor tracks the pattern position while walking a polyline.
type dashCursor struct {
	steps []penStep
	i     int
	rem   float64
}

// walkEdge appends the pen-down sub-segments of one polyline edge, advancing the
// pattern cursor so the pattern continues seamlessly onto the next edge.
func (c *dashCursor) walkEdge(segs [][2]math.Point2, p, q math.Point2) [][2]math.Point2 {
	total := float64(p.DistanceTo(q))
	for at := 0.0; total-at > minCycle; {
		step := c.rem
		if left := total - at; step > left {
			step = left
		}
		if c.steps[c.i].down {
			segs = append(segs, [2]math.Point2{lerp2(p, q, at/total), lerp2(p, q, (at+step)/total)})
		}
		at += step
		c.advance(step)
	}
	return segs
}

// advance consumes pattern length, rolling to the next element when one runs out.
func (c *dashCursor) advance(used float64) {
	c.rem -= used
	if c.rem <= minCycle {
		c.i = (c.i + 1) % len(c.steps)
		c.rem = c.steps[c.i].length
	}
}

// lerp2 interpolates between two points at parameter t ∈ [0,1].
func lerp2(p, q math.Point2, t float64) math.Point2 {
	return math.P2(p.X+(q.X-p.X)*math.Scalar(t), p.Y+(q.Y-p.Y)*math.Scalar(t))
}
