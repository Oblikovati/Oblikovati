// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"

	"oblikovati.org/math"
	"oblikovati.org/model/param"
	"oblikovati.org/model/sketch"
)

// SketchDimensions turns the active sketch's dimensional constraints into render-ready
// views — line segments and a value label, all in sketch-plane (2D) coordinates so the
// head maps them to model space through the sketch plane and projects the label anchor
// to the screen. Returns nil when no sketch is being edited. This keeps all dimension
// geometry/formatting logic here (unit-testable, headless) and the head a dumb renderer.
func (s *Session) SketchDimensions() []DimensionView {
	if s.activeSketch == nil {
		return nil
	}
	units := s.DocumentUnits()
	var out []DimensionView
	for _, d := range s.activeSketch.DimensionConstraints().All() {
		if v, ok := dimensionView(d, units); ok {
			out = append(out, v)
		}
	}
	return out
}

// DimensionView is one dimension ready to draw: its lines (sketch-plane point pairs),
// the value label and the plane-space point to anchor the label text at, plus the
// dimension itself so the head can re-open it for editing on a double-click.
type DimensionView struct {
	Dim      *sketch.DimensionConstraint
	Segments [][2]math.Point2
	Label    string
	LabelAt  math.Point2
	Driven   bool
}

const (
	// radiusPrefix/diameterPrefix tag radial dimensions in the (ASCII default font)
	// label; the leader geometry (radial vs across-circle) also distinguishes them.
	radiusPrefix   = "R"
	diameterPrefix = "D"
	// dimGapFactor offsets the dimension line off the geometry as a fraction of the
	// measured size; dimMinGap (db units, cm) keeps tiny dimensions legible.
	dimGapFactor = 0.15
	dimMinGap    = 0.5
	// angleArcSegments samples the angle dimension's arc.
	angleArcSegments = 16
)

// dimensionView builds the view for one dimension, or false for an unhandled shape.
func dimensionView(d *sketch.DimensionConstraint, units param.UnitsOfMeasure) (DimensionView, bool) {
	refs := d.Refs()
	switch d.Kind() {
	case sketch.DistanceDim:
		a, b := asPoint(refs, 0), asPoint(refs, 1)
		if a == nil || b == nil {
			return DimensionView{}, false
		}
		return distanceView(d, a.Position(), b.Position(), lengthLabel(units, d.Measured())), true
	case sketch.RadiusDim:
		return circleView(d, refs, radiusPrefix+lengthLabel(units, d.Measured()), false)
	case sketch.DiameterDim:
		return circleView(d, refs, diameterPrefix+lengthLabel(units, d.Measured()), true)
	case sketch.AngleDim:
		return angleView(d, refs, angleLabel(units, d.Measured()))
	case sketch.ArcLengthDim:
		return arcLengthView(d, refs, lengthLabel(units, d.Measured()))
	}
	return DimensionView{}, false
}

// lengthLabel / angleLabel render a database-unit measurement for a dimension's on-screen label
// at the document's display precision and FORMAT (decimal/fractional/architectural, decimal-deg/
// DMS). Unlike the seed expression (lengthExpr), the label is recomputed from live geometry each
// frame, so it must format down from the raw measured float (e.g. 9.999999998 cm → "10.00 mm")
// rather than print it losslessly.
func lengthLabel(units param.UnitsOfMeasure, dbValue float64) string {
	return units.FormatDisplay(param.Quantity{Value: dbValue, Unit: param.Length})
}

func angleLabel(units param.UnitsOfMeasure, radians float64) string {
	return units.FormatDisplay(param.Quantity{Value: radians, Unit: param.Angle})
}

// distanceView offsets the dimension line off the measured segment by a perpendicular
// gap, with a witness line back to each measured point, and labels it at the midpoint.
func distanceView(d *sketch.DimensionConstraint, a, b math.Point2, label string) DimensionView {
	dir := normalize(a.VectorTo(b))
	off := math.V2(-dir.Y, dir.X).Scale(dimGap(a.DistanceTo(b)))
	a2, b2 := a.TranslateBy(off), b.TranslateBy(off)
	segs := [][2]math.Point2{{a, a2}, {b, b2}, {a2, b2}}
	return DimensionView{Dim: d, Segments: segs, Label: label, LabelAt: a2.Midpoint(b2), Driven: d.Driven()}
}

// circleView draws a radial leader (radius) or an across-the-circle line (diameter),
// extended past the rim by the gap, with the label at the outer end.
func circleView(d *sketch.DimensionConstraint, refs []sketch.Entity, label string, diameter bool) (DimensionView, bool) {
	c, ok := refs[0].(*sketch.Circle)
	if !ok {
		return DimensionView{}, false
	}
	center, r := c.Center.Position(), c.Radius
	dir := math.V2(stdmath.Sqrt2/2, stdmath.Sqrt2/2) // 45° leader
	out := center.TranslateBy(dir.Scale(r + dimGap(r)))
	near := center
	if diameter {
		near = center.TranslateBy(dir.Scale(-r))
	}
	segs := [][2]math.Point2{{near, out}}
	return DimensionView{Dim: d, Segments: segs, Label: label, LabelAt: out, Driven: d.Driven()}, true
}

// angleView draws an arc between the two lines about their intersection, labeled at the
// arc midpoint. False when the lines are parallel (no vertex).
func angleView(d *sketch.DimensionConstraint, refs []sketch.Entity, label string) (DimensionView, bool) {
	l1, l2 := asLine(refs, 0), asLine(refs, 1)
	if l1 == nil || l2 == nil {
		return DimensionView{}, false
	}
	v, ok := lineIntersection(l1, l2)
	if !ok {
		return DimensionView{}, false
	}
	startA := angleAwayFrom(v, l1)
	endA := startA + shortDelta(startA, angleAwayFrom(v, l2))
	// Size the arc by the shorter line's far-end span (its near end may sit on the
	// vertex, which would otherwise collapse the radius to the minimum gap).
	r := dimGap(stdmath.Min(v.DistanceTo(farthestEnd(v, l1)), v.DistanceTo(farthestEnd(v, l2))))
	segs := arcSegments(v, r, startA, endA)
	labelAt := v.TranslateBy(dirVec((startA + endA) / 2).Scale(r))
	return DimensionView{Dim: d, Segments: segs, Label: label, LabelAt: labelAt, Driven: d.Driven()}, true
}

// arcLengthView puts a short leader at the arc's midpoint, labeled with the length.
func arcLengthView(d *sketch.DimensionConstraint, refs []sketch.Entity, label string) (DimensionView, bool) {
	a, ok := refs[0].(*sketch.Arc)
	if !ok {
		return DimensionView{}, false
	}
	c, r := a.Center.Position(), a.Radius()
	mid := arcMidAngle(a)
	on := c.TranslateBy(dirVec(mid).Scale(r))
	out := c.TranslateBy(dirVec(mid).Scale(r + dimGap(r)))
	return DimensionView{Dim: d, Segments: [][2]math.Point2{{on, out}}, Label: label, LabelAt: out, Driven: d.Driven()}, true
}

// asPoint/asLine fetch a typed ref or nil (defensive against unexpected ref shapes).
func asPoint(refs []sketch.Entity, i int) *sketch.Point {
	if i >= len(refs) {
		return nil
	}
	p, _ := refs[i].(*sketch.Point)
	return p
}

func asLine(refs []sketch.Entity, i int) *sketch.Line {
	if i >= len(refs) {
		return nil
	}
	l, _ := refs[i].(*sketch.Line)
	return l
}

// normalize returns the unit vector of v, or +X for a (near-)zero vector.
func normalize(v math.Vector2) math.Vector2 {
	l := v.Length()
	if l < math.DefaultTolerance {
		return math.V2(1, 0)
	}
	return v.Scale(1 / l)
}

// dimGap is the dimension-line offset for a measured size (a fraction, floored).
func dimGap(size math.Scalar) math.Scalar {
	if g := size * dimGapFactor; g > dimMinGap {
		return g
	}
	return dimMinGap
}

// dirVec is the unit vector at angle a (radians).
func dirVec(a float64) math.Vector2 { return math.V2(stdmath.Cos(a), stdmath.Sin(a)) }

// angleAwayFrom is the angle of the direction from the vertex toward the line's far
// endpoint — so the angle arc opens between the two lines, not away from them.
func angleAwayFrom(vertex math.Point2, l *sketch.Line) float64 {
	d := vertex.VectorTo(farthestEnd(vertex, l))
	return stdmath.Atan2(d.Y, d.X)
}

// farthestEnd returns the line endpoint farther from the vertex.
func farthestEnd(vertex math.Point2, l *sketch.Line) math.Point2 {
	a, b := l.A.Position(), l.B.Position()
	if vertex.DistanceTo(a) > vertex.DistanceTo(b) {
		return a
	}
	return b
}

// lineIntersection returns the intersection of the two infinite lines, false if parallel.
func lineIntersection(l1, l2 *sketch.Line) (math.Point2, bool) {
	p, r := l1.A.Position(), l1.A.Position().VectorTo(l1.B.Position())
	q, s := l2.A.Position(), l2.A.Position().VectorTo(l2.B.Position())
	rxs := r.Cross(s)
	if stdmath.Abs(rxs) < math.DefaultTolerance {
		return math.Point2{}, false
	}
	t := p.VectorTo(q).Cross(s) / rxs
	return p.TranslateBy(r.Scale(t)), true
}

// shortDelta returns the signed angle (in (-π,π]) to rotate from a1 to a2.
func shortDelta(a1, a2 float64) float64 {
	d := stdmath.Mod(a2-a1, 2*stdmath.Pi)
	if d <= -stdmath.Pi {
		d += 2 * stdmath.Pi
	}
	if d > stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	return d
}

// arcSegments samples an arc from a0 to a1 about center at radius r into line segments.
func arcSegments(center math.Point2, r, a0, a1 float64) [][2]math.Point2 {
	prev := center.TranslateBy(dirVec(a0).Scale(r))
	segs := make([][2]math.Point2, 0, angleArcSegments)
	for i := 1; i <= angleArcSegments; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(angleArcSegments)
		cur := center.TranslateBy(dirVec(a).Scale(r))
		segs = append(segs, [2]math.Point2{prev, cur})
		prev = cur
	}
	return segs
}

// arcMidAngle returns the angle (about the center) of the arc's midpoint, respecting
// sweep direction.
func arcMidAngle(a *sketch.Arc) float64 {
	c := a.Center.Position()
	sa := stdmath.Atan2(a.Start.Position().Y-c.Y, a.Start.Position().X-c.X)
	ea := stdmath.Atan2(a.End.Position().Y-c.Y, a.End.Position().X-c.X)
	if a.CounterClockwise && ea < sa {
		ea += 2 * stdmath.Pi
	}
	if !a.CounterClockwise && ea > sa {
		ea -= 2 * stdmath.Pi
	}
	return (sa + ea) / 2
}
