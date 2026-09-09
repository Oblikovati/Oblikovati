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

// openCrossing is one exact crossing of an open conic imprint with a frame edge, straight or curved.
type openCrossing struct {
	loop, edge int
	sEdge      float64 // fraction along a STRAIGHT frame edge, in [0,1]
	tEdge      float64 // parameter on the frame edge's OWN curve
	tConic     float64 // parameter along the conic
	at         math.Point3
}

// openFrameCrossings solves every (open conic, frame edge) crossing in closed form, returning one list
// per open curve, each ordered along the conic.
//
// A CURVED frame edge used to be skipped, which made an open imprint that ends on one invisible: it
// entered the arrangement as a chord dangling inside the face, bounded nothing, and the face came back
// whole. That is the disc-shaped cap of a cylinder bitten by a tool leaving through the RIM — the
// commonest thing a rim crossing does (ADR-0061 stage 4). It is solved by the same conic-against-conic
// substitution the island rule already uses on such an edge (conicEdgeCrossingPoints).
func (c *planeFaceUV) openFrameCrossings(open []geom.Curve3) [][]openCrossing {
	out := make([][]openCrossing, len(open))
	for oi, cv := range open {
		pc, ok := toPlaneConic(cv, c.plane)
		if !ok {
			continue
		}
		for li, l := range c.loops {
			for ei, e := range l.edges {
				out[oi] = append(out[oi], c.openEdgeCrossings(cv, pc, li, ei, e)...)
			}
		}
		sort.Slice(out[oi], func(i, j int) bool { return out[oi][i].tConic < out[oi][j].tConic })
	}
	return out
}

// openEdgeCrossings intersects one open conic with one frame edge, of either kind.
func (c *planeFaceUV) openEdgeCrossings(cv geom.Curve3, pc planeConic, li, ei int, e loopEdge) []openCrossing {
	pts, _, ok := conicEdgeCrossingPoints(pc, e, c.plane, c.res)
	if !ok {
		return nil
	}
	out := make([]openCrossing, 0, len(pts))
	for _, h := range pts {
		p3 := to3D(c.plane, h)
		t, okT := geom.ConicParamAt(cv, p3)
		tE, okE := geom.CurveParamAt(e.curve, p3)
		if !okT || !okE {
			continue
		}
		out = append(out, openCrossing{loop: li, edge: ei, sEdge: edgeFraction(e, tE), tEdge: tE, tConic: t, at: cv.PointAt(t)})
	}
	return out
}

// edgeFraction places a parameter on the edge's own span as a fraction in [0,1], the currency a
// straight frame edge is split on.
func edgeFraction(e loopEdge, t float64) float64 {
	if e.t1 == e.t0 {
		return 0
	}
	return (t - e.t0) / (e.t1 - e.t0)
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

// openCurvedKind reports whether a curve is an OPEN CURVED arc — one this file's treatment applies to.
// A closed curve is an island; a straight segment is neither.
//
// The property is curvature, not conic-ness. A torus's spiric branch is an open arc exactly as an
// elliptical one is, and everything downstream drives off Domain and PointAt alone. Asking for a conic
// sent every spiric branch to the STRAIGHT bucket, where a curved arc becomes the CHORD between its
// ends: the two branches of a single spiric oval collapsed to a zero-area sliver, the receiving face
// kept everything, and a half-space cut through a torus left its lid untrimmed (ADR-0062).
func openCurvedKind(cv geom.Curve3) bool {
	return !geom.IsStraightCurve(cv) && !geom.CurveIsClosed(cv)
}

// clipSectionToFace bounds a section curve to the pieces of it that lie INSIDE a planar face's trim.
//
// It is the co-refinement step. A section has to be bounded before either arrangement sees it, or
// each bounds it in its own chart — the face against its polygon, the ruled wall against its
// neighbouring sections — and the two arrive at the same corner by different routes. Clipping once,
// here, gives both sides literally the same curves with the same endpoints, so the shared edges weld
// instead of leaving a T-junction (#3459).
//
// It returns EVERY inside run, not one arc between the outermost crossings. A section crossing a face
// can enter and leave it several times, and the outermost span then spans the gap as well: a bore's
// section circle crossing a bar's footprint enters at two opposite ends, so the span between its
// extreme crossings ran straight through the material the bar does not cover.
//
// ok=false when the curve is not a conic in the face's plane, or when a run cannot be bounded.
func clipSectionToFace(cv geom.Curve3, uf curvedFace) ([]geom.Curve3, bool) {
	pl := facePlane(uf)
	pc, ok := toPlaneConic(cv, pl)
	if !ok {
		return nil, false
	}
	cuts := sectionFaceCuts(cv, pc, uf, pl)
	if len(cuts) < 2 {
		// No crossing at all. That is DECIDED — there is no imprint — whenever the section never
		// reaches the face's interior: it runs along the face's own boundary, or it lies outside the
		// trim. Only a section that IS inside without crossing out is an inability (ADR-0061).
		if sectionClearOfFaceInterior(cv, uf) {
			return nil, true
		}
		return nil, false
	}
	out, touches, ok := insideSectionRuns(cv, cuts, uf)
	if !ok {
		return nil, false
	}
	if len(out) == 0 && touches {
		return nil, true // DECIDED: every run is on this face's boundary or outside it, so there is
		// no imprint — an empty answer from a decided walk is a proof, not an inability.
	}
	return out, len(out) > 0
}

// insideSectionRuns bounds the runs of the section that lie inside the face's trim, and reports whether
// any run instead ran along the face's OWN boundary.
//
// A run on the boundary is a contact, not an imprint: nothing splits a face by its own edge (ADR-0060).
// sectionOnFaceBoundary states the same rule for a section that runs along an edge from end to end; a
// closed conic that coincides with only PART of the boundary — a rim circle met by the major arc of a
// D-profile prism seated on it — is that rule asked per run.
func insideSectionRuns(cv geom.Curve3, cuts []float64, uf curvedFace) (out []geom.Curve3, touches, ok bool) {
	res := geom.ResolutionForBox(faceLoopBox(uf))
	for _, run := range sectionRuns(cv, cuts) {
		mid := cv.PointAt(sectionParamAt(cv, run.lo+(run.hi-run.lo)/2))
		if pointOnFaceBoundary(mid, uf, res) {
			touches = true
			continue
		}
		if !faceContainsExact(uf, mid) {
			continue
		}
		arc, got := geom.ConicArcBetween(cv, cv.PointAt(run.lo), cv.PointAt(sectionParamAt(cv, run.hi)), mid)
		if !got {
			return nil, false, false
		}
		out = append(out, arc)
	}
	return out, touches, true
}

// sectionRuns turns the crossing parameters into the runs of curve they bound.
//
// An OPEN section (a hyperbola branch) is bounded by the gaps BETWEEN consecutive cuts. A CLOSED one
// — a circle or an ellipse — has one run MORE than it has gaps, because its last cut runs back round
// to its first: two crossings cut a closed conic into TWO arcs, not one.
//
// Emitting only the gaps left that wrap-around arc unrepresented, and when it was the arc lying inside
// the face the clip found no runs at all and declined a section it had correctly crossed. On a real
// part that single decline started a cascade: the cut fell to triangle CSG, its faceted result became
// the next cut's operand, and a disk that rebuilds as 461 analytic faces came back as 11780 planar
// ones in 3 shells with 28 open edges (Oblikovati/Oblikovati#3459).
func sectionRuns(cv geom.Curve3, cuts []float64) []edgeSpan {
	var out []edgeSpan
	for i := 1; i < len(cuts); i++ {
		out = append(out, edgeSpan{cuts[i-1], cuts[i]})
	}
	if !geom.CurveIsClosed(cv) || len(cuts) < 2 {
		return out
	}
	if lo, hi := cuts[len(cuts)-1], cuts[0]+sectionPeriod; hi-lo > conicSpanSlack {
		out = append(out, edgeSpan{lo, hi})
	}
	return out
}

// sectionPeriod is the parameter span a closed conic covers over its whole curve: Circle and
// EllipseFull are both written on [0, 1], so one turn is 1.
const sectionPeriod = 1.0

// sectionParamAt folds a run's parameter back onto the curve's own domain.
//
// ONLY a closed conic needs it — its wrap-around run is built past the end of its [0, 1] domain — and
// only a closed conic HAS that domain to fold onto. A hyperbola branch is parameterised by a
// hyperbolic angle that runs NEGATIVE either side of its vertex, so folding it onto [0, 1) sends a
// crossing at −5.4e-4 to 0.99946 and the imprint lands on the wrong part of the curve entirely. That
// is what an unconditional fold did to a wrapped emboss: its cone sections are hyperbolas, and the
// join fell back to 1326 faceted planes.
func sectionParamAt(cv geom.Curve3, t float64) float64 {
	if !geom.CurveIsClosed(cv) {
		return t
	}
	return wrapUnitParam(t)
}

// wrapUnitParam folds a closed conic's parameter back onto [0, 1), the domain it reports.
func wrapUnitParam(t float64) float64 {
	t -= stdmath.Floor(t)
	if t >= 1 {
		return 0
	}
	return t
}

// sectionFaceCuts returns the section's own parameters at every crossing with the face's trim,
// ascending and deduplicated — the boundaries the runs above are taken between.
func sectionFaceCuts(cv geom.Curve3, pc planeConic, uf curvedFace, pl geom.Plane) []float64 {
	res := geom.ResolutionForBox(faceLoopBox(uf))
	var cuts []float64
	for _, l := range uf.loops {
		for _, e := range l.edges {
			hits, _, ok := conicEdgeCrossingPoints(pc, e, pl, res)
			if !ok {
				continue
			}
			for _, h := range hits {
				if t, got := geom.ConicParamAt(cv, to3D(pl, h)); got {
					cuts = append(cuts, t)
				}
			}
		}
	}
	sort.Float64s(cuts)
	return dedupedParams(cuts)
}

// dedupedParams collapses crossings that coincide — a corner touched by two boundary edges is one
// crossing, and a run of zero width between them would carry no midpoint to classify.
func dedupedParams(ts []float64) []float64 {
	if len(ts) < 2 {
		return ts
	}
	out := ts[:1]
	for _, t := range ts[1:] {
		if t-out[len(out)-1] > conicSpanSlack {
			out = append(out, t)
		}
	}
	return out
}

// openStraightCrossings solves every (open conic, STRAIGHT IMPRINT) crossing in closed form, returning
// the parameters to inject on each side.
//
// The chart already splits an open conic where it meets the FRAME, and a straight imprint where it meets
// a conic frame edge. What neither covers is one imprint meeting another: the chord a planar neighbour
// cuts across this face, crossing the hyperbola branch a ruled wall sections it with. Both are imprints,
// so no frame crossing is solved for them, and the arrangement then found their crossing between the
// SAMPLED polyline of the branch and the chord — cutting the corner, and landing the shared vertex about
// 3·10⁻⁴ off the exact incidence. A cone cut parallel to its axis stayed open there: the wall's own trim
// put the vertex exactly on its rim, the lid's put it a hair above, and the two could not weld
// (ADR-0062).
//
// This is the same rule ADR-0060 states for the frame — every incidence a chart solves is a shared
// vertex — applied to the pair the frame does not mention.
func (c *planeFaceUV) openStraightCrossings(open, straight []geom.Curve3) ([][]openCrossing, []faceFrameCrossing) {
	byOpen := make([][]openCrossing, len(open))
	var onStraight []faceFrameCrossing
	for oi, cv := range open {
		pc, ok := toPlaneConic(cv, c.plane)
		if !ok {
			continue
		}
		for si, sc := range straight {
			lo, hi := sc.Domain()
			a3, b3 := sc.PointAt(lo), sc.PointAt(hi)
			hits, _ := conicFrameHits(pc, to2D(c.plane, a3), to2D(c.plane, b3), c.res)
			for _, h := range hits {
				t, ok := geom.ConicParamAt(cv, to3D(c.plane, h.p))
				if !ok {
					continue
				}
				at := cv.PointAt(t)
				byOpen[oi] = append(byOpen[oi], openCrossing{loop: imprintIncidence, edge: si, tConic: t, at: at})
				onStraight = append(onStraight, faceFrameCrossing{
					loop: imprintIncidence, edge: si, imp: si, sImp: h.sEdge, tConic: t, at: at,
				})
			}
		}
	}
	return byOpen, onStraight
}

// imprintIncidence marks a crossing that is between two IMPRINTS rather than with a frame edge, so a
// consumer that indexes the frame by loop and edge skips it.
const imprintIncidence = -1

// sortedOpenCrossings orders one open conic's crossings along the conic, which is the order openSegs
// samples between them in.
func sortedOpenCrossings(cs []openCrossing) []openCrossing {
	sort.Slice(cs, func(i, j int) bool { return cs[i].tConic < cs[j].tConic })
	return cs
}

// sectionClearOfFaceInterior reports the section never entering the face's interior: every station is
// on the face's own boundary or outside its trim. It is the decided reading of "no crossing" — the
// answer a coplanar neighbour seated ON this face's rim gives, where the two share an arc of boundary
// and meet nowhere else (ADR-0060, ADR-0061).
func sectionClearOfFaceInterior(cv geom.Curve3, uf curvedFace) bool {
	res := geom.ResolutionForBox(faceLoopBox(uf))
	lo, hi := cv.Domain()
	for i := 0; i <= sectionContactSamples; i++ {
		p := cv.PointAt(lo + (hi-lo)*float64(i)/sectionContactSamples)
		if !pointOnFaceBoundary(p, uf, res) && faceContainsExact(uf, p) {
			return false
		}
	}
	return true
}
