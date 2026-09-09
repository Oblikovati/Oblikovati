// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Bringing a shared boundary to the pairing as whole edges on BOTH sides (ADR-0061 stage 5).
//
// A host wall's rim is ONE closed circle; the boss seated on it presents a bare ARC of that circle.
// The run they share is therefore a whole edge on the boss's side and part of an edge on the host's,
// which no edge-to-edge pairing can match — the configuration ADR-0061 recorded as left unmerged, and
// what an earlier attempt read as "the shared boundary is not a whole edge". It is: it becomes one as
// soon as each side is cut at the other's vertices, which is what the stitch does downstream anyway.
// Doing it here, before the pairing, is the whole of the "partial boundary" problem.
//
// The cuts are restricted to runs the two actually WALK together. A vertex that merely lands on the
// other's curve — a rim's end touching a wall it does not run along — splits nothing, so no face gains
// a vertex its neighbours do not carry.

// splitAtSharedRunEnds returns both faces with their boundaries cut at the other's vertices that fall
// inside a run the two share.
func splitAtSharedRunEnds(a, b curvedFace, res geom.Resolution) (curvedFace, curvedFace) {
	cutsForA, cutsForB := sharedRunEnds(b, a, res), sharedRunEnds(a, b, res)
	a.loops = splitLoopsAtPoints(a.loops, weldedCuts(cutsForA, res), res)
	b.loops = splitLoopsAtPoints(b.loops, weldedCuts(cutsForB, res), res)
	return a, b
}

// weldedCuts drops cut points that repeat one another within the weld. One station is named twice —
// by the rim's end and by the wall edge that starts there — and the two arrive as different bits, which
// splitEdgeAtPoints (whose duplicate test is exact) would cut at twice, minting a zero-length edge
// between them. The first point of each welded group survives, so the order is the caller's.
func weldedCuts(pts []math.Point3, res geom.Resolution) []math.Point3 {
	out := make([]math.Point3, 0, len(pts))
	for _, p := range pts {
		if !anyPointWelds(out, p, res) {
			out = append(out, p)
		}
	}
	return out
}

// anyPointWelds reports whether p is already in the list within the weld tolerance.
func anyPointWelds(pts []math.Point3, p math.Point3, res geom.Resolution) bool {
	for _, q := range pts {
		if float64(q.DistanceTo(p)) <= res.Weld() {
			return true
		}
	}
	return false
}

// sharedRunEnds is every endpoint of `of`'s edges that overlap an edge of `in` — the stations `in` has
// to be cut at for the two to pair edge to edge. splitLoopsAtPoints keeps only those that fall
// strictly inside a span, so a coincident end costs nothing.
func sharedRunEnds(of, in curvedFace, res geom.Resolution) []math.Point3 {
	var pts []math.Point3
	for _, l := range of.loops {
		pts = append(pts, runEndsOfLoop(l, in, res)...)
	}
	return pts
}

// runEndsOfLoop collects one loop's overlapping edges' endpoints.
func runEndsOfLoop(l curvedLoop, in curvedFace, res geom.Resolution) []math.Point3 {
	var pts []math.Point3
	for _, e := range l.edges {
		if overlapsAnyEdge(e, in, res) {
			pts = append(pts, e.start(), e.end())
		}
	}
	return pts
}

// overlapsAnyEdge reports whether the edge walks a common stretch with some edge of the face.
func overlapsAnyEdge(e loopEdge, f curvedFace, res geom.Resolution) bool {
	for _, l := range f.loops {
		for _, o := range l.edges {
			if edgesOverlap(e, o, res) {
				return true
			}
		}
	}
	return false
}

// edgesOverlap reports whether two edges walk a common stretch of ONE curve with positive length. It
// probes each edge's midpoint against the other's span and each end against the other's interior: a
// pair sharing a run shows at one of those stations, a pair that merely meets at a vertex at none.
//
// One station is enough because of what the operands are. These are two faces of one result, so their
// boundary edges do not CROSS at an interior point of both — that would be a self-intersection, which
// Validate rejects. A station of one lying on the other's span therefore means a run, not a crossing.
func edgesOverlap(x, y loopEdge, res geom.Resolution) bool {
	return spansPoint(x, midParamPoint(y), res) || spansPoint(y, midParamPoint(x), res) ||
		endsInsideSpan(x, y, res) || endsInsideSpan(y, x, res)
}

// endsInsideSpan reports whether either end of `of` falls strictly inside `in`'s span.
func endsInsideSpan(in, of loopEdge, res geom.Resolution) bool {
	_, atStart := edgeParamOn(in, of.start(), res)
	_, atEnd := edgeParamOn(in, of.end(), res)
	return atStart || atEnd
}

// spansPoint reports whether p lies on the edge's own parameter span.
func spansPoint(e loopEdge, p math.Point3, res geom.Resolution) bool {
	_, ok := curveParamWithin(e.curve, e.t0, e.t1, p, res)
	return ok
}

// midParamPoint is the edge's parametric midpoint — the station a shared run cannot miss when one edge
// covers the other whole.
func midParamPoint(e loopEdge) math.Point3 { return e.curve.PointAt((e.t0 + e.t1) / 2) }
