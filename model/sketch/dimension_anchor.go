// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// Where a dimension's value text sits (#2017). The placement is stored RELATIVE to the dimension's
// own frame, not as a point in the sketch: a dimension annotates its geometry, so when that
// geometry moves, rotates or resizes the annotation has to travel with it. Stored absolutely, a
// dragged label stayed behind while its leader stretched to reach it — the dimension's offset from
// the geometry silently changed, and a line dragged far enough flipped the glyph to the other side.
//
// The frame is (origin, x, y) where origin is the anchor the dimension would use with no user
// placement at all, x runs along the dimension and y across it. A stored offset of (0,0) therefore
// means "the default position", and any offset keeps its meaning as the geometry changes.

const (
	// dimGapFactor offsets the dimension line off the geometry as a fraction of the measured size;
	// dimMinGap (database units, cm) keeps a tiny dimension legible.
	dimGapFactor = 0.15
	dimMinGap    = 0.5
)

// LabelAnchor returns where this dimension's value text belongs right now, derived from live
// geometry plus any user placement. ok is false for a dimension whose shape has no anchor yet
// (an unsupported kind, or refs that are not the shape the kind expects).
//
// Example: pt, ok := dim.LabelAnchor() — the point a renderer draws the value at.
func (d *DimensionConstraint) LabelAnchor() (math.Point2, bool) {
	origin, x, y, ok := d.labelFrame()
	if !ok {
		return math.Point2{}, false
	}
	if d.textOffset == nil {
		return origin, true
	}
	return origin.TranslateBy(x.Scale(d.textOffset.X).Add(y.Scale(d.textOffset.Y))), true
}

// TextPoint returns the dimension's annotation-text placement in sketch coordinates and whether
// the user has placed it (Inventor Point2d TextPoint, #1875). The point is recomputed from live
// geometry, so it is always where the text actually is — not where it was when it was dropped.
func (d *DimensionConstraint) TextPoint() (math.Point2, bool) {
	if d.textOffset == nil {
		return math.Point2{}, false
	}
	origin, x, y := d.placementFrame()
	return origin.TranslateBy(x.Scale(d.textOffset.X).Add(y.Scale(d.textOffset.Y))), true
}

// SetTextPoint places the annotation text at a sketch point, storing it relative to the dimension's
// frame so it tracks the geometry from then on (#1875, #2017).
func (d *DimensionConstraint) SetTextPoint(p math.Point2) {
	origin, x, y := d.placementFrame()
	v := origin.VectorTo(p)
	off := math.V2(v.Dot(x), v.Dot(y)) // x,y are orthonormal, so the dot products are the components
	d.textOffset = &off
}

// placementFrame is the frame a stored placement is expressed in. It is the dimension's own frame
// when it has one; otherwise the sketch axes, which makes the stored offset the plain sketch point.
// Kinds with no drawn anchor (offsetDim and the other M21 shapes) still carry a TextPoint through
// the API, persistence and copy — a dimension nothing renders yet must not silently lose the
// placement a caller set on it (#1875).
func (d *DimensionConstraint) placementFrame() (math.Point2, math.Vector2, math.Vector2) {
	if origin, x, y, ok := d.labelFrame(); ok {
		return origin, x, y
	}
	return math.P2(0, 0), math.V2(1, 0), math.V2(0, 1)
}

// ClearTextPoint returns the text to its derived position.
func (d *DimensionConstraint) ClearTextPoint() { d.textOffset = nil }

// labelFrame is the dimension's local frame: the anchor it would use unplaced, plus orthonormal
// axes along and across it. Every kind that draws returns one; anything else is false.
func (d *DimensionConstraint) labelFrame() (math.Point2, math.Vector2, math.Vector2, bool) {
	refs := d.Refs()
	switch d.Kind() {
	case DistanceDim:
		return distanceFrame(d, refs)
	case RadiusDim, DiameterDim:
		return circleFrame(refs)
	case AngleDim:
		return angleFrame(refs)
	case ArcLengthDim:
		return arcLengthFrame(refs)
	}
	return math.Point2{}, math.Vector2{}, math.Vector2{}, false
}

// DistanceLineDirection is the direction this distance dimension's line runs: along the measured
// segment when aligned, along X when horizontal, along Y when vertical (#2025). The label frame
// and the renderer share it, so the drawn dimension line and the text always agree.
//
//	dir := dim.DistanceLineDirection(a, b) // {1,0} for a horizontal dimension
func (d *DimensionConstraint) DistanceLineDirection(a, b math.Point2) math.Vector2 {
	switch d.orientation {
	case HorizontalDistance:
		return math.V2(1, 0)
	case VerticalDistance:
		return math.V2(0, 1)
	default:
		return unitOr(a.VectorTo(b), math.V2(1, 0))
	}
}

// distanceFrame anchors at the midpoint of the offset dimension line, with x along that line and
// y across it. Rotating the segment rotates an ALIGNED frame, so a label placed above a
// horizontal line stays above it once the line is vertical; a horizontal or vertical dimension
// keeps its axis instead, because that is the direction it measures along.
func distanceFrame(d *DimensionConstraint, refs []Entity) (math.Point2, math.Vector2, math.Vector2, bool) {
	a, b := refPoint(refs, 0), refPoint(refs, 1)
	if a == nil || b == nil {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, false
	}
	pa, pb := a.Position(), b.Position()
	dir := d.DistanceLineDirection(pa, pb)
	perp := math.V2(-dir.Y, dir.X)
	origin := pa.Midpoint(pb).TranslateBy(perp.Scale(dimGap(pa.DistanceTo(pb))))
	return origin, dir, perp, true
}

// circleFrame anchors on the default 45° leader just outside the rim, with x running outward. The
// origin moves with the centre AND the radius, so the label keeps its gap off the rim when the
// dimension drives the circle bigger.
func circleFrame(refs []Entity) (math.Point2, math.Vector2, math.Vector2, bool) {
	c, ok := refEntity[*Circle](refs, 0)
	if !ok {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, false
	}
	dir := math.V2(stdmath.Sqrt2/2, stdmath.Sqrt2/2)
	r := c.Radius
	origin := c.Center.Position().TranslateBy(dir.Scale(r + dimGap(r)))
	return origin, dir, math.V2(-dir.Y, dir.X), true
}

// angleFrame anchors on the arc midpoint between the two lines, with x running out from the vertex.
func angleFrame(refs []Entity) (math.Point2, math.Vector2, math.Vector2, bool) {
	l1, ok1 := refEntity[*Line](refs, 0)
	l2, ok2 := refEntity[*Line](refs, 1)
	if !ok1 || !ok2 {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, false
	}
	v, ok := infiniteLineIntersection(l1, l2)
	if !ok {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, false
	}
	startA := angleFromVertex(v, l1)
	midA := startA + shortestDelta(startA, angleFromVertex(v, l2))/2
	r := dimGap(stdmath.Min(v.DistanceTo(farEnd(v, l1)), v.DistanceTo(farEnd(v, l2))))
	dir := angleVec(midA)
	return v.TranslateBy(dir.Scale(r)), dir, math.V2(-dir.Y, dir.X), true
}

// arcLengthFrame anchors just off the arc's midpoint, with x running outward from the centre.
func arcLengthFrame(refs []Entity) (math.Point2, math.Vector2, math.Vector2, bool) {
	a, ok := refEntity[*Arc](refs, 0)
	if !ok {
		return math.Point2{}, math.Vector2{}, math.Vector2{}, false
	}
	c, r := a.Center.Position(), a.Radius()
	dir := angleVec(arcMidpointAngle(a))
	return c.TranslateBy(dir.Scale(r + dimGap(r))), dir, math.V2(-dir.Y, dir.X), true
}

// dimGap is the dimension line's offset for a measured size (a fraction of it, floored).
func dimGap(size math.Scalar) math.Scalar {
	if g := size * dimGapFactor; g > dimMinGap {
		return g
	}
	return dimMinGap
}

// unitOr normalises v, falling back to def for a (near-)zero vector.
func unitOr(v math.Vector2, def math.Vector2) math.Vector2 {
	if u, ok := unitVec(v); ok {
		return u
	}
	return def
}

// refPoint / refEntity fetch a typed ref, defensively against unexpected ref shapes.
func refPoint(refs []Entity, i int) *Point {
	p, _ := refEntity[*Point](refs, i)
	return p
}

func refEntity[T Entity](refs []Entity, i int) (T, bool) {
	var zero T
	if i >= len(refs) {
		return zero, false
	}
	e, ok := refs[i].(T)
	return e, ok
}

// angleVec is the unit vector at angle a (radians).
func angleVec(a float64) math.Vector2 { return math.V2(stdmath.Cos(a), stdmath.Sin(a)) }

// angleFromVertex is the angle from the vertex toward the line's far endpoint, so an angle arc
// opens between the two lines rather than away from them.
func angleFromVertex(vertex math.Point2, l *Line) float64 {
	d := vertex.VectorTo(farEnd(vertex, l))
	return stdmath.Atan2(d.Y, d.X)
}

// farEnd returns the line endpoint farther from the vertex.
func farEnd(vertex math.Point2, l *Line) math.Point2 {
	a, b := l.A.Position(), l.B.Position()
	if vertex.DistanceTo(a) > vertex.DistanceTo(b) {
		return a
	}
	return b
}

// infiniteLineIntersection returns where the two infinite lines cross, false if parallel.
func infiniteLineIntersection(l1, l2 *Line) (math.Point2, bool) {
	p, r := l1.A.Position(), l1.A.Position().VectorTo(l1.B.Position())
	q, s := l2.A.Position(), l2.A.Position().VectorTo(l2.B.Position())
	rxs := r.Cross(s)
	if stdmath.Abs(rxs) < math.DefaultTolerance {
		return math.Point2{}, false
	}
	return p.TranslateBy(r.Scale(p.VectorTo(q).Cross(s) / rxs)), true
}

// shortestDelta is the signed angle in (-π,π] from a1 to a2.
func shortestDelta(a1, a2 float64) float64 {
	d := stdmath.Mod(a2-a1, 2*stdmath.Pi)
	if d <= -stdmath.Pi {
		d += 2 * stdmath.Pi
	}
	if d > stdmath.Pi {
		d -= 2 * stdmath.Pi
	}
	return d
}

// arcMidpointAngle is the angle about the centre of the arc's midpoint, respecting sweep direction.
func arcMidpointAngle(a *Arc) float64 {
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
