// SPDX-License-Identifier: GPL-2.0-only

package linetype

import "oblikovati.org/math"

// dotLength is the pen-down length a pattern dot (0) renders as — long enough to
// survive rasterization at typical sketch zoom, short enough to read as a dot.
const dotLength = 0.02 // cm

// minCycle guards against degenerate patterns whose elements sum to ~zero, which
// would otherwise loop forever along an edge.
const minCycle = 1e-6

// dashable is a point a pattern can be walked along: planar or in model space. The dashing is
// pure arc-length arithmetic, so the 2D and 3D forms differ only in the point type (#2039).
type dashable[P any] interface {
	DistanceTo(P) math.Scalar
	Lerp(P, math.Scalar) P
}

// DashPolyline splits a polyline into the pen-down segments of a .lin pattern,
// flowing the pattern continuously across vertices. A nil/degenerate pattern
// returns nil — the caller draws the polyline solid.
//
//	segs := linetype.DashPolyline(pts, false, linetype.Builtin(types.SketchLineCenter))
func DashPolyline(pts []math.Point2, closed bool, pattern []float64) [][2]math.Point2 {
	return dashPolyline(pts, closed, pattern)
}

// DashPolyline3D is DashPolyline for a model-space polyline — what the 3D-sketch overlay dashes
// a construction curve or a line-type override with.
//
//	segs := linetype.DashPolyline3D(sketch.SamplePolyline3D(e, 64), false, pattern)
func DashPolyline3D(pts []math.Point3, closed bool, pattern []float64) [][2]math.Point3 {
	return dashPolyline(pts, closed, pattern)
}

// dashPolyline walks the pattern along a polyline of either dimension.
func dashPolyline[P dashable[P]](pts []P, closed bool, pattern []float64) [][2]P {
	steps, cycle := penSteps(pattern)
	if len(pts) < 2 || cycle < minCycle {
		return nil
	}
	if closed {
		pts = append(append([]P{}, pts...), pts[0])
	}
	segs := [][2]P{}
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
// cursor's own pattern position so the pattern continues seamlessly onto the next
// edge. A generic method (Go 1.27) rather than a free function taking *dashCursor:
// the operation belongs to the cursor's state, parameterized per call by point type.
func (c *dashCursor) walkEdge[P dashable[P]](segs [][2]P, p, q P) [][2]P {
	total := float64(p.DistanceTo(q))
	for at := 0.0; total-at > minCycle; {
		step := c.rem
		if left := total - at; step > left {
			step = left
		}
		if c.steps[c.i].down {
			segs = append(segs, [2]P{
				p.Lerp(q, math.Scalar(at/total)), p.Lerp(q, math.Scalar((at+step)/total)),
			})
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
