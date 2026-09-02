// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// planeUV is a bounded, NON-periodic planar seat face expressed as a uvSide, so a partial curved-on-planar
// boolean (a drilled hole / boss base whose imprint conic CLIPS the face boundary) runs through the one
// shared (u,v)-arrangement trimmer instead of triangle-soup CSG (#1591, ADR-0049). The chart is the plane's
// own (u,v) = to2D(plane, ·) — an isometry, no azimuth seam, no rim circles. The frame is the face POLYGON
// (segPolygon edges) rather than rims+seam, and the periodic machinery is inert: placeSeams a no-op,
// u/vPeriodic and wrapsAllU false. The material predicate keeps face-interior cells that are outside the tool.
type planeUV struct {
	plane  geom.Plane
	seatUV [][]math.Point2        // seat face boundary in (u,v), outer loop first (even-odd containment)
	seat3D [][]math.Point3        // the same loops in 3D, so each polygon edge re-emits as its exact segment
	inTool func(math.Point3) bool // reports whether a 3D point is inside the tool solid (removed for a drill)
	res    geom.Resolution        // model-relative tolerance for the exact conic∩edge crossings
}

// planeCrossing is one exact conic∩polygon-edge crossing: the shared point at = conic.PointAt(tConic), used
// as a vertex by BOTH the conic sample and the split polygon edge so the seat/cap/wall arcs weld exactly.
type planeCrossing struct {
	loop, edge int
	sEdge      float64     // edge parameter, to order splits along one edge
	tConic     float64     // conic parameter (the weld currency)
	at         math.Point3 // the shared crossing point = conic.PointAt(tConic)
}

// planeUV satisfies uvSide: a non-periodic, polygon-framed plane (#1591).
var _ uvSide = (*planeUV)(nil)

// paramOf inverts a seat-plane point to its (u,v) chart coordinates (uvSide) — a plain orthonormal projection,
// no seam offset (the plane is non-periodic).
func (c *planeUV) paramOf(p math.Point3) math.Point2 { return to2D(c.plane, p) }

// placeSeams is a no-op: a bounded plane has no artificial seam to move clear of the imprint (uvSide).
func (c *planeUV) placeSeams(_ []geom.Curve3) {}

// vPeriodic reports that a plane's v does not wrap (uvSide).
func (c *planeUV) vPeriodic() bool { return false }

// uPeriodic reports that a plane's u does not wrap — u is a real world distance, so the boundary welder must
// NOT fold u≈2π onto u=0 (uvSide, ADR-0049 D-c).
func (c *planeUV) uPeriodic() bool { return false }

// wrapsAllU reports that a plane never wraps the azimuth — there is none (uvSide).
func (c *planeUV) wrapsAllU() bool { return false }

// multiFace reports that a plane cut may leave the seat DISCONNECTED (a slot severing the face into two
// pieces), so the boundary loops are grouped into separate faces by containment (uvSide).
func (c *planeUV) multiFace() bool { return true }

// wrappingSolidFaces never applies to a plane (there is no wrapping tube band); it defers to the standard
// contractible-outer emission (uvSide).
func (c *planeUV) wrappingSolidFaces(_ []Face2D, _ []uvSeg, _ geom.Surface, _ curvedFace) ([]curvedFace, bool) {
	return nil, false
}

// finalizeLoops is identity: a plane has no apex-pole rim or torus hole to re-mark (uvSide).
func (c *planeUV) finalizeLoops(loops []curvedLoop) []curvedLoop { return loops }

// emitRun re-emits one boundary run as an exact loopEdge (uvSide). Both the imprint conic (segImprint) and a
// polygon boundary edge (segPolygon) are analytic curves re-emitted over their recovered parameter span, so
// emitImprintRun serves both — the shared re-emitter that makes the seat arc and the tool wall weld exactly.
func (c *planeUV) emitRun(run []recoveredEdge) (loopEdge, bool) { return emitImprintRun(run) }

// orientLoops applies the plane's winding convention (uvSide): every kept loop's edges run forward, and the
// imprint (conic) sub-arcs are reversed into the lid — the shared cut boundary the tool wall reuses so the
// two faces weld. A plane is open in both u and v, so the kept face always has an outer loop (outerless=false).
func (c *planeUV) orientLoops(loops []emittedLoop, _ bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	for _, e := range loops {
		faceLoops = append(faceLoops, curvedLoop{edges: e.face})
		lid = append(lid, reverseEdgeChain(e.section)...)
	}
	return faceLoops, lid, false
}

// assembleSegments samples the imprint conic(s) and the face-polygon frame into the tagged (u,v) segment
// set (uvSide), INJECTING the exact conic∩edge crossings as shared vertices of both. Without the injection
// the arrangement would place the crossing on the sampled chord — off the true conic by the sagitta — so the
// re-emitted arc would miss the polygon edge and the tool wall's split base, leaving a free edge (#1591).
func (c *planeUV) assembleSegments(imprint []geom.Curve3) []uvSeg {
	var crossings []planeCrossing
	out := make([]uvSeg, 0, len(imprint)*imprintSampleCount+len(c.seat3D)*4)
	for _, cv := range imprint {
		pc, ok := toPlaneConic(cv, c.plane)
		if !ok {
			continue
		}
		cr := c.conicCrossings(cv, pc)
		crossings = append(crossings, cr...)
		out = append(out, c.sampleConicUV(cv, cr)...)
	}
	return append(out, c.frameSegments(crossings)...)
}

// conicCrossings returns every exact crossing of one conic with the seat polygon: the closed-form conic
// parameter and the shared 3D point conic.PointAt(t), which both the conic sample and the split polygon edge
// terminate on so they weld byte-identically.
func (c *planeUV) conicCrossings(cv geom.Curve3, pc planeConic) []planeCrossing {
	var out []planeCrossing
	for li, ring := range c.seatUV {
		for i, n := 0, len(ring); i < n; i++ {
			hits, _ := conicEdgeHits(pc, ring[i], ring[(i+1)%n], c.res)
			for _, h := range hits {
				tc, ok := geom.ConicParamAt(cv, to3D(c.plane, h.p))
				if !ok {
					continue
				}
				out = append(out, planeCrossing{loop: li, edge: i, sEdge: h.sEdge, tConic: tc, at: cv.PointAt(tc)})
			}
		}
	}
	return out
}

// sampleConicUV samples one imprint conic into (u,v) segments carrying the source curve + its parameters, so
// a boundary run re-emits as the exact analytic arc. Each crossing parameter is inserted as a sample boundary
// so a vertex lands EXACTLY on the crossing (conic.PointAt(tConic)) — the shared weld point.
func (c *planeUV) sampleConicUV(cv geom.Curve3, crossings []planeCrossing) []uvSeg {
	lo, hi, ok := imprintSampleRange(cv, crossings)
	if !ok {
		return nil
	}
	params := make([]float64, 0, imprintSampleCount+len(crossings)+1)
	for i := 0; i <= imprintSampleCount; i++ {
		params = append(params, lo+(hi-lo)*float64(i)/imprintSampleCount)
	}
	for _, cr := range crossings {
		params = append(params, cr.tConic)
	}
	params = sortedUniqueParams(params)
	segs := make([]uvSeg, 0, len(params))
	for i := 1; i < len(params); i++ {
		a, b := to2D(c.plane, cv.PointAt(params[i-1])), to2D(c.plane, cv.PointAt(params[i]))
		segs = append(segs, uvSeg{a: a, b: b, curve: cv, tA: params[i-1], tB: params[i], kind: segImprint})
	}
	return segs
}

// imprintSampleRange is the parameter interval an imprint curve is sampled over.
//
// A BOUNDED conic — a circle or an ellipse — is sampled across its whole domain, as it always was.
// An UNBOUNDED one has no such domain: geom.Hyperbola reports (-Inf, +Inf), so sampling it that way
// produced infinite parameters and left only the two crossing parameters standing. The imprint then
// collapsed to a single straight CHORD between them, while the wall on the other side of the same
// contact re-emitted the exact arc — two different boundaries for one shared edge, so the shell
// could not close (#3459).
//
// The part of an unbounded branch that can bound a face is the part inside the seat, which lies
// between its crossings with the seat polygon. Sampling that span — widened so the arrangement sees
// the curve continue through each crossing rather than terminate on it — gives the same treatment
// every other conic gets. Fewer than two crossings means the branch never enters the seat, so it
// contributes no imprint at all.
func imprintSampleRange(cv geom.Curve3, crossings []planeCrossing) (lo, hi float64, ok bool) {
	lo, hi = cv.Domain()
	if !stdmath.IsInf(lo, 0) && !stdmath.IsInf(hi, 0) {
		return lo, hi, true
	}
	if len(crossings) < 2 {
		return 0, 0, false
	}
	lo, hi = crossings[0].tConic, crossings[0].tConic
	for _, cr := range crossings[1:] {
		lo, hi = stdmath.Min(lo, cr.tConic), stdmath.Max(hi, cr.tConic)
	}
	if hi-lo <= 0 {
		return 0, 0, false
	}
	pad := unboundedImprintPad * (hi - lo)
	return lo - pad, hi + pad, true
}

// unboundedImprintPad widens an unbounded imprint's sampled span past its outermost crossings, as a
// FRACTION of that span, so the sampled polyline crosses the seat boundary instead of ending on it.
// It is dimensionless, so it carries no model scale.
const unboundedImprintPad = 0.05 // tol:numeric — a fraction of the crossing span, not a length

// frameSegments emits the seat polygon as segPolygon edges, each SPLIT at its conic crossings so the sub-edge
// ends exactly on the shared crossing point (a straight segment to that point, kinked by less than a weld).
func (c *planeUV) frameSegments(crossings []planeCrossing) []uvSeg {
	var out []uvSeg
	for li, ring := range c.seat3D {
		for i, n := 0, len(ring); i < n; i++ {
			out = append(out, c.splitEdge(li, i, ring[i], ring[(i+1)%n], crossings)...)
		}
	}
	return out
}

// splitEdge breaks one polygon edge at its crossings (ordered along the edge), emitting a straight segPolygon
// sub-edge between consecutive vertices — the corners and the shared crossing points conic.PointAt(tConic).
func (c *planeUV) splitEdge(loop, edge int, a3, b3 math.Point3, crossings []planeCrossing) []uvSeg {
	verts := []math.Point3{a3}
	for _, cr := range sortedEdgeCrossings(crossings, loop, edge) {
		verts = append(verts, cr.at)
	}
	verts = append(verts, b3)
	out := make([]uvSeg, 0, len(verts)-1)
	for i := 1; i < len(verts); i++ {
		seg := geom.NewLineSegment(verts[i-1], verts[i])
		out = append(out, uvSeg{a: to2D(c.plane, verts[i-1]), b: to2D(c.plane, verts[i]), curve: seg, tA: 0, tB: 1, kind: segPolygon})
	}
	return out
}

// planeMaterial builds the planeUV material predicate: keep a cell whose deep-interior (u,v) point lies inside
// the seat face polygon AND outside the tool solid (a drill removes the tool; the seat retains the rest). A
// closure (not a bound value) so it reads the receiver live, matching the ruled/torus material builders.
func planeMaterial(c *planeUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			return pointInUVLoops(uv, c.seatUV) && !c.inTool(to3D(c.plane, uv))
		}
	}
}

// sortedUniqueParams sorts the parameter list and drops near-duplicates (an injected crossing that coincides
// with a uniform sample), so no zero-length sample segment is emitted.
func sortedUniqueParams(params []float64) []float64 {
	sort.Float64s(params)
	out := params[:0:0]
	for i, p := range params {
		if i == 0 || p-out[len(out)-1] > 1e-12 {
			out = append(out, p)
		}
	}
	return out
}

// sortedEdgeCrossings returns the crossings on one polygon edge ordered by their edge parameter, so the edge
// splits into consecutive sub-segments in geometric order.
func sortedEdgeCrossings(crossings []planeCrossing, loop, edge int) []planeCrossing {
	var on []planeCrossing
	for _, cr := range crossings {
		if cr.loop == loop && cr.edge == edge {
			on = append(on, cr)
		}
	}
	sort.Slice(on, func(i, j int) bool { return on[i].sEdge < on[j].sEdge })
	return on
}

// capMaterial builds the material predicate for the exposed overhang UNDERSIDE of a boss straddling the seat
// edge: keep a cell inside the tool (the boss footprint) AND outside the seat polygon — the mirror of
// planeMaterial, run on the SAME base-plane arrangement so the two share the imprint arc exactly (#1591).
func capMaterial(c *planeUV) func() materialPredicate {
	return func() materialPredicate {
		return func(uv math.Point2) bool {
			return c.inTool(to3D(c.plane, uv)) && !pointInUVLoops(uv, c.seatUV)
		}
	}
}

// planeCrossingsOf returns the exact seat-boundary crossings of one imprint conic (the assembler shares them
// with the wall base split and the T-junction resolution so every face meets on the same points).
func (c *planeUV) planeCrossingsOf(cv geom.Curve3) []planeCrossing {
	pc, ok := toPlaneConic(cv, c.plane)
	if !ok {
		return nil
	}
	return c.conicCrossings(cv, pc)
}

// pointInUVLoops reports whether q is inside a face given as (u,v) loops (outer first, then holes): inside the
// outer loop and outside every hole — the even-odd containment the planar boolean uses (pointInPolygon2D).
func pointInUVLoops(q math.Point2, loops [][]math.Point2) bool {
	if len(loops) == 0 || !pointInPolygon2D(q, loops[0]) {
		return false
	}
	for _, hole := range loops[1:] {
		if pointInPolygon2D(q, hole) {
			return false
		}
	}
	return true
}
