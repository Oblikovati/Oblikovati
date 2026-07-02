// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Corner joining and 3D rebuild of the planar wire offset (wire_offset.go).
// At each corner the offsets either CROSS (the chain turns toward the offset
// side → trim both at their support intersection) or GAP (turns away → close
// per the WireOffsetCorner mode).

// joinOffsetCorners walks consecutive seg pairs (wrapping when closed),
// trimming crossings and inserting gap closures.
func joinOffsetCorners(src, offs []wireSeg, d, tol float64, corner WireOffsetCorner, closed bool) ([]wireSeg, error) {
	inserted := make([][]wireSeg, len(offs)) // closures appended after seg i
	pairs := len(offs) - 1
	if closed {
		pairs = len(offs)
	}
	for k := 0; k < pairs; k++ {
		i, j := k, (k+1)%len(offs)
		fill, err := joinPair(&src[i], &offs[i], &offs[j], d, tol, corner)
		if err != nil {
			return nil, err
		}
		inserted[i] = fill
	}
	var out []wireSeg
	for i := range offs {
		if segLength(&offs[i]) > tol {
			out = append(out, offs[i])
		}
		out = append(out, inserted[i]...)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("ops.OffsetPlanarWire: offset %g consumed the whole wire", d)
	}
	return out, nil
}

// joinPair resolves one corner between prev and next: weld if snug, trim if
// the offsets cross, close the gap otherwise. The original corner point comes
// from the source chain (prev's end).
func joinPair(srcPrev, prev, next *wireSeg, d, tol float64, corner WireOffsetCorner) ([]wireSeg, error) {
	if prev == next {
		return nil, nil // single-seg closed chain (a full circle) — already joined
	}
	if float64(prev.b.DistanceTo(next.a)) <= tol {
		setSegStart(next, prev.b)
		return nil, nil
	}
	turn := float64(srcPrev.endTangent().Cross(next.startTangent()))
	if d*turn > 0 {
		if trimAtSupportIntersection(prev, next) {
			return nil, nil
		}
		return []wireSeg{bevel(prev.b, next.a)}, nil // numerically parallel: bevel
	}
	return closeGap(srcPrev.b, prev, next, d, tol, corner)
}

// closeGap inserts the closure for a gap corner about original corner P.
func closeGap(p math.Point2, prev, next *wireSeg, d, tol float64, corner WireOffsetCorner) ([]wireSeg, error) {
	switch corner {
	case WireCornerCircular:
		return []wireSeg{closureArc(p, prev.b, next.a, d)}, nil
	case WireCornerExtend:
		if trimAtSupportIntersection(prev, next) {
			return nil, nil
		}
		return closeLinear(prev, next, tol)
	default:
		return closeLinear(prev, next, tol)
	}
}

// closeLinear extends both sides tangentially to their intersection (a miter);
// near-parallel tangents bevel straight across.
func closeLinear(prev, next *wireSeg, tol float64) ([]wireSeg, error) {
	m, ok := rayIntersection(prev.b, prev.endTangent(), next.a, next.startTangent().Negate())
	if !ok {
		return []wireSeg{bevel(prev.b, next.a)}, nil
	}
	out := make([]wireSeg, 0, 2)
	if float64(prev.b.DistanceTo(m)) > tol {
		out = append(out, bevel(prev.b, m))
	}
	if float64(m.DistanceTo(next.a)) > tol {
		out = append(out, bevel(m, next.a))
	}
	return out, nil
}

// closureArc rounds the gap with an arc about p of radius |d|, swept the short
// way from E to S.
func closureArc(p, e, s math.Point2, d float64) wireSeg {
	a0 := stdmath.Atan2(float64(e.Y-p.Y), float64(e.X-p.X))
	a1 := stdmath.Atan2(float64(s.Y-p.Y), float64(s.X-p.X))
	sweep := stdmath.Mod(a1-a0, 2*stdmath.Pi)
	if sweep > stdmath.Pi {
		sweep -= 2 * stdmath.Pi
	}
	if sweep < -stdmath.Pi {
		sweep += 2 * stdmath.Pi
	}
	arc := wireSeg{kind: wsArc, center: p, r: stdmath.Abs(d), a0: a0, sweep: sweep}
	arc.syncArcEnds()
	return arc
}

func bevel(a, b math.Point2) wireSeg { return wireSeg{kind: wsLine, a: a, b: b} }

// rayIntersection intersects two rays (origin + direction); ok is false when
// near-parallel.
func rayIntersection(p1 math.Point2, d1 math.Vector2, p2 math.Point2, d2 math.Vector2) (math.Point2, bool) {
	denom := float64(d1.Cross(d2))
	if stdmath.Abs(denom) < 1e-12 { // tol:numeric (unit-tangent cross determinant, dimensionless)
		return math.Point2{}, false
	}
	w := p1.VectorTo(p2)
	t := float64(w.Cross(d2)) / denom
	return p1.TranslateBy(d1.Scale(math.Scalar(t))), true
}

// trimAtSupportIntersection moves prev's end and next's start onto the
// intersection of their support curves nearest the current joint; false when
// the supports do not intersect.
func trimAtSupportIntersection(prev, next *wireSeg) bool {
	cands := supportIntersections(prev, next)
	if len(cands) == 0 {
		return false
	}
	best, bestCost := cands[0], stdmath.Inf(1)
	for _, c := range cands {
		cost := float64(c.DistanceTo(prev.b)) + float64(c.DistanceTo(next.a))
		if cost < bestCost {
			best, bestCost = c, cost
		}
	}
	setSegEnd(prev, best)
	setSegStart(next, best)
	return true
}

// supportIntersections intersects the two segs' unbounded support curves
// (line↔line, line↔circle, circle↔circle; a polyline's corner end behaves as
// its end segment's line).
func supportIntersections(a, b *wireSeg) []math.Point2 {
	if a.kind == wsArc && b.kind == wsArc {
		return geom.Circle2dCircle2dIntersection(supportCircle(a), supportCircle(b), 0)
	}
	if a.kind == wsArc {
		return circleLineCands(supportCircle(a), supportLine(b, false))
	}
	if b.kind == wsArc {
		return circleLineCands(supportCircle(b), supportLine(a, true))
	}
	if p, ok := rayIntersection(segEndAnchor(a), a.endTangent(), segStartAnchor(b), b.startTangent()); ok {
		return []math.Point2{p}
	}
	return nil
}

func circleLineCands(c geom.Circle2d, l geom.Line2d) []math.Point2 {
	return geom.LineCircle2dIntersection(l, c, 0)
}

func supportCircle(s *wireSeg) geom.Circle2d { return geom.NewCircle2d(s.center, s.r) }

// supportLine is the unbounded line through a line/poly seg's joint end.
func supportLine(s *wireSeg, atEnd bool) geom.Line2d {
	anchor, dir := segStartAnchor(s), s.startTangent()
	if atEnd {
		anchor, dir = segEndAnchor(s), s.endTangent()
	}
	l, _ := geom.NewLine2d(anchor, dir)
	return l
}

func segStartAnchor(s *wireSeg) math.Point2 { return s.a }
func segEndAnchor(s *wireSeg) math.Point2   { return s.b }

// setSegEnd moves a seg's end to p (an arc re-aims its sweep; a poly replaces
// its last vertex).
func setSegEnd(s *wireSeg, p math.Point2) {
	switch s.kind {
	case wsArc:
		end := stdmath.Atan2(float64(p.Y-s.center.Y), float64(p.X-s.center.X))
		s.sweep = wrapTowards(end-s.a0, s.sweep)
		s.syncArcEnds()
	case wsPoly:
		s.poly[len(s.poly)-1] = p
		s.b = p
	default:
		s.b = p
	}
}

// setSegStart moves a seg's start to p.
func setSegStart(s *wireSeg, p math.Point2) {
	switch s.kind {
	case wsArc:
		start := stdmath.Atan2(float64(p.Y-s.center.Y), float64(p.X-s.center.X))
		end := s.a0 + s.sweep
		s.a0 = start
		s.sweep = wrapTowards(end-start, s.sweep)
		s.syncArcEnds()
	case wsPoly:
		s.poly[0] = p
		s.a = p
	default:
		s.a = p
	}
}

// wrapTowards maps delta into the 2π band sharing reference's sign — a trim
// shortens an arc, never flips its direction.
func wrapTowards(delta, reference float64) float64 {
	d := stdmath.Mod(delta, 2*stdmath.Pi)
	if reference >= 0 && d < 0 {
		d += 2 * stdmath.Pi
	}
	if reference < 0 && d > 0 {
		d -= 2 * stdmath.Pi
	}
	return d
}

// segLength is the seg's approximate length (degeneracy filter).
func segLength(s *wireSeg) float64 {
	if s.kind == wsArc {
		return stdmath.Abs(s.sweep) * s.r
	}
	if s.kind == wsPoly {
		sum := 0.0
		for i := 0; i+1 < len(s.poly); i++ {
			sum += float64(s.poly[i].DistanceTo(s.poly[i+1]))
		}
		return sum
	}
	return float64(s.a.DistanceTo(s.b))
}

// buildWireBody lifts the joined 2D chain back to a 3D wire-only body.
func buildWireBody(segs []wireSeg, pl wirePlane, closed bool) (*topo.Body, error) {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("wireOffset", "body", 0)))
	verts := wireChainVertices(bld, segs, pl, closed)
	uses := make([]topo.Use, len(segs))
	for i := range segs {
		curve, err := segCurve3(&segs[i], pl)
		if err != nil {
			return nil, err
		}
		end := verts[(i+1)%len(verts)]
		if !closed && i == len(segs)-1 {
			end = verts[len(verts)-1]
		}
		e := bld.AddEdge(curve, verts[i], end, topo.NewLineage(topo.Tok("wireOffset", "edge", i)))
		uses[i] = topo.Fwd(e)
	}
	body := bld.Build()
	body.AttachWire(topo.NewLineage(topo.Tok("wireOffset", "wire", 0)), uses)
	return body, nil
}

// wireChainVertices mints the chain's junction vertices (one per seg start,
// plus the open end).
func wireChainVertices(bld *topo.Builder, segs []wireSeg, pl wirePlane, closed bool) []*topo.Vertex {
	var verts []*topo.Vertex
	for i := range segs {
		verts = append(verts, bld.AddVertex(pl.to3(segs[i].a), topo.NewLineage(topo.Tok("wireOffset", "vertex", i))))
	}
	if !closed {
		last := &segs[len(segs)-1]
		verts = append(verts, bld.AddVertex(pl.to3(last.b), topo.NewLineage(topo.Tok("wireOffset", "vertex", len(segs)))))
	}
	return verts
}

// segCurve3 lifts one 2D seg to its 3D curve.
func segCurve3(s *wireSeg, pl wirePlane) (geom.Curve3, error) {
	switch s.kind {
	case wsArc:
		return geom.NewArc3d(pl.to3(s.center), pl.n, pl.u, s.r, s.a0, s.sweep)
	case wsPoly:
		pts := make([]math.Point3, len(s.poly))
		for i, p := range s.poly {
			pts[i] = pl.to3(p)
		}
		return geom.NewPolyline(pts)
	default:
		return geom.NewLineSegment(pl.to3(s.a), pl.to3(s.b)), nil
	}
}
