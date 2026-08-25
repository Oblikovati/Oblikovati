// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// Analytic extrude for closed line+arc profiles (#2164). Where buildAnalyticCylinder keeps a single
// full circle analytic, this keeps ANY line+arc loop analytic: each arc profile segment becomes a
// true partial-cylinder side face bounded by real geom.Arc3d cap edges, each line segment a planar
// side face bounded by line cap edges. So a face projected onto a sketch, then offset/filleted, sees
// real arcs — not the ~60 chord segments the faceted prism produced (a piston crown, #2164). It uses
// the SAME lineage tokens as the faceted prism (prismEdges/addCaps/addSides), so one edge per profile
// segment keeps stable reference keys. A NewBody extrude keeps this analytic body untouched (combine
// skips the boolean); a joined/cut extrude re-facets it, as any curved tool does today.

// analyticSeg is one profile segment reduced to sketch-space 2D data in CCW traversal order (a→b).
// An arc also carries its center and a midpoint on its true sweep (for the disambiguating third point
// of geom.Arc3dByThreePoints and the cylinder axis).
type analyticSeg struct {
	isArc       bool
	a, b        math.Point2
	mid, center math.Point2
}

// buildAnalyticExtrusion builds a watertight solid prism from a closed line+arc loop, keeping arcs
// analytic. It returns nil — so the caller falls back to the faceted buildPrism — when the loop has a
// non line/arc segment (a spline/ellipse), has no arc at all (a pure polygon gains nothing analytic),
// or is degenerate.
func buildAnalyticExtrusion(loop sketch.Loop, plane sketch.Plane, sp span, feat string) *topo.Body {
	segs, ok := analyticProfileSegments(loop)
	if !ok || !anyArc(segs) || sp.depth() == 0 {
		return nil
	}
	return assembleAnalyticExtrusion(ccwNormalizedSegments(segs), plane, sp, feat)
}

// anyArc reports whether the loop has at least one arc segment.
func anyArc(segs []analyticSeg) bool {
	for _, s := range segs {
		if s.isArc {
			return true
		}
	}
	return false
}

// analyticProfileSegments reduces the loop's entities to ordered line/arc segments and chains them by
// shared endpoints into one CCW-or-CW ring (traversal a→b). ok is false when any entity is not a line
// or arc (read through the ShapedEntity/ArcShaped capabilities, never a type switch — #1624), or the
// entities do not form a single closed chain.
func analyticProfileSegments(loop sketch.Loop) ([]analyticSeg, bool) {
	raw, ok := rawProfileSegments(loop.Entities())
	if !ok {
		return nil, false
	}
	return chainSegments(raw)
}

// rawProfileSegments reduces each entity to a segment in its NATURAL direction (arc: start→end),
// before chaining orients them into traversal order.
func rawProfileSegments(ents []sketch.ProfileEntity) ([]analyticSeg, bool) {
	if len(ents) < 2 {
		return nil, false
	}
	out := make([]analyticSeg, 0, len(ents))
	for _, pe := range ents {
		s, ok := rawSegmentOf(pe.Entity)
		if !ok {
			return nil, false
		}
		out = append(out, s)
	}
	return out, true
}

// rawSegmentOf reduces one entity to a line or arc segment via the sketch-entity capabilities. ok is
// false for anything that is not a line or arc.
func rawSegmentOf(e sketch.Entity) (analyticSeg, bool) {
	shaped, isShaped := e.(sketch.ShapedEntity)
	if !isShaped {
		return analyticSeg{}, false
	}
	pts := shaped.ShapePoints()
	switch shaped.Kind() {
	case sketch.LineKind:
		if len(pts) != 2 {
			return analyticSeg{}, false
		}
		return analyticSeg{a: pts[0], b: pts[1]}, true
	case sketch.ArcKind:
		arc, ok := e.(sketch.ArcShaped)
		if !ok || len(pts) != 3 {
			return analyticSeg{}, false
		}
		return analyticSeg{isArc: true, a: pts[1], b: pts[2], center: pts[0], mid: arc.ArcMidpoint()}, true
	default:
		return analyticSeg{}, false
	}
}

// chainSegments orients the raw segments into one connected ring: starting from the first, each next
// segment shares an endpoint with the running free end and is flipped so its a meets it. ok is false
// when the segments do not form a single closed chain (a gap, a fork, or a leftover).
func chainSegments(raw []analyticSeg) ([]analyticSeg, bool) {
	tol := segmentChainTol(raw)
	used := make([]bool, len(raw))
	chain := make([]analyticSeg, 0, len(raw))
	chain = append(chain, raw[0])
	used[0] = true
	cur := raw[0].b
	for len(chain) < len(raw) {
		next, nb, found := nextChainSegment(raw, used, cur, tol)
		if !found {
			return nil, false
		}
		chain = append(chain, next)
		cur = nb
	}
	if !cur.IsEqualTo(chain[0].a, tol) { // the ring must close back to the start
		return nil, false
	}
	return chain, true
}

// nextChainSegment finds the unused segment with an endpoint at cur, orients it a→b so a==cur, marks it
// used, and returns it with its new free end.
func nextChainSegment(raw []analyticSeg, used []bool, cur math.Point2, tol float64) (analyticSeg, math.Point2, bool) {
	for i := range raw {
		if used[i] {
			continue
		}
		if raw[i].a.IsEqualTo(cur, tol) {
			used[i] = true
			return raw[i], raw[i].b, true
		}
		if raw[i].b.IsEqualTo(cur, tol) {
			used[i] = true
			s := raw[i]
			s.a, s.b = s.b, s.a // flip to traversal order; mid/center are direction-independent
			return s, s.a, true
		}
	}
	return analyticSeg{}, math.Point2{}, false
}

// segmentChainTol is a small endpoint-match tolerance scaled to the loop's extent.
func segmentChainTol(raw []analyticSeg) float64 {
	span := 0.0
	for _, s := range raw {
		span = stdmath.Max(span, stdmath.Max(stdmath.Abs(float64(s.a.X)), stdmath.Abs(float64(s.a.Y))))
	}
	return stdmath.Max(1e-9, span*1e-9)
}

// ccwNormalizedSegments returns the ring wound counter-clockwise (interior on the left), reversing a
// clockwise ring — the winding buildExtrusionShell also enforces so caps/sides orient consistently.
func ccwNormalizedSegments(segs []analyticSeg) []analyticSeg {
	if segmentsSignedArea(segs) >= 0 {
		return segs
	}
	n := len(segs)
	out := make([]analyticSeg, n)
	for i, s := range segs {
		s.a, s.b = s.b, s.a
		out[n-1-i] = s
	}
	return out
}

// segmentsSignedArea is twice the signed area of the corner polygon (CCW ⇒ positive), used only for
// its sign. Arc bulge is ignored — the corner winding alone fixes the traversal orientation.
func segmentsSignedArea(segs []analyticSeg) float64 {
	area := 0.0
	for i, s := range segs {
		q := segs[(i+1)%len(segs)].a
		area += float64(s.a.X*q.Y - q.X*s.a.Y)
	}
	return area
}
