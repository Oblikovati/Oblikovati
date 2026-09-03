// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Phase 5 of the ruled-side (u,v) arrangement (#2212): turn traced loops back into analytic
// face loops.
//
// Each run of traced edges that came from one imprint curve is re-emitted as that curve, so the
// result carries exact arcs and conics rather than the sampled polyline phase 1 built. Loops are
// grouped into faces by containment (an inner loop is a hole in the smallest face that contains
// it) and oriented, and a degenerate loop collapsed onto a cone apex is dropped.

// dropApexLoop removes a degenerate apex-rim loop from a kept cone face. When the band's bottom rim is the
// cone apex (rBot=0), an apex-kept cut emits that rim as a zero-radius circle at the apex point; the apex is
// an interior POLE of the closed face, not a boundary, so the loop (carrying no section arcs, so no lid) is
// dropped — leaving the single cut loop the analytic apexCapSide produced (#1405).
func (c ruledUV) dropApexLoop(loops []curvedLoop) []curvedLoop {
	if c.band.rBot > 1e-9 {
		return loops
	}
	out := make([]curvedLoop, 0, len(loops))
	for _, lp := range loops {
		if !c.loopAtApex(lp) {
			out = append(out, lp)
		}
	}
	return out
}

// loopAtApex reports whether every edge of a loop sits at the cone apex (the degenerate v=0 rim point).
func (c ruledUV) loopAtApex(lp curvedLoop) bool {
	for _, e := range lp.edges {
		if float64(e.start().DistanceTo(c.band.bottom)) > 1e-6 {
			return false
		}
	}
	return len(lp.edges) > 0
}

// emittedLoop is one re-emitted boundary loop awaiting orientation: its full edge chain, the imprint
// (section) sub-arcs alone (for the lid), its mean axial level (to order hi boundary first), and the signed
// (u,v) area of the dedge loop it came from. keptBoundaryEdges orients edges so the kept material is on the
// left, so a CCW (positive area) boundary encloses kept material (an outer loop) and a CW (negative area)
// one bounds a dropped island (a hole) — the signal a closed-surface trim (torus) uses to tell the small
// cap (kept inside, CCW) from its genus-1 complement (kept outside, the oval a CW hole) (#1406).
type emittedLoop struct {
	face    []loopEdge
	section []loopEdge
	mv      float64
	area    float64
}

// emitKeptLoops re-emits every (u,v) boundary loop to analytic edges, sorted with the higher (hi) boundary
// first to match the analytic split convention (loops[0] is the hi boundary / outer loop).
func emitKeptLoops(c uvSide, loops [][]dedge, segs []uvSeg) ([]emittedLoop, bool) {
	ix := newUVSegIndex(segs)
	out := make([]emittedLoop, 0, len(loops))
	for _, lp := range loops {
		face, section, ok := emitLoopEdges(c, lp, ix)
		if !ok {
			return nil, false
		}
		out = append(out, emittedLoop{face: face, section: section, mv: meanEdgeV(c, face), area: dedgeLoopArea(lp)})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].mv > out[j].mv })
	return out, true
}

// dedgeLoopArea returns the signed (u,v) area of a boundary loop by the shoelace formula over its dedge
// start points. Positive is counter-clockwise. Meaningful only for a loop that does not wrap a seam (a
// contractible imprint, e.g. a torus oval); a seam-wrapping band's area is not used.
func dedgeLoopArea(loop []dedge) float64 {
	sum := 0.0
	for _, e := range loop {
		sum += float64(e.a.X)*float64(e.b.Y) - float64(e.b.X)*float64(e.a.Y)
	}
	return sum / 2
}

// groupLoopFaces groups the kept region's boundary loops into connected FACES (#1403). A WRAPPING band or a
// single patch stays one face — all loops together, the half-space convention — so the plane-cut path is
// unchanged. A non-wrapping region with several loops is grouped by (u,v) containment: each outer loop
// (CCW, area>0) is a face; each hole (CW) joins the smallest outer that contains it; DISJOINT outer loops
// (the two lens caps a rod punches in a fat cone's wall) become SEPARATE faces — the generalization a
// curved∩curved cut needs over a plane cut, where the kept region is always one connected piece.
func groupLoopFaces(multiFace, wrapping bool, loops [][]dedge) [][][]dedge {
	if !multiFace || wrapping || len(loops) <= 1 {
		return [][][]dedge{loops} // half-space/torus, or a wrapping band, or a single patch — one face
	}
	var groups []*faceGroup
	var holes [][]dedge
	for _, lp := range loops {
		if dedgeLoopArea(lp) >= 0 {
			groups = append(groups, &faceGroup{loops: [][]dedge{lp}, area: dedgeLoopArea(lp), outer: lp})
		} else {
			holes = append(holes, lp)
		}
	}
	if len(groups) == 0 { // all holes (a closed-surface complement) — keep as one face (outerless)
		return [][][]dedge{loops}
	}
	for _, h := range holes {
		if g := smallestContainingFace(groups, loopPointInside(h)); g != nil {
			g.loops = append(g.loops, h)
		}
	}
	out := make([][][]dedge, len(groups))
	for i, g := range groups {
		out[i] = g.loops
	}
	return out
}

// faceGroup accumulates one connected face's boundary loops (its outer plus any holes that nest in it)
// during groupLoopFaces.
type faceGroup struct {
	outer []dedge
	loops [][]dedge
	area  float64
}

// smallestContainingFace returns the face group whose outer loop contains p and has the smallest area (so a
// hole nested in nested outers attaches to its immediate parent), or nil when none contains it.
func smallestContainingFace(groups []*faceGroup, p math.Point2) *faceGroup {
	var best *faceGroup
	bestArea := stdmath.MaxFloat64
	for _, g := range groups {
		if g.area < bestArea && dedgeLoopContains(g.outer, p) {
			best, bestArea = g, g.area
		}
	}
	return best
}

// loopPointInside returns a representative interior point of a loop: the average of an edge's endpoint and
// the loop centroid, nudged inside. The first vertex sits ON the boundary, so a midpoint toward the centroid
// lands strictly inside for the convex-ish lens/annulus loops this groups.
func loopPointInside(loop []dedge) math.Point2 {
	var cx, cy float64
	for _, e := range loop {
		cx += float64(e.a.X)
		cy += float64(e.a.Y)
	}
	n := float64(len(loop))
	return math.P2(cx/n, cy/n)
}

// dedgeLoopContains reports whether p is inside the (u,v) polygon of loop, by even-odd ray casting along +u.
func dedgeLoopContains(loop []dedge, p math.Point2) bool {
	inside := false
	for _, e := range loop {
		ay, by := float64(e.a.Y), float64(e.b.Y)
		if (ay > float64(p.Y)) == (by > float64(p.Y)) {
			continue
		}
		x := float64(e.a.X) + (float64(p.Y)-ay)/(by-ay)*(float64(e.b.X)-float64(e.a.X))
		if float64(p.X) < x {
			inside = !inside
		}
	}
	return inside
}

// orientLoops applies the analytic splitSide orientation convention to the ordered loops: the lo boundaries
// and the default hi boundary are forward with their section arcs reversed into the lid; but in a WRAPPING
// band the hi loop is REVERSED when it carries a rim and the source side traversed its top rim reversed
// (band.topRimReversed), the lid then taking that loop's section arcs forward — so the rebuilt rim keeps the
// sense opposite its cap and the band, lid and caps stay a consistent manifold. The reversal is gated on
// wrapping because a non-wrapping tongue is one mixed loop the analytic tongueSide leaves un-reversed
// (#1405, mirroring curved_halfspace_ruled_uv.go's splitSide vs tongueSide).
func (c ruledUV) orientLoops(loops []emittedLoop, wrapping bool) ([]curvedLoop, []loopEdge, bool) {
	faceLoops := make([]curvedLoop, 0, len(loops))
	var lid []loopEdge
	for i, e := range loops {
		face, lidSec := e.face, reverseEdgeChain(e.section)
		if i == 0 && wrapping && c.band.topRimReversed && allRimEdges(e.face) {
			face, lidSec = reverseEdgeChain(e.face), e.section
		}
		faceLoops = append(faceLoops, curvedLoop{edges: face})
		lid = append(lid, lidSec...)
	}
	return faceLoops, lid, false // a ruled side is open in v: its kept face always has an outer loop
}

// allRimEdges reports whether every edge of a loop is a rim (circle/arc) — a PURE top-rim hi boundary, the
// only wrapping hi loop the topRimReversed whole-loop reversal applies to; a mixed section+rim hi boundary
// (a clips-rim annulus) keeps its arrangement orientation instead.
func allRimEdges(edges []loopEdge) bool {
	for _, e := range edges {
		switch e.curve.(type) {
		case geom.Circle, geom.Arc3d:
		default:
			return false
		}
	}
	return len(edges) > 0
}

// meanEdgeV is the mean axial level of an edge chain (the average v of its endpoints), used to order the
// hi boundary loop first.
func meanEdgeV(c uvSide, edges []loopEdge) float64 {
	if len(edges) == 0 {
		return 0
	}
	sum := 0.0
	for _, e := range edges {
		sum += float64(c.paramOf(e.start()).Y) + float64(c.paramOf(e.end()).Y)
	}
	return sum / float64(2*len(edges))
}

// rotateToTransition rotates the recovered loop so it begins at a curve boundary (where the kind/curve
// changes), so every run mergeRuns forms is ONE contiguous arc. Without it a loop that happens to start in
// the middle of a curve splits that curve into two runs around the loop seam — and a closed conic run that
// wraps re-emits as a spurious full curve. A loop on a single curve (no transition) is returned unchanged.
func rotateToTransition(rec []recoveredEdge) []recoveredEdge {
	n := len(rec)
	for i := range n {
		if !sameRun(rec[(i-1+n)%n], rec[i]) {
			return append(append([]recoveredEdge{}, rec[i:]...), rec[:i]...)
		}
	}
	return rec
}

// mergeRuns groups consecutive recovered edges that lie on the same analytic curve into runs, so a chain of
// many sampled sub-edges along one conic (or many sub-arcs of one rim) re-emits as a single loopEdge.
func mergeRuns(rec []recoveredEdge) [][]recoveredEdge {
	var runs [][]recoveredEdge
	for _, e := range rec {
		if n := len(runs); n > 0 && sameRun(runs[n-1][0], e) {
			runs[n-1] = append(runs[n-1], e)
		} else {
			runs = append(runs, []recoveredEdge{e})
		}
	}
	return runs
}

// emitRun re-emits one run of recovered edges (all on the same curve) as a single exact loopEdge.
func (c ruledUV) emitRun(run []recoveredEdge) (loopEdge, bool) {
	switch run[0].kind {
	case segRim:
		return c.emitRimRun(run)
	case segImprint:
		return emitImprintRun(run)
	default:
		return c.emitSeamRun(run)
	}
}

// emitRimRun re-emits a run along a band rim as one circular edge: the full rim circle when the run wraps
// the whole azimuth, else a geom.Arc3d over the run's azimuth span in the surface's own frame (so it lands
// exactly on the rim). The span sums per-edge azimuth deltas, which stays correct across the seam and for
// a single full-wrap edge (where the raw endpoints alone would read as a zero span).
func (c ruledUV) emitRimRun(run []recoveredEdge) (loopEdge, bool) {
	bottom := stdmath.Abs(float64(run[0].a.Y)-c.band.vMin) <= stdmath.Abs(float64(run[0].a.Y)-c.band.vMax)
	center, radius, circle := c.band.top, c.band.rTop, c.band.topCirc
	if bottom {
		center, radius, circle = c.band.bottom, c.band.rBot, c.band.bottomCirc
	}
	span := 0.0
	for _, e := range run {
		span += float64(e.b.X) - float64(e.a.X)
	}
	if stdmath.Abs(span) >= 2*stdmath.Pi-1e-6 {
		return loopEdge{curve: circle, t0: 0, t1: 1}, true // a full rim circle, reused whole
	}
	// NewArc3d's start angle is measured on the surface's own ref frame, so add seamU back to the
	// seam-relative u of the run's start.
	arc, err := geom.NewArc3d(center, c.axis, c.ref, radius, float64(run[0].a.X)+c.seamU, span)
	if err != nil {
		return loopEdge{}, false
	}
	return loopEdge{curve: arc, t0: 0, t1: 1}, true
}

// emitImprintRun re-emits a run along one imprint curve as a single loopEdge over the recovered parameter
// span. For a CLOSED conic (ellipse/circle) the parameters wrap, so the run's parameter sequence is
// unwrapped to a monotone span (handling the param seam, like the analytic sectionArm); a run that covers
// the whole closed curve re-emits it over its full domain. Open conic arms (hyperbola/parabola/line) carry
// monotone parameters already.
func emitImprintRun(run []recoveredEdge) (loopEdge, bool) {
	curve := run[0].curve
	t0 := run[0].tA
	tEnd := run[len(run)-1].tB
	if isClosedCurve(curve) {
		prev := t0
		for _, e := range run {
			prev = unwrapParamNear(prev, e.tB)
		}
		tEnd = prev
		if lo, hi := curve.Domain(); stdmath.Abs(tEnd-t0) >= (hi-lo)-1e-6 {
			// The whole closed curve, but KEEP the run's traversal sense: a boundary walked in the curve's
			// DECREASING-parameter direction re-emits as [hi, lo], not [lo, hi]. Discarding the sign made the
			// two walls of a general curved∩curved boolean emit a SHARED imprint loop identically, so the weld
			// saw the same orientation on both faces (curvedStitch orients a closed edge by t1<t0) and the
			// solid failed the orientation check — silently rejected by validBooleanSolid, falling back to the
			// bespoke handler. The two walls walk the shared loop in OPPOSITE senses, so preserving the sign
			// welds them consistently. A half-space cut keeps the forward sense it always had (its single kept
			// region walks the closed section forward), so those paths are unchanged (#1403).
			if tEnd < t0 {
				return loopEdge{curve: curve, t0: hi, t1: lo}, true
			}
			return loopEdge{curve: curve, t0: lo, t1: hi}, true
		}
	}
	return loopEdge{curve: curve, t0: t0, t1: tEnd}, true
}

// emitSeamRun re-emits a (rare, surviving) seam-ruling run as a straight edge between its 3D endpoints — a
// ruling of the side from the run's first to last (u,v) vertex.
func (c ruledUV) emitSeamRun(run []recoveredEdge) (loopEdge, bool) {
	p0 := c.point3(float64(run[0].a.X), float64(run[0].a.Y))
	p1 := c.point3(float64(run[len(run)-1].b.X), float64(run[len(run)-1].b.Y))
	return loopEdge{curve: geom.NewLineSegment(p0, p1), t0: 0, t1: 1}, true
}

// isClosedCurve reports whether a curve closes on itself, so its parameter wraps and a boundary run along it
// must be unwrapped to a monotone span before re-emission. The answer comes from geom.CurveIsClosed — a
// measurement of whether the curve's ends meet, against its own scale — because the type switch that used to
// live here classed every curve kind it did not list as open, and the exact ruled∩quadric section then
// skipped the closed-loop re-emission both walls rely on to weld (#3489). A full spiric band closes and a
// single-oval branch does not, exactly as before, but now because their ends do or do not meet (#1406).
func isClosedCurve(curve geom.Curve3) bool { return geom.CurveIsClosed(curve) }

// sameRun reports whether two recovered edges belong to one re-emittable run: the same kind, and for an
// imprint the same source curve, for a rim the same rim level (a rim sits at constant v, so equal v).
func sameRun(a, b recoveredEdge) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case segImprint, segPolygon:
		// A polygon edge, like an imprint arc, is a distinct analytic curve; equal v does NOT imply same run
		// (that is only true for a constant-v rim), so identity is the run test (#1591).
		return a.curve == b.curve
	case segRim:
		return stdmath.Abs(float64(a.a.Y)-float64(b.a.Y)) < 1e-6
	default:
		return true
	}
}
