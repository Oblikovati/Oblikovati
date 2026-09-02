// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// An OPEN curved imprint on a planar face: the third kind, beside the straight segment and the
// closed conic island.
//
// A plane cuts a cone in a circle, an ellipse or a HYPERBOLA BRANCH, depending only on how it is
// tilted. The first two are closed and are sampled whole (curved_plane_face_uv_island.go); the
// third is unbounded and closes nothing, so it was falling into the STRAIGHT bucket and being read
// as the chord between PointAt(0) and PointAt(1). The face then split on a straight line while the
// ruled wall on the other side of the same contact split on the exact arc — one shared edge, two
// different boundaries, and a shell that cannot close. That is what stopped an emboss pad from
// joining onto a chamfer cone (#3459).
//
// The treatment mirrors the closed island's, with one difference forced by unboundedness: the span
// to sample comes from the curve's CROSSINGS with the frame rather than from its domain, because
// its domain is the whole real line. The crossing points are shared verbatim — the frame edge is
// split at conic.PointAt(t) and the imprint terminates on the same value — so the two sides weld
// byte-identically rather than to within a sampling error.

// openCrossing is one exact crossing of an open conic imprint with a STRAIGHT frame edge.
type openCrossing struct {
	loop, edge int
	sEdge      float64 // parameter along the frame edge, in [0,1]
	tConic     float64 // parameter along the conic
	at         math.Point3
}

// openFrameCrossings solves every (open conic, straight frame edge) crossing in closed form,
// returning one list per open curve, each ordered along the conic.
func (c *planeFaceUV) openFrameCrossings(open []geom.Curve3) [][]openCrossing {
	out := make([][]openCrossing, len(open))
	for oi, cv := range open {
		pc, ok := toPlaneConic(cv, c.plane)
		if !ok {
			continue
		}
		for li, l := range c.loops {
			for ei, e := range l.edges {
				if _, isCircle := e.curve.(geom.Circle); isCircle {
					continue // circle frame edges are handled by frameCrossings
				}
				out[oi] = append(out[oi], c.openEdgeCrossings(cv, pc, li, ei, e)...)
			}
		}
		sort.Slice(out[oi], func(i, j int) bool { return out[oi][i].tConic < out[oi][j].tConic })
	}
	return out
}

// openEdgeCrossings intersects one open conic with one straight frame edge.
func (c *planeFaceUV) openEdgeCrossings(cv geom.Curve3, pc planeConic, li, ei int, e loopEdge) []openCrossing {
	a2, b2 := to2D(c.plane, e.start()), to2D(c.plane, e.end())
	hits, _ := conicFrameHits(pc, a2, b2, c.res)
	out := make([]openCrossing, 0, len(hits))
	for _, h := range hits {
		t, ok := geom.ConicParamAt(cv, to3D(c.plane, h.p))
		if !ok {
			continue
		}
		out = append(out, openCrossing{loop: li, edge: ei, sEdge: h.sEdge, tConic: t, at: cv.PointAt(t)})
	}
	return out
}

// openSegs samples each open conic over the span its crossings bound, tagged with the SOURCE curve
// and its parameters so a kept boundary run re-emits the exact analytic arc. A curve with fewer
// than two crossings never enters the face and contributes nothing.
func (c *planeFaceUV) openSegs(open []geom.Curve3, crossings [][]openCrossing) []uvSeg {
	var out []uvSeg
	for oi, cv := range open {
		params, ok := openConicSampleParams(crossings[oi])
		if !ok {
			continue
		}
		for i := 1; i < len(params); i++ {
			a3, b3 := cv.PointAt(params[i-1]), cv.PointAt(params[i])
			out = append(out, uvSeg{
				a: to2D(c.plane, a3), b: to2D(c.plane, b3),
				curve: cv, tA: params[i-1], tB: params[i], kind: segImprint,
			})
		}
	}
	return out
}

// openConicSampleParams is the sample parameter list for an unbounded imprint: the span between its
// outermost crossings, widened so the sampled polyline crosses the frame instead of ending on it,
// with every crossing parameter injected so a vertex lands EXACTLY on the shared point.
func openConicSampleParams(crossings []openCrossing) ([]float64, bool) {
	if len(crossings) < 2 {
		return nil, false
	}
	lo, hi := crossings[0].tConic, crossings[len(crossings)-1].tConic
	if hi-lo <= 0 {
		return nil, false
	}
	pad := openImprintSpanPad * (hi - lo)
	lo, hi = lo-pad, hi+pad
	params := make([]float64, 0, imprintSampleCount+len(crossings)+1)
	for i := 0; i <= imprintSampleCount; i++ {
		params = append(params, lo+(hi-lo)*float64(i)/imprintSampleCount)
	}
	for _, cr := range crossings {
		params = append(params, cr.tConic)
	}
	return sortedUniqueParams(params), true
}

// openImprintSpanPad widens an unbounded imprint's sampled span past its outermost crossings, as a
// FRACTION of that span. It is dimensionless, so it carries no model scale.
const openImprintSpanPad = 0.05 // tol:numeric — a fraction of the crossing span, not a length

// straightFrameEdgeSegs emits one STRAIGHT frame edge, split at the open-conic crossings on it so
// each sub-edge terminates exactly on the shared crossing point conic.PointAt(t) — the same value
// the imprint terminates on, which is what lets the two weld.
func (c *planeFaceUV) straightFrameEdgeSegs(li, ei int, e loopEdge, open [][]openCrossing) []uvSeg {
	on := crossingsOnEdge(open, li, ei)
	if len(on) == 0 {
		return []uvSeg{{a: to2D(c.plane, e.start()), b: to2D(c.plane, e.end()), curve: e.curve, tA: e.t0, tB: e.t1, kind: segPolygon}}
	}
	sort.Slice(on, func(i, j int) bool { return on[i].sEdge < on[j].sEdge })
	pts := make([]math.Point3, 0, len(on)+2)
	ss := make([]float64, 0, len(on)+2)
	pts, ss = append(pts, e.start()), append(ss, 0)
	for _, cr := range on {
		pts, ss = append(pts, cr.at), append(ss, cr.sEdge)
	}
	pts, ss = append(pts, e.end()), append(ss, 1)
	out := make([]uvSeg, 0, len(pts)-1)
	for i := 1; i < len(pts); i++ {
		if float64(pts[i-1].DistanceTo(pts[i])) <= c.res.Weld() {
			continue // a crossing sitting on a corner: no sub-edge between them
		}
		out = append(out, uvSeg{
			a: to2D(c.plane, pts[i-1]), b: to2D(c.plane, pts[i]), curve: e.curve,
			tA: e.t0 + ss[i-1]*(e.t1-e.t0), tB: e.t0 + ss[i]*(e.t1-e.t0), kind: segPolygon,
		})
	}
	return out
}

// crossingsOnEdge gathers every open-conic crossing recorded on one frame edge.
func crossingsOnEdge(open [][]openCrossing, li, ei int) []openCrossing {
	var on []openCrossing
	for _, list := range open {
		for _, cr := range list {
			if cr.loop == li && cr.edge == ei {
				on = append(on, cr)
			}
		}
	}
	return on
}

// openConicKind reports whether a curve is an OPEN conic — one this file's treatment applies to.
// A closed conic is an island; a straight segment is neither.
func openConicKind(cv geom.Curve3) bool {
	cf, ok := geom.AsConic(cv)
	if !ok {
		return false
	}
	lo, hi := cv.Domain()
	return cf.Hyperbolic || stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0)
}

// clipSectionToFace bounds an OPEN section curve to the span between its outermost crossings with a
// planar face's trim, returning that bounded arc.
//
// It is the co-refinement step. An unbounded section has to be bounded before either arrangement
// sees it, or each bounds it in its own chart — the face against its polygon, the ruled wall
// against its neighbouring sections — and the two arrive at the same corner by different routes.
// Clipping once, here, gives both sides literally the same curve with the same endpoints, so the
// shared edge welds instead of leaving a T-junction (#3459).
//
// ok=false when the curve is not a conic in the face's plane, or crosses its trim fewer than twice
// (it does not pass through the face, so there is nothing to share).
func clipSectionToFace(cv geom.Curve3, uf curvedFace) (geom.Curve3, bool) {
	pl := facePlane(uf)
	pc, ok := toPlaneConic(cv, pl)
	if !ok {
		return nil, false
	}
	res := geom.ResolutionForBox(faceLoopBox(uf))
	lo, hi, found := stdmath.Inf(1), stdmath.Inf(-1), 0
	for _, ring := range planarRings(uf) {
		for i, n := 0, len(ring); i < n; i++ {
			hits, _ := conicFrameHits(pc, to2D(pl, ring[i]), to2D(pl, ring[(i+1)%n]), res)
			for _, h := range hits {
				t, ok := geom.ConicParamAt(cv, to3D(pl, h.p))
				if !ok {
					continue
				}
				lo, hi, found = stdmath.Min(lo, t), stdmath.Max(hi, t), found+1
			}
		}
	}
	if found < 2 || hi <= lo {
		return nil, false
	}
	return geom.ConicSubArc(cv, lo, hi)
}
