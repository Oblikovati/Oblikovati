// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// CurveIsClosed reports whether a bounded curve's two domain ends land on the same point — a closed
// circuit whose parameter wraps, as opposed to an open arc that merely starts and stops.
//
// The question is answered by MEASUREMENT, not by curve kind. It used to live in kernel/brep as a
// type switch over geometry kinds, which the kernel rules forbid outside this package for exactly the
// failure it caused there: a curve kind the switch did not list (the exact ruled∩quadric section) was
// silently classed as open, so the boundary re-emission that anchors a shared closed loop to its
// curve's own domain never ran, the two operand walls of a boolean handed the welder the same loop
// opened at different points, and the solid failed to stitch (Oblikovati/Oblikovati#3489). A closed
// curve is closed because its ends meet, whatever type it is.
//
// The gauge is the stitch resolution of the curve's own sampled extent — a distance test against the
// curve's scale, so a genuinely closed loop cannot be misread as open by rounding (#1602). An
// unbounded curve cannot close.
//
//	closed := geom.CurveIsClosed(sectionArc) // true for a full oval, false for one branch of it
func CurveIsClosed(c Curve3) bool {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || lo == hi {
		return false
	}
	gap := float64(c.PointAt(lo).DistanceTo(c.PointAt(hi)))
	return gap <= ResolutionForPoints(curveGaugePoints(c, lo, hi)).Stitch()
}

// curveGaugePoints samples the curve just densely enough to gauge its spatial extent — the scale its
// closure gap is judged against. A polyline's own vertices already are that sample.
func curveGaugePoints(c Curve3, lo, hi float64) []math.Point3 {
	if pl, ok := c.(*Polyline); ok {
		return pl.Vertices
	}
	const gaugeSamples = 8
	pts := make([]math.Point3, 0, gaugeSamples+1)
	for i := 0; i <= gaugeSamples; i++ {
		pts = append(pts, c.PointAt(lo+(hi-lo)*float64(i)/gaugeSamples))
	}
	return pts
}
