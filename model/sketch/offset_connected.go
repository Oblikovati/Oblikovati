// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// OffsetConnectedLoop offsets a connected chain of sketch curves (path) by signed distance d along
// the LEFT normal of the traversal direction (d>0 offsets to the left of travel), keeping the curves
// analytic — a line stays a line, an arc a concentric arc, a circle a concentric circle — and joining
// adjacent offset curves at their intersection so corners meet (miter/extend/trim), exactly as
// Inventor's sketch Offset does. It returns the created offset entities. A closed path wraps the last
// corner to the first; an open path leaves the two ends untrimmed.
//
// For a loop traversed counter-clockwise the left normal points inward, so a positive d SHRINKS it
// (an inner offset); a negative d grows it.
//
//	ents, err := sk.OffsetConnectedLoop(path, 0.25)
func (s *Sketch) OffsetConnectedLoop(path *Path, d float64) ([]Entity, error) {
	ents := path.Entities()
	if len(ents) == 0 {
		return nil, fmt.Errorf("offset: path has no entities (want at least one line/arc/circle)")
	}
	if circ, ok, err := s.offsetSingleCircle(ents, d); ok {
		return circ, err
	}
	elems, err := s.buildOffsetElements(ents, d)
	if err != nil {
		return nil, err
	}
	starts, ends := offsetCornerEndpoints(elems, path.IsClosed())
	return s.buildOffsetEntities(elems, starts, ends), nil
}

// offsetElement is one entity's analytic offset primitive plus the data to join it to its neighbours.
// A line keeps a point on the offset line (start), a unit travel direction (dir) and its unit left
// normal; an arc keeps its centre, offset radius and effective-CCW. start/end are the offset
// primitive's own endpoints in traversal order (used for an open path's untrimmed ends); origStart/
// origEnd are the original shared corner points (used to pick the right intersection branch).
type offsetElement struct {
	isLine     bool
	dir        math.Vector2 // line: unit travel direction
	normal     math.Vector2 // line: unit left normal
	center     math.Point2  // arc: centre
	radius     float64      // arc: offset radius r'
	effCCW     bool         // arc: effective CCW (arc.CCW XOR reversed)
	dist       float64      // signed offset distance (for the parallel-neighbour fallback)
	start, end math.Point2  // offset primitive's own endpoints, in traversal order
	origStart  math.Point2  // original traversal-start corner
	origEnd    math.Point2  // original traversal-end corner
}

// analyticSeg is an entity reduced to its analytic form in the entity's NATURAL direction: a line
// p0→p1, or an arc of the given centre/radius/CCW with natural endpoints p0 (start) and p1 (end).
type analyticSeg struct {
	isLine bool
	p0, p1 math.Point2
	center math.Point2
	radius float64
	ccw    bool
}

// offsetSingleCircle handles a single-entity full-circle loop: a concentric circle of the offset
// radius, with no corners. ok is false when the path is not one closed circle. A circle traverses
// CCW naturally (reversed flips it), so the left normal points inward when effective-CCW.
func (s *Sketch) offsetSingleCircle(ents []ProfileEntity, d float64) ([]Entity, bool, error) {
	if len(ents) != 1 {
		return nil, false, nil
	}
	center, r, ok := circleCenterRadius(ents[0].Entity)
	if !ok {
		return nil, false, nil
	}
	rp := offsetRadius(r, d, !ents[0].Reversed())
	if rp <= 0 {
		return nil, true, fmt.Errorf("offset: circle radius %.4g offset by %.4g collapses to %.4g", r, d, rp)
	}
	return []Entity{s.circles.AddByCenterRadius(center, math.Scalar(rp))}, true, nil
}

// buildOffsetElements reduces every path entity to its offset primitive in traversal order.
func (s *Sketch) buildOffsetElements(ents []ProfileEntity, d float64) ([]offsetElement, error) {
	elems := make([]offsetElement, len(ents))
	for i, pe := range ents {
		el, err := offsetElementOf(pe, d)
		if err != nil {
			return nil, err
		}
		elems[i] = el
	}
	return elems, nil
}

// offsetElementOf builds the offset primitive for one profile entity, honoring its traversal
// direction (a reversed entity is walked from its natural end to start).
func offsetElementOf(pe ProfileEntity, d float64) (offsetElement, error) {
	seg, err := analyticSegOf(pe.Entity)
	if err != nil {
		return offsetElement{}, err
	}
	a, b := seg.p0, seg.p1
	if pe.Reversed() {
		a, b = b, a
	}
	if seg.isLine {
		return lineOffsetElement(a, b, d)
	}
	return arcOffsetElement(seg, a, b, pe.Reversed(), d)
}

// lineOffsetElement shifts the traversal-ordered line a→b by d along its left normal.
func lineOffsetElement(a, b math.Point2, d float64) (offsetElement, error) {
	u, ok := unitVec(a.VectorTo(b))
	if !ok {
		return offsetElement{}, fmt.Errorf("offset: degenerate zero-length line at %v", a)
	}
	n := math.V2(-u.Y, u.X) // left normal of travel (unit, since u is unit)
	shift := n.Scale(d)
	return offsetElement{
		isLine: true, dir: u, normal: n, dist: d,
		start: a.TranslateBy(shift), end: b.TranslateBy(shift),
		origStart: a, origEnd: b,
	}, nil
}

// arcOffsetElement builds the concentric offset arc for the traversal-ordered arc endpoints a→b.
// Effective-CCW = arc.CCW XOR reversed; the left normal points inward (toward the centre) when
// effective-CCW, so the offset radius is r-d there and r+d otherwise.
func arcOffsetElement(seg analyticSeg, a, b math.Point2, reversed bool, d float64) (offsetElement, error) {
	effCCW := seg.ccw != reversed
	rp := offsetRadius(seg.radius, d, effCCW)
	if rp <= 0 {
		return offsetElement{}, fmt.Errorf("offset: arc radius %.4g offset by %.4g collapses to %.4g", seg.radius, d, rp)
	}
	scale := rp / seg.radius
	return offsetElement{
		center: seg.center, radius: rp, effCCW: effCCW, dist: d,
		start: radialScale(seg.center, a, scale), end: radialScale(seg.center, b, scale),
		origStart: a, origEnd: b,
	}, nil
}

// offsetRadius is the concentric offset radius: r-d when the left normal points inward (effective-
// CCW), r+d otherwise.
func offsetRadius(r, d float64, effCCW bool) float64 {
	if effCCW {
		return r - d
	}
	return r + d
}

// radialScale returns center + (p-center)·scale — a point moved along its radius from center.
func radialScale(center, p math.Point2, scale float64) math.Point2 {
	return center.TranslateBy(center.VectorTo(p).Scale(scale))
}

// analyticSegOf reduces a supported entity to a line or circular-arc form; it errors for a curve with
// no analytic form (a spline, or an obliquely projected arc that fits as an ellipse).
func analyticSegOf(e Entity) (analyticSeg, error) {
	switch v := e.(type) {
	case *Line:
		return analyticSeg{isLine: true, p0: v.A.Position(), p1: v.B.Position()}, nil
	case *Arc:
		return analyticSeg{
			center: v.Center.Position(), radius: float64(v.Radius()), ccw: v.CounterClockwise,
			p0: v.Start.Position(), p1: v.End.Position(),
		}, nil
	case *ProjectedCurve:
		return projectedAnalyticSeg(v)
	default:
		return analyticSeg{}, fmt.Errorf("offset: curve is not analytic (%T)", e)
	}
}

// projectedAnalyticSeg reduces a projected curve to its analytic line/arc form (Inventor keeps a
// projected edge analytic). A full projected circle cannot join a multi-entity chain, and a non-
// analytic (shapeNone) projection has no analytic offset, so both error.
func projectedAnalyticSeg(pc *ProjectedCurve) (analyticSeg, error) {
	sh := pc.shape
	switch sh.kind {
	case shapeLine:
		return analyticSeg{isLine: true, p0: sh.a, p1: sh.b}, nil
	case shapeArc:
		return analyticSeg{
			center: sh.center, radius: sh.radius, ccw: sh.sweep > 0,
			p0: arcPointAt(sh.center, sh.radius, sh.start),
			p1: arcPointAt(sh.center, sh.radius, sh.start+sh.sweep),
		}, nil
	default:
		return analyticSeg{}, fmt.Errorf("offset: curve is not analytic (projected shape kind %d)", sh.kind)
	}
}

// circleCenterRadius returns the centre and radius of a full-circle entity (a Circle or a projected
// circle), reporting false for anything else.
func circleCenterRadius(e Entity) (math.Point2, float64, bool) {
	switch v := e.(type) {
	case *Circle:
		return v.Center.Position(), float64(v.Radius), true
	case *ProjectedCurve:
		if v.shape.kind == shapeCircle {
			return v.shape.center, v.shape.radius, true
		}
	}
	return math.Point2{}, 0, false
}

// offsetCornerEndpoints computes each offset element's trimmed start/end. Interior corners (and, for a
// closed path, the wrap corner from the last element to the first) are the intersection of adjacent
// offset primitives; an open path keeps each end element's own untrimmed offset endpoint.
func offsetCornerEndpoints(elems []offsetElement, closed bool) (starts, ends []math.Point2) {
	n := len(elems)
	starts = make([]math.Point2, n)
	ends = make([]math.Point2, n)
	for i, el := range elems {
		starts[i], ends[i] = el.start, el.end
	}
	for i := 0; i+1 < n; i++ {
		c := cornerPoint(elems[i], elems[i+1])
		ends[i], starts[i+1] = c, c
	}
	if closed && n >= 2 {
		c := cornerPoint(elems[n-1], elems[0])
		ends[n-1], starts[0] = c, c
	}
	return starts, ends
}

// cornerPoint is the new shared endpoint where two adjacent offset primitives meet: their
// intersection nearest the original shared corner, or a normal-averaged fallback when they do not
// cross (parallel lines, a missed circle) so a corner is still produced.
func cornerPoint(a, b offsetElement) math.Point2 {
	corner := a.origEnd // == b.origStart, the shared endpoint of the two originals
	if p, ok := intersectElements(a, b, corner); ok {
		return p
	}
	return fallbackCorner(a, b, corner)
}

// intersectElements intersects two offset primitives, picking the branch nearest the original corner.
func intersectElements(a, b offsetElement, near math.Point2) (math.Point2, bool) {
	switch {
	case a.isLine && b.isLine:
		return lineLineCorner(a, b)
	case a.isLine:
		return lineArcCorner(a, b, near)
	case b.isLine:
		return lineArcCorner(b, a, near)
	default:
		return arcArcCorner(a, b, near)
	}
}

// lineLineCorner intersects two infinite offset lines (false when parallel).
func lineLineCorner(a, b offsetElement) (math.Point2, bool) {
	la, err := geom.NewLine2d(a.start, a.dir)
	if err != nil {
		return math.Point2{}, false
	}
	lb, err := geom.NewLine2d(b.start, b.dir)
	if err != nil {
		return math.Point2{}, false
	}
	return geom.Line2dIntersection(la, lb, 0)
}

// lineArcCorner intersects an offset line with an offset arc's circle, nearest the original corner.
func lineArcCorner(line, arc offsetElement, near math.Point2) (math.Point2, bool) {
	l, err := geom.NewLine2d(line.start, line.dir)
	if err != nil {
		return math.Point2{}, false
	}
	c := geom.NewCircle2d(arc.center, arc.radius)
	return nearestOf(geom.LineCircle2dIntersection(l, c, 0), near)
}

// arcArcCorner intersects two offset arcs' circles, nearest the original corner.
func arcArcCorner(a, b offsetElement, near math.Point2) (math.Point2, bool) {
	c1 := geom.NewCircle2d(a.center, a.radius)
	c2 := geom.NewCircle2d(b.center, b.radius)
	return nearestOf(geom.Circle2dCircle2dIntersection(c1, c2, 0), near)
}

// nearestOf returns the candidate closest to near (false for an empty set).
func nearestOf(cands []math.Point2, near math.Point2) (math.Point2, bool) {
	if len(cands) == 0 {
		return math.Point2{}, false
	}
	best, bestD := cands[0], near.DistanceTo(cands[0])
	for _, p := range cands[1:] {
		if d := near.DistanceTo(p); d < bestD {
			best, bestD = p, d
		}
	}
	return best, true
}

// fallbackCorner offsets the original corner along the average of the two left normals, so parallel
// or non-crossing offset primitives (a collinear line continuation, a tangent that just misses) still
// produce a joined corner rather than a gap.
func fallbackCorner(a, b offsetElement, corner math.Point2) math.Point2 {
	avg := a.leftNormalAt(corner).Add(b.leftNormalAt(corner))
	u, ok := unitVec(avg)
	if !ok {
		u = a.leftNormalAt(corner) // opposing normals cancel: fall back to one side
	}
	return corner.TranslateBy(u.Scale(a.dist))
}

// leftNormalAt is the unit left normal of the element's travel at point p: constant for a line; the
// radial direction toward the centre (effective-CCW) or away from it for an arc.
func (el offsetElement) leftNormalAt(p math.Point2) math.Vector2 {
	if el.isLine {
		return el.normal
	}
	u, ok := unitVec(el.center.VectorTo(p))
	if !ok {
		return math.Vector2{}
	}
	if el.effCCW {
		return u.Negate() // inward
	}
	return u
}

// buildOffsetEntities creates the trimmed offset geometry: a line for a line element, a concentric arc
// for an arc element, joined at the computed corner endpoints.
func (s *Sketch) buildOffsetEntities(elems []offsetElement, starts, ends []math.Point2) []Entity {
	out := make([]Entity, 0, len(elems))
	for i, el := range elems {
		if el.isLine {
			out = append(out, s.lines.AddByTwoPoints(starts[i], ends[i]))
			continue
		}
		out = append(out, s.arcs.AddByCenterStartEnd(el.center, starts[i], ends[i], el.effCCW))
	}
	return out
}
