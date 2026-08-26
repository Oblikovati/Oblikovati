// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// mergeHoleIntoNotch performs the KEML clip (spec "Notched face"): it re-routes the outer loop's
// receded front edge between the two Nodes through the hole rim's HOST-side sub-arc — the ellipse
// arc that stays on the host (fillet-kept) side of the receded boundary, the complement of the
// dip/patch arc Task 2/3/4 use for the FillSurface rim rail — producing the single notched face
// loop OCCT calls result_8 (outer minus the part of the hole above the boundary). ok=false on a
// malformed input: an empty loop, or an outer loop with no single edge spanning both Nodes.
func mergeHoleIntoNotch(outer, hole filletLoop, nodes [2]crossing,
	flat func(math.Point3) math.Point2, back func(math.Point2) math.Point3, trims rimNodeTrims) (filletLoop, bool) {
	if len(outer.pts) == 0 || len(hole.pts) == 0 {
		return filletLoop{}, false
	}
	seg, ok := frontEdgeSegment(outer, nodes, flat)
	if !ok {
		return filletLoop{}, false
	}
	notch := spliceSubArc(outer, seg, orientedHostArc(outer, seg, hole, nodes, flat, back, trims))
	if selfCrosses(notch, flat) {
		return filletLoop{}, false // malformed input: even the near/far-paired splice is not simple
	}
	return notch, true
}

// orientedHostArc extracts the hole's host-side sub-arc (hostSideSubArc) and orients it so its
// FIRST point connects to whichever Node sits nearer outer.pts[seg] — the pairing that keeps the
// merged loop's winding consistent with outer's. hostSideSubArc's native order always starts at
// nodes[1] and ends at nodes[0] (Task 2's fixed dip-range convention: dip = nodes[0].I+1..nodes[1].I,
// host = the complement), so it is reversed exactly when nodes[0] is the nearer one.
//
// (The brief's alternative — try native, keep it unless loopsSelfCross flags it, else reverse —
// has a blind spot here: the WRONG pairing's connector chord lies COLLINEAR with, and passes
// through, the arc's own far endpoint (both sit on the straight receded-boundary line), which is a
// touch/overlap, not a transversal crossing, so segmentsCross never flags it — the wrong pairing
// silently produces a self-consistent but semantically inverted loop (area 760+426.9 instead of
// 760-426.9, confirmed while debugging RED→GREEN, see task-5-report.md). The near/far pairing below
// is exact and sidesteps the blind spot entirely; selfCrosses stays as the final honest-reject net.)
func orientedHostArc(outer filletLoop, seg int, hole filletLoop, nodes [2]crossing,
	flat func(math.Point3) math.Point2, back func(math.Point2) math.Point3, trims rimNodeTrims) filletLoop {
	native := hostSideSubArc(hole, nodes, back, trims)
	if nearerNode(outer, seg, nodes, flat) == 0 {
		return reverseOpenArc(native)
	}
	return native
}

// nearerNode returns 0 or 1: which of nodes[0]/nodes[1] projects closer to outer.pts[seg] along the
// front edge's own direction (both Nodes sit exactly on that line, so the dot product with the
// segment direction is a valid ordering, not just a distance).
func nearerNode(outer filletLoop, seg int, nodes [2]crossing, flat func(math.Point3) math.Point2) int {
	a := flat(outer.pts[seg])
	dir := a.VectorTo(flat(outer.pts[(seg+1)%len(outer.pts)]))
	if a.VectorTo(nodes[0].P).Dot(dir) < a.VectorTo(nodes[1].P).Dot(dir) {
		return 0
	}
	return 1
}

// frontEdgeSegment locates the single outer-loop edge carrying the receded boundary — the one
// whose 2D projection both Nodes lie on — so the splice knows exactly where to cut in. ok=false
// when zero or more than one edge qualifies (a malformed/ambiguous input, honest-reject per ADR-3).
func frontEdgeSegment(outer filletLoop, nodes [2]crossing, flat func(math.Point3) math.Point2) (int, bool) {
	tol := ResolutionForPoints(outer.pts).Weld()
	found, hits := -1, 0
	for i := range outer.pts {
		a, b := flat(outer.pts[i]), flat(outer.pts[(i+1)%len(outer.pts)])
		if onSegment2D(a, b, nodes[0].P, tol) && onSegment2D(a, b, nodes[1].P, tol) {
			found, hits = i, hits+1
		}
	}
	return found, hits == 1
}

// onSegment2D reports whether p lies within tol of the segment a→b, with its projection parameter
// inside [0,1] (a small tol slack so a Node sitting exactly at an endpoint still counts).
func onSegment2D(a, b, p math.Point2, tol float64) bool {
	ab := a.VectorTo(b)
	length := ab.Length()
	if length < tol {
		return false
	}
	t := a.VectorTo(p).Dot(ab) / (length * length)
	slack := tol / length
	if t < -slack || t > 1+slack {
		return false
	}
	return a.TranslateBy(ab.Scale(t)).DistanceTo(p) <= tol
}

// hostSideSubArc extracts the hole rim's HOST-side sub-arc — the complement of Task 2's dip range
// (indices nodes[1].I+1..nodes[0].I, wrapping) — as an open polyline from the exact P+ crossing to
// the exact P- crossing (native forward order). Interior segments carry the rim's per-segment curves
// so the mesher still samples the true rim curve there, not a chord.
//
// The TWO segments touching the truncation crossings P± cannot carry the ORIGINAL untrimmed segment
// curve: the crossing sits INSIDE that segment (nodes[1].I leaving P+, nodes[0].I arriving at P-), so
// the stored curve is domain-mismatched there — its PointAt(0)/PointAt(1) are the ORIGINAL
// pre-truncation samples, not the crossing, so a mesher sampling its [0,1] domain would walk across the
// DISCARDED span and kink exactly at the joint Task 6's topo weld must match. They therefore carry the
// TRIMMED sub-arc of that same conic instead (trims, from the node's own rim parameter): the truncated
// remainder traced on the rim rather than across it. srcE is carried on those partial segments so the
// weld can share the hole-rim edge identity. An all-nil trims (a node the rim solve could not refine)
// falls back to the straight chord this function used to give unconditionally.
func hostSideSubArc(hole filletLoop, nodes [2]crossing, back func(math.Point2) math.Point3, trims rimNodeTrims) filletLoop {
	n := len(hole.pts)
	var arc filletLoop
	arc.addID(back(nodes[1].P), trims.out[1], 0, srcIDAt(hole.srcE, nodes[1].I)) // P+ leg: node -> sample
	for i := (nodes[1].I + 1) % n; ; i = (i + 1) % n {
		if i == nodes[0].I {
			arc.addID(hole.pts[i], trims.in[0], srcIDAt(hole.srcV, i), srcIDAt(hole.srcE, i)) // P- leg
			break
		}
		arc.addID(hole.pts[i], curveAt(hole.curves, i), srcIDAt(hole.srcV, i), srcIDAt(hole.srcE, i))
	}
	arc.addID(back(nodes[0].P), nil, 0, 0)
	return arc
}

// reverseOpenArc reverses an OPEN point sequence (not a closed ring) while preserving provenance and
// re-deriving each segment's curve through its recovered midpoint in the new direction — the
// open-path twin of reverseFilletLoop (fillet_orient.go), which assumes a closed ring's modular
// indexing and cannot be reused for a splice fragment.
func reverseOpenArc(arc filletLoop) filletLoop {
	n := len(arc.pts)
	mids := arcMidpoints(arc)
	out := filletLoop{pts: make([]math.Point3, n), curves: make([]geom.Curve3, n), srcV: make([]uint64, n), srcE: make([]uint64, n)}
	for i := range n {
		out.pts[i] = arc.pts[n-1-i]
		out.srcV[i] = arc.srcV[n-1-i]
		if i == n-1 {
			continue // the last point of an open list has no outgoing segment
		}
		seg := n - 2 - i
		out.srcE[i] = arc.srcE[seg]
		if arc.curves[seg] != nil {
			out.curves[i], _ = geom.Arc3dByThreePoints(out.pts[i], mids[seg], arc.pts[seg])
		}
	}
	return out
}

// spliceSubArc rebuilds outer with the edge leaving pts[seg] replaced by arc's points — the
// mechanical splice; mergeHoleIntoNotch supplies whichever arc orientation keeps the result simple.
// The two new connector segments (outer.pts[seg]→arc's first point, arc's last point→
// outer.pts[seg+1]) are op-generated (srcE=0): the original front-edge identity does not survive
// being split by the notch.
func spliceSubArc(outer filletLoop, seg int, arc filletLoop) filletLoop {
	var out filletLoop
	for i := range seg {
		out.addID(outer.pts[i], outer.curves[i], outer.srcV[i], outer.srcE[i])
	}
	out.addID(outer.pts[seg], nil, outer.srcV[seg], 0)
	for i, p := range arc.pts {
		out.addID(p, arc.curves[i], arc.srcV[i], arc.srcE[i])
	}
	for i := seg + 1; i < len(outer.pts); i++ {
		out.addID(outer.pts[i], outer.curves[i], outer.srcV[i], outer.srcE[i])
	}
	return out
}

// selfCrosses reports whether the loop's 2D projection is a non-simple (self-intersecting) polygon
// — the orientation gate mergeHoleIntoNotch uses to pick the splice direction that stays simple.
func selfCrosses(loop filletLoop, flat func(math.Point3) math.Point2) bool {
	return loopsSelfCross(project2D(loop.pts, flat), nil)
}
