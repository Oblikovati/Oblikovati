// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Mixed planar+curved boolean by PER-FACE dispatch (ADR-0058). With the face model and the stitch
// unified, an operand no longer has to be all-planar: its straight-edged planar faces run the exact
// planar imprint→split→classify pipeline, while every other face (a curved wall, or a planar face
// bounded by a curved edge — a boss seat's circular hole) PASSES THROUGH whole, classified as a unit
// and welded by the same unified stitch. Scope is conservative and declines loudly (ErrUnsupportedMixedBoolean →
// the caller's curved/CSG fallbacks): every pass-through face must be box-disjoint from EVERY face of
// the other operand, so no imprint can touch it, its membership in the other solid is uniform, and no
// T-junction can appear on its boundary. A kept Difference tool face (material inside the target) is
// reversed whole into the cavity — the embedded-void cut (a block minus an interior cylinder) comes
// out exact. Fragment classification against a mixed body uses the general analytic
// point-in-solid classifier (the frustum fast paths, else ray parity — classify_point.go); an
// all-planar operand keeps the winding-number solidProbe bit-for-bit.

// insideOracle is a body's cached point-membership test for fragment classification: the planar
// winding-number solidProbe, or the mixed analytic probe.
type insideOracle interface {
	inside(p math.Point3) bool
	// onPlaneStep is the offset that clears this solid's own on-plane band, for the two-sided probe
	// that resolves a point lying in a plane of its boundary (boolean_classify_coplanar.go). false
	// when the oracle carries no geometry to derive one from.
	onPlaneStep() (float64, bool)
}

// facePartition splits a body's flattened faces into the polygonal-planar pipeline set (plane
// surface AND all-straight edges) and the pass-through set (everything else), the latter with each
// face's true topo bounding box (loop-point boxes underestimate curved faces — a rim circle's
// loop edge collapses to its seam point).
type facePartition struct {
	planar      []curvedFace
	planarFull  []curvedFace   // per planar face: the UNstripped face (true trim, for exact imprint clipping)
	planarHoles [][]curvedLoop // per planar face: curved hole loops detached before the polygonal split
	uv          []curvedFace   // curved-OUTER planar faces the exact-frame chart can split (planeFaceUV)
	uvBox       []math.Box
	wall        []curvedFace // full-band cylinder walls the ruled chart can split (cylinderSideSolidSplit)
	wallBox     []math.Box
	pass        []curvedFace
	passBox     []math.Box
	body        *topo.Body
}

// partitionFaces flattens b and buckets each face for per-face dispatch. A planar face whose OUTER
// loop is straight but which carries CURVED hole loops (a boss seat's rim circle, a drilled plate's
// bore hole) is still splittable: the curved holes are DETACHED for the polygonal split and
// re-attached exactly to the containing fragment afterwards. This is sound because every curved hole
// loop borders a pass-through face of the SAME body (the bore/boss wall — a polygonal face cannot
// carry the curved edge), and passDisjointFrom already keeps the whole tool a pad away from that
// wall's box, which contains the hole: no imprint, membership boundary, or fragment probe can come
// near a detached hole.
func partitionFaces(b *topo.Body) facePartition {
	p := facePartition{body: b}
	topoFaces := b.Faces()
	for i, cf := range facesOfAny(b) {
		if stripped, holes, ok := detachCurvedHoles(cf); ok {
			p.planar = append(p.planar, stripped)
			p.planarFull = append(p.planarFull, cf)
			p.planarHoles = append(p.planarHoles, holes)
			continue
		}
		if _, ok := newPlaneFaceUV(cf, geom.ResolutionForBox(topoFaces[i].RangeBox())); ok {
			p.uv = append(p.uv, cf)
			p.uvBox = append(p.uvBox, topoFaces[i].RangeBox())
			continue
		}
		if _, ok := ruledFaceOf(cf); ok {
			p.wall = append(p.wall, cf)
			p.wallBox = append(p.wallBox, topoFaces[i].RangeBox())
			continue
		}
		p.pass = append(p.pass, cf)
		p.passBox = append(p.passBox, topoFaces[i].RangeBox())
	}
	return p
}

// detachCurvedHoles classifies a face for the polygonal split: a plane surface with an all-straight
// outer loop qualifies; straight inner loops stay in the split, curved inner loops are detached (to
// re-attach exactly). ok=false sends the face to the pass-through bucket (curved surface, or a curved
// edge on the outer loop).
func detachCurvedHoles(f curvedFace) (curvedFace, []curvedLoop, bool) {
	if _, isPlane := f.surface.(geom.Plane); !isPlane || len(f.loops) == 0 || !straightLoop(f.loops[0]) {
		return curvedFace{}, nil, false
	}
	stripped := f
	stripped.loops = []curvedLoop{f.loops[0]}
	var detached []curvedLoop
	for _, l := range f.loops[1:] {
		if straightLoop(l) {
			stripped.loops = append(stripped.loops, l)
		} else {
			detached = append(detached, l)
		}
	}
	return stripped, detached, true
}

// straightLoop reports whether every edge of the loop is a straight segment.
func straightLoop(l curvedLoop) bool {
	for _, e := range l.edges {
		switch e.curve.(type) {
		case geom.LineSegment, geom.Line:
		default:
			return false
		}
	}
	return true
}

// mixedProbe is the point-in-solid oracle for a body with pass-through faces: the conditioning-gated
// frustum closed form when the body is a simple primitive, else the orientation-independent ray-parity
// classifier with the winding fallback (classify_point.go — the OCCT BRepClass3d analogue).
type mixedProbe struct {
	faces []curvedFace
	box   math.Box
	fast  func(math.Point3) bool
}

// newInsideOracle builds b's membership oracle: the exact winding-number solidProbe for an all-planar
// body (bit-for-bit the pure-planar pipeline's verdicts), else the mixed analytic probe over the
// already-flattened faces.
func newInsideOracle(b *topo.Body, faces []curvedFace) insideOracle {
	if sp := newSolidProbe(b); sp.planar {
		return sp
	}
	mp := &mixedProbe{faces: faces, box: b.RangeBox()}
	mp.fast, _ = primitiveSolidInside(faces)
	return mp
}

func (mp *mixedProbe) onPlaneStep() (float64, bool) {
	return offPlaneProbeSteps * geom.ResolutionForBox(mp.box).Plane(), true
}

func (mp *mixedProbe) inside(p math.Point3) bool {
	if mp.fast != nil {
		return mp.fast(p)
	}
	if in, ok := rayParityInsideClean(mp.faces, p, mp.box); ok {
		return in
	}
	return newFluxQuery(mp.faces).windingInside(p)
}

// allFaces is the partition's full flattened face list (planar then pass) for the membership oracle —
// the TRUE faces (planarFull), never the stripped ones: a detached hole is a real boundary the
// classifier's rays must see, or every ray through the hole region counts a phantom crossing.
func (p facePartition) allFaces() []curvedFace {
	all := append(append([]curvedFace{}, p.planarFull...), p.uv...)
	return append(append(all, p.wall...), p.pass...)
}

// passThroughKept classifies each pass-through face as a whole — its membership in the other solid is
// uniform (box-disjoint from the other's boundary) — keeping or dropping it by the boolean's keep
// table. A kept Difference tool face (material inside the target) is REVERSED into the cavity, the
// same sense flip the planar classify applies to its fragments (reverseCurvedFaces). ok=false
// declines the whole boolean: a face with no sample point (a boundaryless sphere/torus).
func passThroughKept(pass []curvedFace, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	var out []curvedFace
	for _, f := range pass {
		p, ok := passSamplePoint(f)
		if !ok {
			return nil, false
		}
		if !keep(op, isB, other.inside(p)) {
			continue
		}
		if op == Difference && isB {
			f = reverseCurvedFaces([]curvedFace{f})[0] // a kept tool face bounds the cavity
		}
		out = append(out, f)
	}
	return out, true
}

// passSamplePoint is a point of the face for its uniform membership test: a point strictly INSIDE its
// trim — the in-trim (u,v) grid point farthest from its boundary — never a loop vertex, which can sit
// on the other operand's boundary where membership is undefined (a boss's rim on the underside of
// the plate it stands on, ADR-0060). A face the chart cannot sample falls back to a loop-edge start;
// a boundaryless face (a full sphere) has none and declines.
func passSamplePoint(f curvedFace) (math.Point3, bool) {
	region := faceTrimRegion(f)
	if u0, u1, v0, v1, ok := fluxDomain(f, region); ok {
		ff := fluxFace{cf: f, region: region, u0: u0, u1: u1, v0: v0, v1: v1, sign: 1}
		if p, _, ok := faceProbePoint(&ff); ok {
			return p, true
		}
	}
	for _, l := range f.loops {
		for _, e := range l.edges {
			return e.start(), true
		}
	}
	return math.Point3{}, false
}

// booleanMixed is booleanOnce's per-face-dispatch counterpart for operands with pass-through faces.
// It returns ErrUnsupportedMixedBoolean whenever the conservative scope gate declines, so the caller falls to the
// curved/CSG paths exactly as before.
func booleanMixed(op Op, a, b *topo.Body) (*topo.Body, bool, error) {
	pa, pb := partitionFaces(a), partitionFaces(b)
	if !passClearOf(pa, pb) || !passClearOf(pb, pa) {
		return nil, false, ErrUnsupportedMixedBoolean
	}
	// A planar face a wall sections with a CONIC cannot receive that imprint in the polygonal bucket (whose
	// currency is [][2]math.Point3); move it to the exact-frame bucket, which carries geom.Curve3 imprints
	// (#3460). This runs BEFORE crossingFaceCandidates so every index derived from the partitions — the
	// candidate pairs, the imprint lists, the fragment selection — is computed from the promoted buckets.
	promoteConicReceivers(&pa, &pb)
	promoteConicReceivers(&pb, &pa)
	promoteCoplanarReceivers(&pa, &pb)
	promoteCoplanarReceivers(&pb, &pa)
	pra, prb := newInsideOracle(a, pa.allFaces()), newInsideOracle(b, pb.allFaces())
	pairs := crossingFaceCandidates(pa.planar, pb.planar)
	// Imprints clip against the TRUE trims (planarFull): a face's detached holes are void, so no
	// imprint is minted inside them (the phantom that broke the detached-hole premise). The exact-frame
	// (uv) faces' imprints run BEFORE the polygonal split, mirroring the same segments onto the other
	// side's imprint lists so the two faces split on identical coordinates.
	impA, impB, prov := imprintCandidates(pa.planarFull, pb.planarFull, pairs)
	uvImpA, uvImpB, wallImpA, wallImpB, okI := mixedCurvedImprints(&pa, &pb, impA, impB)
	if !okI {
		return nil, false, ErrUnsupportedMixedBoolean
	}
	kept, demoted, okK := mixedKeptFragments(pa, pb, impA, impB, pra, prb, pairs, op, prov)
	pass, okP := mixedPassFaces(pa, pb, pra, prb, uvImpA, uvImpB, op)
	pass = append(pass, demoted...)
	walls, okQ := mixedWallFaces(pa, pb, pra, prb, wallImpA, wallImpB, op)
	if !okK || !okP || !okQ {
		return nil, false, ErrUnsupportedMixedBoolean
	}
	return stitch(kept, append(pass, walls...), prov)
}

// mixedKeptFragments runs both operands' polygonal splits (with detached-hole re-attachment).
func mixedKeptFragments(pa, pb facePartition, impA, impB [][][2]math.Point3, pra, prb insideOracle, pairs facePairs, op Op, prov []imprintSeg) ([]subFace, []curvedFace, bool) {
	keptA, demotedA, okA := selectFacesDetached(pa, impA, prb, pb.planar, pairs.bForA, op, false, prov, pb.allFaces())
	keptB, demotedB, okB := selectFacesDetached(pb, impB, pra, pa.planar, pairs.aForB, op, true, prov, pa.allFaces())
	return append(append([]subFace{}, keptA...), keptB...), append(demotedA, demotedB...), okA && okB
}

// mixedCurvedImprints plans both operands' exact-frame and wall imprints in one pass, then pairs the
// exact-frame faces against the OTHER operand's ruled walls: that pairing writes the same section curve
// into both the uv face's and the wall's list, so the two sides split on identical coordinates (#3460).
func mixedCurvedImprints(pa, pb *facePartition, impA, impB [][][2]math.Point3) (uvA, uvB, wallA, wallB [][]geom.Curve3, ok bool) {
	uvA, uvB, okU := bothUVImprints(pa, pb, impA, impB)
	wallA, okWA := wallImprints(pa, pb, impB)
	wallB, okWB := wallImprints(pb, pa, impA)
	if !okU || !okWA || !okWB {
		return nil, nil, nil, nil, false
	}
	okXA := pairUVWallImprints(pa, pb, uvA, wallB)
	okXB := pairUVWallImprints(pb, pa, uvB, wallA)
	okXX := pairUVUVImprints(pa, pb, uvA, uvB)
	return uvA, uvB, wallA, wallB, okXA && okXB && okXX
}

// mixedWallFaces trims both operands' walls into the stitch's pass list.
func mixedWallFaces(pa, pb facePartition, pra, prb insideOracle, wallImpA, wallImpB [][]geom.Curve3, op Op) ([]curvedFace, bool) {
	wallA, okA := wallSplitFaces(pa, wallImpA, prb, op, false)
	wallB, okB := wallSplitFaces(pb, wallImpB, pra, op, true)
	return append(wallA, wallB...), okA && okB
}

// mixedPassFaces assembles the stitch's pass list: both sides' whole pass-through faces plus the
// exact-frame (uv) trims, all classified/reversed by the boolean's keep table.
func mixedPassFaces(pa, pb facePartition, pra, prb insideOracle, uvImpA, uvImpB [][]geom.Curve3, op Op) ([]curvedFace, bool) {
	passA, okA := passThroughKept(pa.pass, prb, op, false)
	passB, okB := passThroughKept(pb.pass, pra, op, true)
	uvA, okVA := uvSplitFaces(pa, uvImpA, prb, pb.allFaces(), op, false)
	uvB, okVB := uvSplitFaces(pb, uvImpB, pra, pa.allFaces(), op, true)
	if !okA || !okB || !okVA || !okVB {
		return nil, false
	}
	return append(append(append(passA, passB...), uvA...), uvB...), true
}

// selectFacesDetached is selectFaces plus the exact-hole re-attachment: after a face's fragments are
// selected, each of its detached curved holes is attached to the fragment containing it. ok=false
// (decline) when a hole's containing fragment cannot be identified.
//
// A face whose detached hole the imprint MEETS — the tool crosses the hole's rim — is not the
// polygonal split's case at all: the hole is invisible to that arrangement. Such a face is DEMOTED to
// the exact-frame chart with its full loops, where the hole's circle is a frame edge and every
// crossing with it is solved in closed form (ADR-0060); its trims come back as curvedFaces.
func selectFacesDetached(p facePartition, imprints [][][2]math.Point3, other insideOracle, others []curvedFace, otherCand [][]int, op Op, isB bool, prov []imprintSeg, allOthers []curvedFace) ([]subFace, []curvedFace, bool) {
	var kept []subFace
	var demoted []curvedFace
	for i, f := range p.planar {
		detached := p.planarHoles[i]
		if len(detached) > 0 && imprintMeetsHole(imprints[i], detached, facePlane(f)) {
			trims, ok := demotedHoleFaceTrims(p.planarFull[i], imprints[i], other, allOthers, op, isB)
			if !ok {
				return nil, nil, false
			}
			demoted = append(demoted, trims...)
			continue
		}
		fromFace := selectFragments(f, imprints[i], other, facesAt(others, otherCand[i]), op, isB, prov)
		if len(detached) > 0 && !attachExactHoles(fromFace, detached, facePlane(f)) {
			return nil, nil, false
		}
		kept = append(kept, fromFace...)
	}
	return kept, demoted, true
}

// demotedHoleFaceTrims trims a holed planar face through the exact-frame chart: its polygonal imprints
// become the chart's straight imprints, its holes the frame's conic edges.
func demotedHoleFaceTrims(full curvedFace, imprints [][2]math.Point3, other insideOracle, allOthers []curvedFace, op Op, isB bool) ([]curvedFace, bool) {
	curves := make([]geom.Curve3, 0, len(imprints))
	for _, s := range imprints {
		curves = append(curves, geom.NewLineSegment(s[0], s[1]))
	}
	return uvSplitOne(full, faceLoopBox(full), curves, uvKeepAt(full, allOthers, other, op, isB), op, isB)
}

// imprintMeetsHole reports an imprint segment crossing, ending on, or lying inside a detached hole
// circle — exactly, on the circle's conic form (the endpoint pad it replaced saw only a segment END
// on the rim, and let a segment passing THROUGH a hole reach the polygonal split).
func imprintMeetsHole(imprints [][2]math.Point3, holes []curvedLoop, pl geom.Plane) bool {
	for _, h := range holes {
		for _, e := range h.edges {
			pc, ok := toPlaneConic(e.curve, pl)
			if !ok {
				return true // an unexpected hole kind: conservative
			}
			for _, seg := range imprints {
				if segmentMeetsConic(pc, to2D(pl, seg[0]), to2D(pl, seg[1]), pl, e.curve) {
					return true
				}
			}
		}
	}
	return false
}

// segmentMeetsConic reports a segment crossing a closed conic or having an end inside it.
func segmentMeetsConic(pc planeConic, a, b math.Point2, pl geom.Plane, c geom.Curve3) bool {
	hits, tangent := conicFrameHits(pc, a, b, geom.ResolutionForPoints2D([]math.Point2{a, b}))
	if tangent || len(hits) > 0 {
		return true
	}
	cf, ok := geom.AsConic(c)
	if !ok || cf.Hyperbolic {
		return false
	}
	inside := func(q math.Point2) bool {
		d := cf.Center.VectorTo(to3D(pl, q))
		x, y := float64(d.Dot(cf.Major.AsVector()))/cf.A, float64(d.Dot(cf.Minor.AsVector()))/cf.B
		return x*x+y*y < 1
	}
	return inside(a) || inside(b)
}

// attachExactHoles attaches each detached curved hole loop to the kept fragment whose polygon contains
// it (a boundary sample point of the hole — strictly interior to exactly one arrangement cell, since
// imprintMeetsHole has proven every imprint clear of every detached hole). A hole no KEPT fragment
// contains sat in a cell the boolean dropped, and leaves with it: the material around the boss's rim
// is what the tool took (ADR-0060). false only for a hole with no edge to sample.
func attachExactHoles(frags []subFace, holes []curvedLoop, pl geom.Plane) bool {
	for _, h := range holes {
		if len(h.edges) == 0 {
			return false
		}
		q := to2D(pl, h.edges[0].start())
		if j := fragmentContaining(frags, q, pl); j >= 0 {
			frags[j].exactHoles = append(frags[j].exactHoles, h)
		}
	}
	return true
}

// fragmentContaining returns the index of the fragment whose outer polygon contains q (outside its
// polygon holes), or -1.
func fragmentContaining(frags []subFace, q math.Point2, pl geom.Plane) int {
	for j := range frags {
		if !pointInPolygon2D(q, ring2D(pl, frags[j].outer)) {
			continue
		}
		if polygonHoleContains(frags[j].holes, q, pl) {
			continue
		}
		return j
	}
	return -1
}

// polygonHoleContains reports whether q falls inside any of the fragment's polygon holes.
func polygonHoleContains(holes [][]math.Point3, q math.Point2, pl geom.Plane) bool {
	for _, hr := range holes {
		if pointInPolygon2D(q, ring2D(pl, hr)) {
			return true
		}
	}
	return false
}

// planUVImprints computes, for every exact-frame (uv) face of p, its imprint segments against the
// other operand's planar faces — the plane∩plane line clipped exactly against BOTH trims — and
// APPENDS the same segments to the other face's polygonal imprint list (otherImp, index-aligned with
// other.planar), so both sides split on identical coordinates. The uv×WALL pairs are imprinted
// separately (pairUVWallImprints, #3460); only uv×uv still declines here. ok=false declines: a uv face
// overlapping another uv face, a coplanar contact on a uv face, or a trim clip without a closed form.
func planUVImprints(p, other *facePartition, otherImp [][][2]math.Point3, _ bool) ([][]geom.Curve3, bool) {
	out := make([][]geom.Curve3, len(p.uv))
	for i, uf := range p.uv {
		box := inflateBox(p.uvBox[i])
		for j := range other.planarFull {
			if !box.Intersects(paddedFaceBox(other.planar[j])) {
				continue
			}
			curves, segs, ok := uvPairSegments(uf, other.planarFull[j])
			if !ok {
				return nil, false
			}
			out[i] = append(out[i], curves...)
			otherImp[j] = append(otherImp[j], segs...)
		}
	}
	return out, true
}

// uvPairSegments is the exact shared imprint of one (uv face, planar face) pair: the plane∩plane
// line clipped to the polygonal face's intervals AND the uv face's exact conic intervals, as curves for
// the uv face and segments for the polygonal one. A COPLANAR pair exchanges outlines instead, each clipped
// to the other's material (the flush contact, ADR-0060); ok=false for a failed exact clip.
func uvPairSegments(uf, of curvedFace) ([]geom.Curve3, [][2]math.Point3, bool) {
	p0, dir, ok := geom.PlanePlaneLine(facePlane(uf), facePlane(of))
	if !ok {
		if !coplanar(uf, of) {
			return nil, nil, true
		}
		onUV, okA := coplanarFaceImprints(uf, of)
		onOf, okB := coplanarStraightImprints(of, uf)
		return onUV, onOf, okA && okB
	}
	toolIv := faceLineIntervals(of, p0, dir)
	if len(toolIv) == 0 {
		return nil, nil, true
	}
	uvIv, exact := curvedFaceLineIntervals(uf, p0, dir)
	if !exact {
		return nil, nil, false
	}
	segs := lineIntervalSegments(p0, dir, intersectIntervals(toolIv, uvIv))
	// Each side keeps only the pieces INTERIOR to it — a piece running along a face's own boundary is
	// that face's edge already, not a split (the polygonal pairing's interiorSegments discipline).
	curves := make([]geom.Curve3, 0, len(segs))
	for _, s := range interiorSegments(uf, segs) {
		curves = append(curves, geom.NewLineSegment(s[0], s[1]))
	}
	return curves, interiorSegments(of, segs), true
}

// uvUVPairSegments is the exact shared imprint of two exact-frame faces: the plane∩plane line clipped
// to BOTH faces' exact intervals (ADR-0060). ok=false for a coplanar pair (flush contact, unmodelled)
// or a failed exact clip on either side.
func uvUVPairSegments(ua, ub curvedFace) ([][2]math.Point3, bool) {
	p0, dir, ok := geom.PlanePlaneLine(facePlane(ua), facePlane(ub))
	if !ok {
		return nil, !coplanar(ua, ub) // a coplanar pair takes coplanarUVUVImprints instead
	}
	ivA, exactA := curvedFaceLineIntervals(ua, p0, dir)
	ivB, exactB := curvedFaceLineIntervals(ub, p0, dir)
	if !exactA || !exactB {
		return nil, false
	}
	return lineIntervalSegments(p0, dir, intersectIntervals(ivA, ivB)), true
}

// lineIntervalSegments realises the non-degenerate intervals of a line as 3D segments.
func lineIntervalSegments(p0 math.Point3, dir math.Vector3, ivs [][2]float64) [][2]math.Point3 {
	var segs [][2]math.Point3
	for _, iv := range ivs {
		if iv[1]-iv[0] > 1e-9 { // tol:calibrated — planar imprint overlap length (see arrange2d arrTol)
			segs = append(segs, [2]math.Point3{p0.TranslateBy(dir.Scale(math.Scalar(iv[0]))), p0.TranslateBy(dir.Scale(math.Scalar(iv[1])))})
		}
	}
	return segs
}

// pairUVUVImprints imprints every overlapping (exact-frame face of a, exact-frame face of b) pair,
// appending the SAME segment to both faces' lists — the shared-coordinate invariant the weld relies on.
// ok=false declines with the pair's named reason (uvUVPairSegments).
func pairUVUVImprints(pa, pb *facePartition, uvA, uvB [][]geom.Curve3) bool {
	for i, ua := range pa.uv {
		box := inflateBox(pa.uvBox[i])
		for k, ub := range pb.uv {
			if !box.Intersects(inflateBox(pb.uvBox[k])) {
				continue
			}
			if coplanar(ua, ub) {
				onA, okA := coplanarFaceImprints(ua, ub)
				onB, okB := coplanarFaceImprints(ub, ua)
				if !okA || !okB {
					return false
				}
				uvA[i], uvB[k] = append(uvA[i], onA...), append(uvB[k], onB...)
				continue
			}
			segs, ok := uvUVPairSegments(ua, ub)
			if !ok {
				return false
			}
			for _, s := range segs {
				uvA[i] = append(uvA[i], geom.NewLineSegment(s[0], s[1]))
				uvB[k] = append(uvB[k], geom.NewLineSegment(s[0], s[1]))
			}
		}
	}
	return true
}

// uvSplitFaces trims each exact-frame face by its imprints through the shared (u,v) trimmer,
// classifying cells by the boolean's keep table over the other operand's membership oracle. A face
// with no imprints passes through whole (the pass-through classification). A kept Difference tool
// face reverses into the cavity. ok=false declines: a grazing contact, or a trim error.
func uvSplitFaces(p facePartition, imprints [][]geom.Curve3, other insideOracle, others []curvedFace, op Op, isB bool) ([]curvedFace, bool) {
	var out []curvedFace
	for i, uf := range p.uv {
		faces, ok := uvSplitOne(uf, p.uvBox[i], imprints[i], uvKeepAt(uf, others, other, op, isB), op, isB)
		if !ok {
			return nil, false
		}
		out = append(out, faces...)
	}
	return out, true
}

// uvWholeKept classifies an imprint-free exact-frame face as a whole, at a point strictly inside it (a
// loop vertex would sit on a coplanar neighbour's boundary, where the cover test is undefined).
func uvWholeKept(uf curvedFace, keepAt func(math.Point3) bool, op Op, isB bool) ([]curvedFace, bool) {
	p, ok := faceInteriorPoint(uf)
	if !ok {
		return nil, false
	}
	if !keepAt(p) {
		return nil, true
	}
	if op == Difference && isB {
		return reverseCurvedFaces([]curvedFace{uf}), true
	}
	return []curvedFace{uf}, true
}

// uvSplitOne trims one exact-frame face (or classifies it whole when it has no imprints).
func uvSplitOne(uf curvedFace, box math.Box, imprint []geom.Curve3, keepAt func(math.Point3) bool, op Op, isB bool) ([]curvedFace, bool) {
	if len(imprint) == 0 {
		return uvWholeKept(uf, keepAt, op, isB)
	}
	c, ok := newPlaneFaceUV(uf, geom.ResolutionForBox(box))
	if !ok || !planeFaceContactOK(c, imprint) {
		return nil, false
	}
	faces, _, err := trimByImprint(c, uf, uf.surface, imprint, planeFaceMaterial(c, keepAt))
	if err != nil {
		return nil, false
	}
	if op == Difference && isB {
		faces = reverseCurvedFaces(faces)
	}
	return faces, true
}

// bothUVImprints plans both operands' exact-frame imprints (planUVImprints each way).
func bothUVImprints(pa, pb *facePartition, impA, impB [][][2]math.Point3) (uvImpA, uvImpB [][]geom.Curve3, ok bool) {
	uvImpA, okA := planUVImprints(pa, pb, impB, false)
	uvImpB, okB := planUVImprints(pb, pa, impA, true)
	return uvImpA, uvImpB, okA && okB
}
