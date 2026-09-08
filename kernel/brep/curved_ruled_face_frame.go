// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Frame assembly for the loop-framed ruled chart (ADR-0060): the closed-form incidences between the
// face's boundary edges, the imprint curves and the artificial seam, and the sampled segments that
// carry them into the arrangement with those incidences as shared vertices.

// frameCrossing is one incidence of a frame edge (loop, edge) with an imprint curve (imp ≥ 0) or with
// the seam ruling (imp < 0), in each curve's own parameter. The shared arrangement vertex is the frame
// edge's point at tEdge.
// loopFrameHost is what the loop-framing needs from a surface's own chart: the seam-relative
// parameterisation, and the artificial seam curve closing the periodic strip. Everything else about
// framing a face by its own loops — sampling the boundary, solving every frame×imprint and frame×seam
// incidence in closed form, injecting them as shared vertices — is surface-agnostic, which is what
// OCCT's one face-splitting machine says too: it takes the face's edges and the surface's own
// parameterisation and nothing more (ADR-0062).
type loopFrameHost interface {
	paramOf(p math.Point3) math.Point2
	seamCurve() geom.Curve3
	// vWindow is the chart's own v span — a ruled wall's axial window, a sphere's polar range — which
	// the artificial seam runs the length of.
	vWindow() (vMin, vMax float64)
	// point3 inverts the parameterisation: the surface point at seam-relative (u, v).
	point3(u, v float64) math.Point3
	// seamOverrun is how far past the v window the artificial seam runs. A ruled wall's band is OPEN in
	// v, so an overrun costs nothing and guarantees the seam closes the arrangement past every frame
	// vertex. A CLOSED surface's v window is its whole period: running past it maps back onto the
	// geometry — a sphere's seam beyond a pole lands on the antipodal side — so it must be zero there.
	seamOverrun() float64
	// vClosed reports whether the chart's v is a PERIOD rather than a bounded window — a torus's tube
	// angle. A curve sampled across a period wraps, and the polyline must be split at that seam exactly
	// as it is at the u-seam; without it the wrap lands in the arrangement as one segment spanning the
	// whole rectangle, which slices the region into slivers (ADR-0062).
	vClosed() bool
	// tubeSeamCurve is the artificial seam closing a PERIODIC v — the torus's parallel at the placed
	// tube angle — so its crossings with the frame and the imprint are solved exactly like the
	// u-seam's, rather than folded from wherever the sampling straddled it (ADR-0061 stage 4). ok is
	// false on a chart whose v is a bounded window.
	tubeSeamCurve() (geom.Curve3, bool)
}

// loopFrame is a face's OWN boundary as the arrangement's frame (ADR-0060), shared by every chart that
// frames a face that way whatever its surface.
type loopFrame struct {
	host      loopFrameHost
	face      curvedFace
	res       geom.Resolution
	imprint   []geom.Curve3   // the imprint being trimmed, for seam-incidence lookups
	crossings []frameCrossing // frame×imprint incidences, in curve parameters (seam-independent)
	frameSegs []uvSeg         // the seam-relative sampled frame, filled by assembleSegments
}

type frameCrossing struct {
	loop, edge int
	tEdge      float64
	imp        int
	tImp       float64
	tube       bool // a seam incidence on the TUBE seam (periodic v) rather than the azimuth seam
}

const seamIncidence = -1

// solveFrameCrossings intersects every frame edge with every imprint curve in closed form. ok=false when
// an imprint coincides with a frame edge (one section plane) — admits has already dropped those.
func (c *loopFrame) solveFrameCrossings(imprint []geom.Curve3) ([]frameCrossing, bool) {
	var out []frameCrossing
	for li, l := range c.face.loops {
		for ei, e := range l.edges {
			for ii, imp := range imprint {
				if _, coincident := geom.SectionCrossingCandidates(c.face.surface, e.curve, imp, c.res); coincident {
					return nil, false
				}
				tE, tI := c.curvePairMeets(e.curve, e.t0, e.t1, imp)
				for k := range tE {
					out = append(out, frameCrossing{loop: li, edge: ei, tEdge: tE[k], imp: ii, tImp: tI[k]})
				}
			}
		}
	}
	return out, true
}

// solveImprintCrossings solves every meeting BETWEEN imprint curves on this face, returning the
// parameters to inject on each one.
//
// The arrangement welds coincident vertices; it does not split a segment where another crosses it. So
// two imprints that meet without a shared vertex do not cut each other at all: a rod drilled through an
// already-notched cylinder has its exit bounded by the wall crossing over part of the turn and by the
// notch plane's section over the rest, the two meeting at the corner-junction triple points, and with no
// vertex there the rod's chart kept the whole wall crossing and ignored the notch entirely (ADR-0061
// stage 4). It is the same co-refinement planeFaceUV already does between its islands.
func (c *loopFrame) solveImprintCrossings(imprint []geom.Curve3) [][]float64 {
	out := make([][]float64, len(imprint))
	for i := range imprint {
		for j := i + 1; j < len(imprint); j++ {
			if _, coincident := geom.SectionCrossingCandidates(c.face.surface, imprint[i], imprint[j], c.res); coincident {
				continue // two imprints on one curve: the arrangement's vertex weld already joins them
			}
			lo, hi := imprint[i].Domain()
			ti, tj := c.curvePairMeets(imprint[i], lo, hi, imprint[j])
			out[i] = append(out[i], ti...)
			out[j] = append(out[j], tj...)
		}
	}
	return out
}

// curvePairMeets is where two curves ON THIS FACE'S SURFACE meet, as a parameter on EACH — index-aligned,
// a's span restricted to [aLo, aHi].
//
// Each side's parameter is solved in its own walk. The shared point is never recovered by inverting the
// other curve: a section solver locates its candidate only to the accuracy of that candidate, and the
// two charts that meet along a triple point then name it 5e-5 apart, which the stitch resolves by
// splitting a neighbouring edge into a zero-length remnant instead of welding (ADR-0061 stage 4).
func (c *loopFrame) curvePairMeets(a geom.Curve3, aLo, aHi float64, b geom.Curve3) (ta, tb []float64) {
	bLo, bHi := b.Domain()
	for _, tA := range curveRootsOnOther(a, aLo, aHi, b) {
		for _, tB := range curveRootsOnOther(b, bLo, bHi, a) {
			if float64(a.PointAt(tA).DistanceTo(b.PointAt(tB))) <= c.res.Sew() && !c.meetKnown(a, ta, tA) {
				ta, tb = append(ta, tA), append(tb, tB)
			}
		}
	}
	if len(ta) > 0 {
		return ta, tb
	}
	return c.sectionPairMeets(a, aLo, aHi, b)
}

// meetKnown reports whether a meeting point is already in the list. A curve carrying TWO incidence
// conditions — a straight seam, a ruled crossing — has both vanish at one crossing, so the walk brackets
// that crossing once per condition and each root lands at the same point to rounding; injecting both
// would put two vertices a rounding apart on the boundary, and the sliver between them reads as an
// open edge. One point is one incidence, decided once (ADR-0061 stage 4).
func (c *loopFrame) meetKnown(a geom.Curve3, known []float64, tA float64) bool {
	p := a.PointAt(tA)
	for _, t := range known {
		if float64(a.PointAt(t).DistanceTo(p)) <= c.res.Sew() {
			return true
		}
	}
	return false
}

// sectionPairMeets is the section-plane route to the same question, for a pair whose meeting the
// incidence walk does not bracket — a TANGENTIAL contact, where no sign changes. It inverts the shared
// point on each curve, so it answers only for curves with a closed-form parameter inversion.
func (c *loopFrame) sectionPairMeets(a geom.Curve3, aLo, aHi float64, b geom.Curve3) (ta, tb []float64) {
	bLo, bHi := b.Domain()
	pts, _ := geom.SectionCrossingCandidates(c.face.surface, a, b, c.res)
	for _, p := range pts {
		tA, okA := c.paramWithin(a, aLo, aHi, p)
		tB, okB := c.paramWithin(b, bLo, bHi, p)
		if okA && okB {
			ta, tb = append(ta, tA), append(tb, tB)
		}
	}
	return ta, tb
}

// curveRootsOnOther is where a's own parameter, walked over [lo, hi], satisfies b's incidence: b's
// condition evaluated along a is a scalar function, and every sign change is bisected to its root.
//
// The section-plane solver answers only for two PLANAR sections, and a ruled crossing is not planar — a
// rod through an already-notched cylinder meets the notch section at two triple points that solver
// reports none of.
func curveRootsOnOther(a geom.Curve3, lo, hi float64, b geom.Curve3) []float64 {
	if lo > hi {
		lo, hi = hi, lo
	}
	var out []float64
	for _, on := range geom.CurveIncidence(b) {
		g := func(t float64) float64 { return on(a.PointAt(t)) }
		prev, prevG := lo, g(lo)
		for i := 1; i <= crossingSpanSamples; i++ {
			t := lo + (hi-lo)*float64(i)/crossingSpanSamples
			v := g(t)
			switch {
			case v == 0:
				out = append(out, t) // the station IS the root: a seam placed exactly on a sample
			case prevG*v < 0:
				out = append(out, bisectRoot(g, prev, t))
			}
			prev, prevG = t, v
		}
	}
	return out
}

// solveSeamCrossings intersects the placed seam with every frame edge and every imprint curve, through
// the one incidence solver the frame and the imprint use for one another (seamMeets). Rulings are
// skipped: the seam is placed clear of every ruling edge.
func (c *loopFrame) solveSeamCrossings(imprint []geom.Curve3) []frameCrossing {
	out := c.oneSeamCrossings(c.host.seamCurve(), imprint, false)
	if tube, ok := c.host.tubeSeamCurve(); ok {
		out = append(out, c.oneSeamCrossings(tube, imprint, true)...)
	}
	return out
}

// oneSeamCrossings solves one seam's crossings with every frame edge and every imprint curve.
func (c *loopFrame) oneSeamCrossings(seam geom.Curve3, imprint []geom.Curve3, tube bool) []frameCrossing {
	var out []frameCrossing
	for li, l := range c.face.loops {
		for ei, e := range l.edges {
			for _, t := range c.seamMeets(seam, e.curve, e.t0, e.t1) {
				out = append(out, frameCrossing{loop: li, edge: ei, tEdge: t, imp: seamIncidence, tube: tube})
			}
		}
	}
	for ii, imp := range imprint {
		lo, hi := imp.Domain()
		for _, t := range c.seamMeets(seam, imp, lo, hi) {
			out = append(out, frameCrossing{loop: seamIncidence, edge: ii, tEdge: t, imp: seamIncidence, tube: tube})
		}
	}
	return out
}

// seamMeets is every parameter within [t0, t1] at which a curve meets the seam. It is curvePairMeets,
// so a window loop the seam enters and leaves reports BOTH crossings, and a ruled∩quadric arc — which
// has no section plane — is met through its incidence like any other curve. The seam-hit it replaces
// took the section-plane candidates and stopped at the first within both spans: a loop the seam cut
// through then carried one vertex on it, the far half of the loop was lost with the seam edge it should
// have cancelled against, and the near-pinch cut came back with three open edges (ADR-0061 stage 4).
// A crossing must lie on the seam ITSELF, not merely in its plane — a sphere's meridian is half its
// great circle — which the solver's Sew-distance pairing on the seam's own span guarantees.
func (c *loopFrame) seamMeets(seam, cv geom.Curve3, t0, t1 float64) []float64 {
	if geom.IsStraightCurve(cv) {
		return nil
	}
	ta, _ := c.curvePairMeets(cv, t0, t1, seam)
	return ta
}

// paramWithin inverts a curve at a point and accepts it when the point lies on the curve within the
// sew tolerance and its parameter falls in [t0, t1] (either order; a whole closed curve accepts every
// parameter). A parameter a hair outside the span is clamped to the end it overshoots: the incidence
// is then a frame vertex, which the sampling emits anyway.
func (c *loopFrame) paramWithin(cv geom.Curve3, t0, t1 float64, p math.Point3) (float64, bool) {
	return curveParamWithin(cv, t0, t1, p, c.res)
}

// curveParamWithin is loopFrame.paramWithin without a chart: it is the same question — does p lie on
// this stretch of this curve, and at what parameter — asked by the co-refinement, which runs before any
// chart is built (ADR-0062).
func curveParamWithin(cv geom.Curve3, t0, t1 float64, p math.Point3, res geom.Resolution) (float64, bool) {
	t, ok := geom.CurveParamAt(cv, p)
	if !ok {
		return 0, false
	}
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	if dlo, dhi := cv.Domain(); geom.CurveIsClosed(cv) && hi-lo >= (dhi-dlo)-paramSlack(cv, dhi-dlo, res) {
		return t, float64(cv.PointAt(t).DistanceTo(p)) <= res.Sew()
	}
	slack := paramSlack(cv, hi-lo, res)
	t, on := paramOnSpanBranch(cv, t, lo, hi, slack)
	if !on {
		return 0, false
	}
	t = stdmath.Max(lo, stdmath.Min(hi, t))
	return t, float64(cv.PointAt(t).DistanceTo(p)) <= res.Sew()
}

// paramOnSpanBranch places an inverted parameter on the branch the span lives on. CurveParamAt answers
// inside the curve's OWN domain, and a closed curve's span can run up to that domain's end — where the
// same point comes back as the domain's start, a whole period away (ADR-0061: an inverted azimuth
// carries an arbitrary whole turn). A wall's rim split at the boss's ends yields exactly such a span,
// [t, domain end], and without this the piece touching the seam matched nothing. An open curve has no
// other branch, so it is the unchanged direct test.
func paramOnSpanBranch(cv geom.Curve3, t, lo, hi, slack float64) (float64, bool) {
	if !geom.CurveIsClosed(cv) {
		return t, t >= lo-slack && t <= hi+slack
	}
	dlo, dhi := cv.Domain()
	for _, b := range [3]float64{t, t - (dhi - dlo), t + (dhi - dlo)} {
		if b >= lo-slack && b <= hi+slack {
			return b, true
		}
	}
	return t, false
}

// paramSlack converts the sew tolerance to a parameter slack through the curve's speed over the span.
func paramSlack(cv geom.Curve3, span float64, res geom.Resolution) float64 {
	speed := float64(cv.TangentAt(0.5 * span).Length())
	if speed <= 0 {
		return 0
	}
	return res.Sew() / speed
}

// frameSegments samples every frame edge in its traversal order with the imprint and seam incidences
// injected, tagged segPolygon so a boundary run re-emits the exact sub-edge.
func (c *loopFrame) frameSegments(seamHits []frameCrossing) []uvSeg {
	var out []uvSeg
	for li, l := range c.face.loops {
		for ei, e := range l.edges {
			var inject []float64
			for _, cr := range c.crossings {
				if cr.loop == li && cr.edge == ei {
					inject = append(inject, cr.tEdge)
				}
			}
			atSeam, atTube := seamHitParams(seamHits, func(cr frameCrossing) bool { return cr.loop == li && cr.edge == ei })
			out = append(out, c.sampledPolyline(e.curve, e.t0, e.t1, inject, atSeam, atTube, segPolygon)...)
		}
	}
	return out
}

// seamHitParams collects the parameters of the seam incidences a predicate selects, the azimuth seam's
// and the tube seam's apart.
func seamHitParams(hits []frameCrossing, on func(frameCrossing) bool) (atSeam, atTube []float64) {
	for _, cr := range hits {
		switch {
		case !on(cr):
		case cr.tube:
			atTube = append(atTube, cr.tEdge)
		default:
			atSeam = append(atSeam, cr.tEdge)
		}
	}
	return atSeam, atTube
}

// imprintSegments samples every imprint curve over its domain with its frame and seam incidences injected.
func (c *loopFrame) imprintSegments(imprint []geom.Curve3, seamHits []frameCrossing) []uvSeg {
	var out []uvSeg
	meets := c.solveImprintCrossings(imprint)
	for ii, imp := range imprint {
		inject := append([]float64(nil), meets[ii]...)
		for _, cr := range c.crossings {
			if cr.imp == ii {
				inject = append(inject, cr.tImp)
			}
		}
		atSeam, atTube := seamHitParams(seamHits, func(cr frameCrossing) bool { return cr.loop == seamIncidence && cr.edge == ii })
		lo, hi := imp.Domain()
		out = append(out, c.sampledPolyline(imp, lo, hi, inject, atSeam, atTube, segImprint)...)
	}
	return out
}

// sampledPolyline samples one curve from t0 to t1 on the curve's own grid with the given parameters
// injected, unwrapping the azimuth along the walk so each segment is continuous, snapping every seam
// incidence to the seam exactly, then folding each segment into the [0, 2π] strip. Any segment that
// still straddles the seam (an incidence the solver did not see) is split there by interpolation.
func (c *loopFrame) sampledPolyline(cv geom.Curve3, t0, t1 float64, inject, atSeam, atTube []float64, kind segKind) []uvSeg {
	params := injectedParams(cv, t0, t1, append(append(append([]float64{}, inject...), atSeam...), atTube...))
	pts, poles := c.sampleChartPoints(cv, params, atSeam, atTube)
	var out []uvSeg
	for i := 1; i < len(params); i++ {
		a, b := anchorPoleEnds(pts[i-1], pts[i], poles[i-1], poles[i])
		seg := foldIntoStrip(uvSeg{a: a, b: b, curve: cv, tA: params[i-1], tB: params[i], kind: kind})
		for _, s := range splitSeamCrossing(seg) {
			for _, s := range c.splitVSeamIfClosed(s) {
				if s.a.DistanceTo(s.b) > arrTol {
					out = append(out, s)
				}
			}
		}
	}
	return out
}

// sampleChartPoints maps a curve's sampled parameters into the chart, unwrapping the azimuth along the
// walk so each step is continuous, snapping every seam incidence to the seam exactly. A sample that
// lands on a PARAMETRIC POLE is flagged and left out of the unwrapping: u names no direction there, so
// it may neither take a branch from its predecessor nor hand one to its successor.
func (c *loopFrame) sampleChartPoints(cv geom.Curve3, params []float64, atSeam, atTube []float64) ([]math.Point2, []bool) {
	pts, poles := make([]math.Point2, len(params)), make([]bool, len(params))
	prevU, havePrev := 0.0, false
	for i, t := range params {
		p := cv.PointAt(t)
		uv := c.host.paramOf(p)
		u, v := float64(uv.X), float64(uv.Y)
		if poles[i] = c.atPole(p); poles[i] {
			pts[i] = uv
			continue
		}
		if havePrev {
			u = unwrapAzimuthNear(prevU, u)
		}
		if containsParam(atSeam, t) {
			u = 2 * stdmath.Pi * stdmath.Round(u/(2*stdmath.Pi))
		}
		if containsParam(atTube, t) {
			v = 2 * stdmath.Pi * stdmath.Round(v/(2*stdmath.Pi))
		}
		pts[i], prevU, havePrev = math.P2(u, v), u, true
	}
	return pts, poles
}

// atPole reports whether a point is a SINGULAR point of the chart's surface — a sphere's pole, a cone's
// apex — where the azimuth names no direction. The surface's own derivatives answer it scale-free, the
// same test the trim ring's branch repair reads (Oblikovati/Oblikovati#3447).
func (c *loopFrame) atPole(p math.Point3) bool {
	u, v := c.face.surface.ParamAt(p)
	return sampleOnPole(c.face.surface, math.P2(u, v), false)
}

// anchorPoleEnds gives a segment end that sits on a parametric pole the OTHER end's azimuth. At a pole
// the surface collapses to a single point, so every azimuth names it and continuity may not pick one:
// carried across the pole by continuity, the two samples flanking it become ONE segment leaping a
// half-turn in u at v = ±π/2. That leap crosses the seam, and the boundary walk then follows it out
// along the seam instead of stopping at the pole — the two open edges of the looped split
// (Oblikovati/Oblikovati#1334, ADR-0061). Anchored, each side reaches the pole ON ITS OWN meridian and
// the chart's pole segment bridges the two, which is what the pole segment is for. The 3-D geometry is
// untouched: point3(u, ±π/2) is the same pole for every u.
func anchorPoleEnds(a, b math.Point2, aPole, bPole bool) (math.Point2, math.Point2) {
	if aPole && !bPole {
		return math.P2(float64(b.X), float64(a.Y)), b
	}
	if bPole && !aPole {
		return a, math.P2(float64(a.X), float64(b.Y))
	}
	return a, b
}

// splitVSeamIfClosed splits a segment straddling the v-seam, on a chart whose v is a period.
func (c *loopFrame) splitVSeamIfClosed(s uvSeg) []uvSeg {
	if !c.host.vClosed() {
		return []uvSeg{s}
	}
	return splitVSeamCrossing(s)
}

// injectedParams is the curve's own sampling grid over [t0, t1] with the extra parameters merged, in
// traversal order (reversed when t0 > t1).
func injectedParams(cv geom.Curve3, t0, t1 float64, extra []float64) []float64 {
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	params := curveOwnParams(cv, lo, hi)
	for _, t := range extra {
		if t > lo && t < hi {
			params = append(params, t)
		}
	}
	params = preferInjected(sortedUniqueParams(params), extra)
	if t0 > t1 {
		for i, j := 0, len(params)-1; i < j; i, j = i+1, j-1 {
			params[i], params[j] = params[j], params[i]
		}
	}
	return params
}

// preferInjected replaces each kept station that an injected incidence was merged into by that
// incidence's own parameter. sortedUniqueParams keeps the first of two parameters closer than its
// gap, which is the sampling station when a seam lands exactly on one — a seam placed at π/2 on a rim
// sampled at quarter turns — and the station's value no longer matched the seam mark, so the point
// was not carried onto the seam and the fold broke. The incidence is the decided fact; the station
// is only where the curve happened to be sampled (ADR-0061 stage 4).
func preferInjected(kept, injected []float64) []float64 {
	for _, t := range injected {
		for i, k := range kept {
			if k != t && stdmath.Abs(k-t) <= sortedUniqueGap {
				kept[i] = t
			}
		}
	}
	return kept
}

// containsParam reports whether t is one of the listed parameters.
func containsParam(ts []float64, t float64) bool {
	for _, x := range ts {
		if x == t {
			return true
		}
	}
	return false
}

// foldIntoStrip shifts an unwrapped segment by whole turns so it lies in the [0, 2π] strip; a segment
// that ends exactly on the seam keeps that end at 0 or 2π.
func foldIntoStrip(s uvSeg) uvSeg {
	twoPi := 2 * stdmath.Pi
	lo := stdmath.Min(float64(s.a.X), float64(s.b.X))
	shift := twoPi * stdmath.Floor(lo/twoPi)
	if stdmath.Max(float64(s.a.X), float64(s.b.X))-shift > twoPi {
		shift = twoPi * stdmath.Floor(stdmath.Max(float64(s.a.X), float64(s.b.X))/twoPi-1)
	}
	s.a = math.P2(float64(s.a.X)-shift, float64(s.a.Y))
	s.b = math.P2(float64(s.b.X)-shift, float64(s.b.Y))
	return s
}

// seamSegments emits the two seam verticals (u=0 and u=2π) over the frame's axial window, split at
// every frame and imprint incidence so the seam shares those vertices. The overrun past the window is
// artificial and bounds no real geometry.
func (c *loopFrame) seamSegments(seamHits []frameCrossing) []uvSeg {
	vMin, vMax := c.host.vWindow()
	pad := c.host.seamOverrun()
	vs := []float64{vMin - pad, vMax + pad}
	for _, cr := range seamHits {
		if cr.tube {
			continue // the tube seam's own incidences split the tube seam, not this one
		}
		var cv geom.Curve3
		if cr.loop == seamIncidence {
			cv = c.crossingImprint(cr)
		} else {
			cv = c.face.loops[cr.loop].edges[cr.edge].curve
		}
		vs = append(vs, float64(c.host.paramOf(cv.PointAt(cr.tEdge)).Y))
	}
	sort.Float64s(vs)
	seam := geom.NewLineSegment(c.host.point3(0, vs[0]), c.host.point3(0, vs[len(vs)-1]))
	var out []uvSeg
	for i := 1; i < len(vs); i++ {
		if vs[i]-vs[i-1] <= arrTol {
			continue
		}
		for _, u := range []float64{0, 2 * stdmath.Pi} {
			out = append(out, uvSeg{a: math.P2(u, vs[i-1]), b: math.P2(u, vs[i]), curve: seam, tA: 0, tB: 1, kind: segSeam})
		}
	}
	return out
}

// crossingImprint returns the imprint curve a seam incidence refers to; assembleSegments stores the
// imprint on the chart for this lookup.
func (c *loopFrame) crossingImprint(cr frameCrossing) geom.Curve3 { return c.imprint[cr.edge] }

// loopTurnsTheAzimuth reports whether a boundary loop makes a full turn around the chart's periodic
// direction — a band's rim, a sphere's parallel — rather than closing on itself. It is the NET turn,
// not the raw span: a loop that runs out along a boundary and back covers the same longitudes twice
// without turning at all, and reading the span would call it a wrapping end (ADR-0060).
func (c *loopFrame) loopTurnsTheAzimuth(e emittedLoop) bool { return c.loopTurns(e, true) }

// loopTurns is loopTurnsTheAzimuth for either chart direction: inU selects u (the azimuth/longitude),
// otherwise v (the tube angle).
func (c *loopFrame) loopTurns(e emittedLoop, inU bool) bool {
	return stdmath.Abs(netTurn(e.face, func(p math.Point3) float64 {
		uv := c.host.paramOf(p)
		if inU {
			return float64(uv.X)
		}
		return float64(uv.Y)
	})) > stdmath.Pi
}

// netTurn is the NET turn of an angular coordinate along a chain of edges: each edge is walked finely
// enough that consecutive samples stay within half a turn, so the unwrap cannot mistake the direction,
// and the increments are summed. It is the NET, so a chain that runs out along a boundary and back
// covers the same angles twice and turns by nothing.
func netTurn(chain []loopEdge, coord func(math.Point3) float64) float64 {
	turn, prev := 0.0, 0.0
	first := true
	for _, le := range chain {
		for k := 0; k <= azimuthTurnSamples; k++ {
			x := coord(le.curve.PointAt(le.t0 + (le.t1-le.t0)*float64(k)/azimuthTurnSamples))
			if !first {
				x = unwrapAzimuthNear(prev, x)
				turn += x - prev
			}
			prev, first = x, false
		}
	}
	return turn
}

// loopTurnsAPeriod reports whether a boundary loop turns the whole way round ANY of the chart's own
// periodic directions. A torus is periodic in both: the band a plane parallel to its axis leaves is
// bounded by two spiric ovals, and each of those turns the TUBE angle, not the azimuth — reading only
// the azimuth called them contractible, and the contractible emission then walked each oval out and
// back as a dangling chain, so a loop that should have re-emitted as the whole closed section came out
// as a zero-length edge (ADR-0062). The chart names its own periods; nothing here knows a torus from a
// sphere.
func loopTurnsAPeriod(c *loopFrame, side uvSide, e emittedLoop) bool {
	return (side.uPeriodic() && c.loopTurns(e, true)) || (side.vPeriodic() && c.loopTurns(e, false))
}

// azimuthTurnSamples walks each edge finely enough that consecutive samples stay within half a turn, so
// the unwrap cannot mistake the direction.
const azimuthTurnSamples = 16

// wrappingComponents emits every kept component whose boundary WRAPS the chart's periodic direction, as
// one face per component carrying all its loops. A band's two rims, a sphere cap's section circle above
// its pole, a torus band's two section circles: each is a loop that turns the whole way round, which
// the contractible emission cannot file because it groups loops by (u,v) containment and a wrapping
// loop contains nothing. It dropped them and kept whatever else the component had — a pole, or one of
// two rims (ADR-0061 stage 3).
//
// ok=false when a component wraps nowhere: that one IS contractible and the caller's own path takes it.
func (c *loopFrame) wrappingComponents(side uvSide, kept []Face2D, segs []uvSeg, surface geom.Surface, f curvedFace) ([]curvedFace, []loopEdge, bool) {
	var faces []curvedFace
	var lid []loopEdge
	for _, comp := range keptComponents(kept, side.uPeriodic(), side.vPeriodic()) {
		// The artificial loops go first: a boundary that follows only the seams bounds nothing, and a
		// chart that refuses to re-emit a seam run would decline the whole component over one.
		loops := dropArtificialLoops(chainLoops(keptBoundaryEdges(comp, side.uPeriodic(), side.vPeriodic())), segs)
		emitted, ok := emitKeptLoops(side, loops, segs)
		if !ok || !anyLoopWraps(c, side, emitted) {
			return nil, nil, false
		}
		faceLoops := make([]curvedLoop, 0, len(emitted))
		for _, e := range periodTurningFirst(c, side, emitted) {
			faceLoops = append(faceLoops, curvedLoop{edges: e.face})
			lid = append(lid, reverseEdgeChain(e.section)...)
		}
		faces = append(faces, curvedFace{surface: surface, reversed: f.reversed, lineage: f.lineage,
			loops: side.finalizeLoops(faceLoops),
			chart: chartContours(chartOfKept(comp), side.seamOrigin())})
	}
	return faces, lid, len(faces) > 0
}

// periodTurningFirst puts the loops that TURN a period ahead of the ones that do not, so loops[0] is an
// end of the band rather than a hole in it. The order is otherwise stable, so a band with two wrapping
// ends and no holes is untouched.
//
// A component's loops arrive in trace order, and loops[0] is taken as the face's OUTER boundary. On a
// closed surface that can invert the face: a sphere poking out of a box through three of its faces
// leaves an ANNULUS on the sphere — an outer three-arc loop that wraps, and an inner three-arc loop
// around the box's corner that does not — and traced inner-first the face read as "the little corner
// patch minus everything else", a valid solid whose volume came back as the box exactly (ADR-0061
// stage 4).
func periodTurningFirst(c *loopFrame, side uvSide, emitted []emittedLoop) []emittedLoop {
	out := make([]emittedLoop, 0, len(emitted))
	for _, e := range emitted {
		if loopTurnsAPeriod(c, side, e) {
			out = append(out, e)
		}
	}
	for _, e := range emitted {
		if !loopTurnsAPeriod(c, side, e) {
			out = append(out, e)
		}
	}
	return out
}

// anyLoopWraps reports whether some emitted loop turns one of the chart's periodic directions the whole way round.
func anyLoopWraps(c *loopFrame, side uvSide, emitted []emittedLoop) bool {
	for _, e := range emitted {
		if loopTurnsAPeriod(c, side, e) {
			return true
		}
	}
	return false
}

// bisectRoot refines a sign-change bracket [lo, hi] of g to a model-independent angular tolerance by
// bisection — unconditionally convergent, no derivative needed (the corner equation is smooth but its
// derivative adds no robustness a 60-step bisection to ~1e-16 rad lacks).
func bisectRoot(g func(float64) float64, lo, hi float64) float64 {
	glo := g(lo)
	for range 60 {
		mid := (lo + hi) / 2
		gm := g(mid)
		if gm == 0 || (hi-lo) < 1e-15 { // tol:angular — the bracket is an ANGLE in radians, model-scale free
			return mid
		}
		if glo*gm < 0 {
			hi = mid
		} else {
			lo, glo = mid, gm
		}
	}
	return (lo + hi) / 2
}
