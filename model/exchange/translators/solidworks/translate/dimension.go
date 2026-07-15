// SPDX-License-Identifier: GPL-2.0-only

package translate

import (
	"fmt"
	"math"

	m "oblikovati.org/math"
	"oblikovati.org/model/exchange/translators/solidworks/sldprt"
	"oblikovati.org/model/sketch"
)

// applyDimensions pins each decoded dimension onto the geometry it measures. SolidWorks stores a
// dimension's value but references its entities through the object graph (not yet resolved), so — as
// with constraints — the value is matched to a geometric measurement that already equals it: a
// circle's diameter or radius, or a line's length. The dimension is kept only if it removes a degree
// of freedom without moving a point (keptWithoutMoving), so a mis-matched value never edits geometry.
func applyDimensions(sk *sketch.Sketch, dims []sldprt.Dimension, lines []*sketch.Line, arcs []*sketch.Arc, circles []*sketch.Circle) {
	dc := sk.DimensionConstraints()
	cs := curvesOf(arcs, circles)
	for _, dm := range dims {
		applyOneDimension(sk, dc, dm.Value*metresToCm, lines, cs)
	}
}

// applyOneDimension binds a single value (cm) as the first dimension that measures it: a curve
// diameter, a curve radius, or a line length. Returns whether one was applied.
func applyOneDimension(sk *sketch.Sketch, dc *sketch.DimensionConstraints, v float64, lines []*sketch.Line, cs []curve) bool {
	expr := fmt.Sprintf("%g cm", v)
	for _, c := range cs {
		if math.Abs(2*c.r-v) < geomEps && keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) { return dc.AddDiameter(c.h, expr) }) {
			return true
		}
		if math.Abs(c.r-v) < geomEps && keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) { return dc.AddRadius(c.h, expr) }) {
			return true
		}
	}
	for _, l := range lines {
		length := math.Hypot(l.B.X-l.A.X, l.B.Y-l.A.Y)
		if math.Abs(length-v) < geomEps && keptWithoutMoving(sk, func() (*sketch.DimensionConstraint, error) { return dc.AddDistance(l.A, l.B, expr) }) {
			return true
		}
	}
	return false
}

// keptWithoutMoving applies a dimension and keeps it only if the sketch then has strictly fewer
// degrees of freedom AND no point moved after a solve — so a dimension is a pure DOF reduction, never
// a geometry edit (a value that admits a different solution would move points and is reverted).
// Mirrors the Inventor translator's guard.
func keptWithoutMoving(sk *sketch.Sketch, add func() (*sketch.DimensionConstraint, error)) bool {
	pts := sk.Points()
	snap := make([]m.Point2, pts.Count())
	for i := 0; i < pts.Count(); i++ {
		snap[i] = pts.Item(i).Position()
	}
	before := sk.DegreesOfFreedom()
	dim, err := add()
	if err != nil {
		return false
	}
	sk.Solve()
	if sk.DegreesOfFreedom() < before && !anyPointMoved(pts, snap) {
		return true
	}
	sk.DimensionConstraints().Delete(dim)
	for i := 0; i < pts.Count(); i++ {
		pts.Item(i).SetPosition(snap[i])
	}
	return false
}

// anyPointMoved reports whether any sketch point drifted from its snapshot beyond geomEps.
func anyPointMoved(pts *sketch.Points, snap []m.Point2) bool {
	for i := 0; i < pts.Count() && i < len(snap); i++ {
		p := pts.Item(i).Position()
		if math.Abs(p.X-snap[i].X) > geomEps || math.Abs(p.Y-snap[i].Y) > geomEps {
			return true
		}
	}
	return false
}
