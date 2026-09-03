// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// planeFaceUV frames a PLANAR face whose boundary loops may carry full-circle CONIC edges (a cap's rim
// circle, a seat's hole) for the shared (u,v) trimmer, split by STRAIGHT imprint segments — the
// plane∩plane intersections the mixed per-face boolean produces (ADR-0058). It is the DUAL of planeUV
// (polygon frame + conic imprint): here the frame carries the conics, each sampled with its exact
// crossings against every imprint segment injected as shared vertices of BOTH (the crossing point is
// conic.PointAt(tConic), the closed-form weld currency), so the re-emitted sub-arcs and the split
// imprint segments terminate byte-identically — OCCT's BuilderFace/WireSplitter discipline of exact
// edges carried through a sampled arrangement (BOPAlgo, the reference survey in ADR-0058).
type planeFaceUV struct {
	plane     geom.Plane
	loops     []curvedLoop
	res       geom.Resolution
	frameUV   [][]math.Point2 // sampled boundary rings (outer first) for even-odd cell containment
	incidence []math.Point3   // every imprint end and frame crossing: a vertex the neighbours share
}

// newPlaneFaceUV frames a planar face for the mixed split. ok=false for a frame outside the v1 scope:
// a non-plane surface, or a boundary edge that is neither straight nor a FULL geom.Circle (partial
// arcs and other conics keep the pass-through/decline route until their inversion lands).
func newPlaneFaceUV(f curvedFace, res geom.Resolution) (*planeFaceUV, bool) {
	pl, isPlane := f.surface.(geom.Plane)
	if !isPlane {
		return nil, false
	}
	for _, l := range f.loops {
		for _, e := range l.edges {
			if !planeFaceEdgeOK(e) {
				return nil, false
			}
		}
	}
	return &planeFaceUV{plane: pl, loops: f.loops, res: res}, true
}

// planeFaceEdgeOK reports whether one boundary edge is in the v1 frame scope: straight, or a full
// circle (spanning its whole closed domain in either direction).
func planeFaceEdgeOK(e loopEdge) bool {
	if geom.IsStraightCurve(e.curve) {
		return true
	}
	if _, isCircle := e.curve.(geom.Circle); isCircle {
		// A whole circle frames a face; a partial one is an ARC, which the conic machinery takes.
		return isFullDomain(e.t0, e.t1)
	}
	return geom.IsPlaneConicCurve(e.curve)
}

// planeFaceUV satisfies uvSide: a non-periodic, exact-loop-framed plane (the planeUV conventions).
func (c *planeFaceUV) paramOf(p math.Point3) math.Point2            { return to2D(c.plane, p) }
func (c *planeFaceUV) placeSeams(_ []geom.Curve3)                   {}
func (c *planeFaceUV) vPeriodic() bool                              { return false }
func (c *planeFaceUV) uPeriodic() bool                              { return false }
func (c *planeFaceUV) wrapsAllU() bool                              { return false }
func (c *planeFaceUV) multiFace() bool                              { return true }
func (c *planeFaceUV) emitRun(run []recoveredEdge) (loopEdge, bool) { return emitImprintRun(run) }
func (c *planeFaceUV) finalizeLoops(loops []curvedLoop) []curvedLoop {
	return splitLoopsAtPoints(loops, c.incidence, c.res)
}

func (c *planeFaceUV) wrappingSolidFaces(_ []Face2D, _ []uvSeg, _ geom.Surface, _ curvedFace) ([]curvedFace, bool) {
	return nil, false // a bounded plane never wraps
}

// orientLoops applies the plane's winding convention: every kept loop's edges run forward; the imprint
// sub-segments stay in the face loops (the mixed boolean's neighbour fragments weld on them by
// coordinates, no lid is assembled here). outerless never applies to a bounded plane.
func (c *planeFaceUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	return outerFirst(loops), nil, false
}

// outerFirst orders emitted loops by the (u,v) area they enclose, largest first: a face's OUTER loop
// bounds every hole, so it is the one with the largest magnitude. The emitter orders loops by mean v
// (the ruled band's "hi rim first" convention), which put a pocket's square hole ahead of a cap's rim
// and had the cap misfiled as a polygonal face with a detached hole (ADR-0060).
func outerFirst(loops []emittedLoop) []curvedLoop {
	ordered := append([]emittedLoop{}, loops...)
	sort.SliceStable(ordered, func(i, j int) bool { return stdmath.Abs(ordered[i].area) > stdmath.Abs(ordered[j].area) })
	faceLoops := make([]curvedLoop, 0, len(ordered))
	for _, e := range ordered {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
	}
	return faceLoops
}

// faceFrameCrossing is one exact crossing of a conic FRAME edge with a straight imprint segment: the
// frame edge (loop, edge), the imprint index and its segment parameter, the conic parameter, and the
// shared 3D point conic.PointAt(tConic) both sides terminate on.
type faceFrameCrossing struct {
	loop, edge int
	imp        int
	sImp       float64
	tConic     float64
	at         math.Point3
}

// assembleSegments emits the exact frame loops (straight edges whole, circle edges sampled with the
// imprint crossings injected) plus the imprint segments split at those crossings (uvSide). A CLOSED conic
// island (a ruled wall's section — curved_plane_face_uv_island.go) crosses no frame edge by construction,
// so it is sampled on its own and takes no part in the crossing injection.
func (c *planeFaceUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	straight, islands, open := splitImprintByKind(imprint)
	crossings := c.frameCrossings(straight)
	// An OPEN conic (a hyperbola branch — see curved_plane_face_uv_open.go) crosses the frame's
	// STRAIGHT edges, so its crossings are solved first and the frame is split on them.
	openCrossings := c.openFrameCrossings(open)
	c.incidence = c.incidencePoints(straight, open, crossings, openCrossings)
	segs := append(c.frameSegs(crossings, openCrossings), c.imprintSegs(straight, crossings)...)
	segs = append(segs, c.openSegs(open, openCrossings)...)
	return append(segs, c.islandSegs(islands)...)
}

// frameCrossings solves every conic-frame-edge ∩ imprint-segment crossing in closed form. Every
// non-straight frame edge is a conic or an arc of one (planeFaceEdgeOK), so the crossing solved on the
// whole conic is kept only where it falls within the arc's own span (ADR-0060).
func (c *planeFaceUV) frameCrossings(imprint []geom.Curve3) []faceFrameCrossing {
	var out []faceFrameCrossing
	for li, l := range c.loops {
		for ei, e := range l.edges {
			if !geom.IsStraightCurve(e.curve) {
				out = append(out, c.conicEdgeFrameCrossings(li, ei, e, imprint)...)
			}
		}
	}
	return out
}

// conicEdgeFrameCrossings intersects one conic frame edge with every straight imprint segment.
func (c *planeFaceUV) conicEdgeFrameCrossings(li, ei int, e loopEdge, imprint []geom.Curve3) []faceFrameCrossing {
	pc, ok := toPlaneConic(e.curve, c.plane)
	if !ok {
		return nil
	}
	var out []faceFrameCrossing
	for ii, imp := range imprint {
		a2, b2 := to2D(c.plane, imp.PointAt(0)), to2D(c.plane, imp.PointAt(1))
		hits, _ := conicFrameHits(pc, a2, b2, c.res)
		for _, h := range hits {
			if tc, ok := edgeParamWithin(e, to3D(c.plane, h.p)); ok {
				out = append(out, faceFrameCrossing{loop: li, edge: ei, imp: ii, sImp: h.sEdge, tConic: tc, at: e.curve.PointAt(tc)})
			}
		}
	}
	return out
}

// edgeParamWithin inverts a conic edge at a point on its conic and accepts it only within the edge's
// span [t0, t1] (either order; a whole closed curve spans everything). The span is judged in the
// curve's own parameter, so a partial arc keeps only the crossings that lie ON the arc.
func edgeParamWithin(e loopEdge, p math.Point3) (float64, bool) {
	t, ok := geom.ConicParamAt(e.curve, p)
	if !ok {
		return 0, false
	}
	lo, hi := stdmath.Min(e.t0, e.t1), stdmath.Max(e.t0, e.t1)
	if dlo, dhi := e.curve.Domain(); geom.CurveIsClosed(e.curve) && hi-lo >= (dhi-dlo)-tjTol {
		return t, true
	}
	if t < lo-tjTol || t > hi+tjTol {
		return 0, false
	}
	return stdmath.Max(lo, stdmath.Min(hi, t)), true
}

// frameSegs emits the boundary loops into tagged segments, filling frameUV with the sampled rings the
// material containment reads.
func (c *planeFaceUV) frameSegs(crossings []faceFrameCrossing, open [][]openCrossing) []uvSeg {
	c.frameUV = make([][]math.Point2, 0, len(c.loops))
	var out []uvSeg
	for li, l := range c.loops {
		var ring []math.Point2
		for ei, e := range l.edges {
			segs := c.frameEdgeSegs(li, ei, e, crossings, open)
			out = append(out, segs...)
			for _, s := range segs {
				ring = append(ring, s.a)
			}
		}
		c.frameUV = append(c.frameUV, ring)
	}
	return out
}

// frameEdgeSegs emits one boundary edge: a straight edge as its single exact segment, a circle edge
// sampled along its traversal with every crossing parameter injected so a vertex lands EXACTLY on the
// shared crossing point.
func (c *planeFaceUV) frameEdgeSegs(li, ei int, e loopEdge, crossings []faceFrameCrossing, open [][]openCrossing) []uvSeg {
	if geom.IsStraightCurve(e.curve) {
		return c.straightFrameEdgeSegs(li, ei, e, open)
	}
	params := conicSampleParams(e, crossings, li, ei)
	segs := make([]uvSeg, 0, len(params)-1)
	for i := 1; i < len(params); i++ {
		a3, b3 := e.curve.PointAt(params[i-1]), e.curve.PointAt(params[i])
		segs = append(segs, uvSeg{a: to2D(c.plane, a3), b: to2D(c.plane, b3), curve: e.curve, tA: params[i-1], tB: params[i], kind: segPolygon})
	}
	return segs
}

// conicSampleParams builds the sample parameters of one conic frame edge, in the edge's own
// traversal order (t0→t1), with each crossing's exact conic parameter injected.
func conicSampleParams(e loopEdge, crossings []faceFrameCrossing, li, ei int) []float64 {
	lo, hi := stdmath.Min(e.t0, e.t1), stdmath.Max(e.t0, e.t1)
	params := make([]float64, 0, imprintSampleCount+4)
	for i := 0; i <= imprintSampleCount; i++ {
		params = append(params, lo+(hi-lo)*float64(i)/imprintSampleCount)
	}
	for _, cr := range crossings {
		if cr.loop == li && cr.edge == ei {
			params = append(params, cr.tConic)
		}
	}
	params = sortedUniqueParams(params)
	if e.t0 > e.t1 { // the loop walks the circle backwards: emit in traversal order
		for i, j := 0, len(params)-1; i < j; i, j = i+1, j-1 {
			params[i], params[j] = params[j], params[i]
		}
	}
	return params
}

// imprintSegs splits every straight imprint segment at its conic-frame crossings (ordered along the
// segment) so each sub-segment terminates exactly on the shared crossing points.
func (c *planeFaceUV) imprintSegs(imprint []geom.Curve3, crossings []faceFrameCrossing) []uvSeg {
	var out []uvSeg
	for ii, imp := range imprint {
		verts := imprintVertices(imp, sortedImprintCrossings(crossings, ii))
		for i := 1; i < len(verts); i++ {
			seg := geom.NewLineSegment(verts[i-1], verts[i])
			out = append(out, uvSeg{a: to2D(c.plane, verts[i-1]), b: to2D(c.plane, verts[i]), curve: seg, tA: 0, tB: 1, kind: segImprint})
		}
	}
	return out
}

// imprintVertices is one imprint segment's vertex chain: its two endpoints, with every conic-frame
// crossing ordered along it. A crossing AT an endpoint REPLACES that endpoint with conic.PointAt(t) —
// the very point the neighbouring sub-arc terminates on — instead of inserting a near-duplicate ahead
// of it. An imprint clipped to the frame conic ends ON it by construction, so without this the split
// face's loop closes only to the sampling error of the frame ring (#3488).
func imprintVertices(imp geom.Curve3, on []faceFrameCrossing) []math.Point3 {
	head, tail, mid := imp.PointAt(0), imp.PointAt(1), make([]math.Point3, 0, len(on))
	for _, cr := range on {
		switch {
		case cr.sImp <= tjTol:
			head = cr.at
		case cr.sImp >= 1-tjTol:
			tail = cr.at
		default:
			mid = append(mid, cr.at)
		}
	}
	return append(append([]math.Point3{head}, mid...), tail)
}

// sortedImprintCrossings returns the crossings on one imprint segment ordered along it.
func sortedImprintCrossings(crossings []faceFrameCrossing, imp int) []faceFrameCrossing {
	var on []faceFrameCrossing
	for _, cr := range crossings {
		if cr.imp == imp {
			on = append(on, cr)
		}
	}
	sort.Slice(on, func(i, j int) bool { return on[i].sImp < on[j].sImp })
	return on
}

// planeFaceMaterial builds the mixed split's material predicate: keep a cell whose interior point lies
// inside the sampled face frame (even-odd, the same chords that bound the cells) AND that the boolean's
// keep table selects (bound by the caller over the other operand's membership oracle). A closure so it
// reads frameUV after assembleSegments has filled it.
func planeFaceMaterial(c *planeFaceUV, keep func(math.Point3) bool) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			return pointInUVLoops(uv, c.frameUV) && keep(to3D(c.plane, uv))
		}
	}
}

// planeFaceContactOK declines grazing contact before the trim runs: an imprint segment tangent to a
// frame circle (a double root within the weld) risks a sliver cell the arrangement cannot resolve —
// the same gate planeUV applies to its conic imprints (#1591). The frame test reads only the STRAIGHT
// imprints (a closed conic island reaches the trim already proven clear of every frame edge —
// wallSectionIsland — and a conic-framed face never receives one); the islands are gated separately, on
// meeting nothing they would have to meet on a sampled chord (islandContactOK).
func planeFaceContactOK(c *planeFaceUV, imprint []geom.Curve3) bool {
	// The OPEN conics take no part in either gate: they cross the frame's STRAIGHT edges by
	// construction, and those crossings are solved exactly and shared, not resolved on a chord.
	straight, islands, _ := splitImprintByKind(imprint)
	for _, l := range c.loops {
		for _, e := range l.edges {
			if !geom.IsStraightCurve(e.curve) && !conicEdgeContactOK(c, e, straight) {
				return false
			}
		}
	}
	return islandContactOK(c, islands, straight)
}

// conicEdgeContactOK reports whether no imprint segment grazes one conic frame edge tangentially —
// a double root on the conic that lies within the edge's own span.
func conicEdgeContactOK(c *planeFaceUV, e loopEdge, imprint []geom.Curve3) bool {
	pc, ok := toPlaneConic(e.curve, c.plane)
	if !ok {
		return false
	}
	for _, imp := range imprint {
		a2, b2 := to2D(c.plane, imp.PointAt(0)), to2D(c.plane, imp.PointAt(1))
		hits, tangent := conicEdgeHits(pc, a2, b2, c.res)
		if !tangent {
			continue
		}
		for _, h := range hits {
			if _, on := edgeParamWithin(e, to3D(c.plane, h.p)); on {
				return false
			}
		}
	}
	return true
}

// incidencePoints gathers every point where the imprint meets the frame — each straight imprint's two
// ends, each open conic's ends, and every solved crossing — the vertices finalizeLoops restores on the
// re-emitted edges.
func (c *planeFaceUV) incidencePoints(straight, open []geom.Curve3, crossings []faceFrameCrossing, openCrossings [][]openCrossing) []math.Point3 {
	var pts []math.Point3
	for _, imp := range append(append([]geom.Curve3{}, straight...), open...) {
		pts = append(pts, imp.PointAt(0), imp.PointAt(1))
	}
	for _, cr := range crossings {
		pts = append(pts, cr.at)
	}
	for _, ocs := range openCrossings {
		for _, oc := range ocs {
			pts = append(pts, oc.at)
		}
	}
	return pts
}
