// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

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
		if _, ok := ruledSideBandOf(cf); ok {
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

// passSamplePoint is a point ON the face for its uniform membership test: any loop-edge start. A
// boundaryless face (a full sphere) has none and declines.
func passSamplePoint(f curvedFace) (math.Point3, bool) {
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
	pra, prb := newInsideOracle(a, pa.allFaces()), newInsideOracle(b, pb.allFaces())
	pairs := crossingFaceCandidates(pa.planar, pb.planar)
	if coplanarCurvedContact(pa, pb, pairs) {
		return nil, false, ErrUnsupportedMixedBoolean // a flush contact on a curved-loop face is not modelled here
	}
	// Imprints clip against the TRUE trims (planarFull): a face's detached holes are void, so no
	// imprint is minted inside them (the phantom that broke the detached-hole premise). The exact-frame
	// (uv) faces' imprints run BEFORE the polygonal split, mirroring the same segments onto the other
	// side's imprint lists so the two faces split on identical coordinates.
	impA, impB, prov := imprintCandidates(pa.planarFull, pb.planarFull, pairs)
	uvImpA, uvImpB, wallImpA, wallImpB, okI := mixedCurvedImprints(&pa, &pb, impA, impB)
	if !okI {
		return nil, false, ErrUnsupportedMixedBoolean
	}
	kept, okK := mixedKeptFragments(pa, pb, impA, impB, pra, prb, pairs, op, prov)
	pass, okP := mixedPassFaces(pa, pb, pra, prb, uvImpA, uvImpB, op)
	walls, okQ := mixedWallFaces(pa, pb, pra, prb, wallImpA, wallImpB, op)
	if !okK || !okP || !okQ {
		return nil, false, ErrUnsupportedMixedBoolean
	}
	return stitch(kept, append(pass, walls...), prov)
}

// mixedKeptFragments runs both operands' polygonal splits (with detached-hole re-attachment).
func mixedKeptFragments(pa, pb facePartition, impA, impB [][][2]math.Point3, pra, prb insideOracle, pairs facePairs, op Op, prov []imprintSeg) ([]subFace, bool) {
	keptA, okA := selectFacesDetached(pa.planar, impA, prb, pb.planar, pairs.bForA, op, false, prov, pa.planarHoles)
	keptB, okB := selectFacesDetached(pb.planar, impB, pra, pa.planar, pairs.aForB, op, true, prov, pb.planarHoles)
	return append(append([]subFace{}, keptA...), keptB...), okA && okB
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
	return uvA, uvB, wallA, wallB, okXA && okXB
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
	uvA, okVA := uvSplitFaces(pa, uvImpA, prb, op, false)
	uvB, okVB := uvSplitFaces(pb, uvImpB, pra, op, true)
	if !okA || !okB || !okVA || !okVB {
		return nil, false
	}
	return append(append(append(passA, passB...), uvA...), uvB...), true
}

// selectFacesDetached is selectFaces plus the exact-hole re-attachment: after a face's fragments are
// selected, each of its detached curved holes is attached to the fragment containing it. ok=false
// (decline) when a hole's containing fragment cannot be identified.
func selectFacesDetached(faces []curvedFace, imprints [][][2]math.Point3, other insideOracle, others []curvedFace, otherCand [][]int, op Op, isB bool, prov []imprintSeg, detached [][]curvedLoop) ([]subFace, bool) {
	var kept []subFace
	for i, f := range faces {
		if len(detached[i]) > 0 && imprintTouchesHole(imprints[i], detached[i]) {
			return nil, false // the tool crosses a detached hole's rim: this split cannot model it
		}
		fromFace := selectFragments(f, imprints[i], other, facesAt(others, otherCand[i]), op, isB, prov)
		if len(detached[i]) > 0 && !attachExactHoles(fromFace, detached[i], facePlane(f)) {
			return nil, false
		}
		kept = append(kept, fromFace...)
	}
	return kept, true
}

// attachExactHoles attaches each detached curved hole loop to the kept fragment whose polygon contains
// it (a boundary sample point of the hole — strictly interior to exactly one fragment, since the tool
// and its imprints are provably far from every detached hole). false when no kept fragment contains a
// hole — the material around it was cut away in a configuration this dispatch does not model.
func attachExactHoles(frags []subFace, holes []curvedLoop, pl geom.Plane) bool {
	for _, h := range holes {
		if len(h.edges) == 0 {
			return false
		}
		q := to2D(pl, h.edges[0].start())
		j := fragmentContaining(frags, q, pl)
		if j < 0 {
			return false
		}
		frags[j].exactHoles = append(frags[j].exactHoles, h)
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

// coplanarCurvedContact reports a candidate pair where either face carries curved loops AND the two
// are coplanar — a flush contact whose ON/ON classification the mixed dispatch does not model yet.
func coplanarCurvedContact(pa, pb facePartition, pairs facePairs) bool {
	for i, js := range pairs.bForA {
		for _, j := range js {
			fa, fb := pa.planarFull[i], pb.planarFull[j]
			if (!allStraightFace(fa) || !allStraightFace(fb)) && coplanar(fa, fb) {
				return true
			}
		}
	}
	return false
}

// imprintTouchesHole reports an imprint segment endpoint landing on (or within the weld pad of) a
// detached hole circle — a tool crossing the hole boundary, which the detached-hole split cannot
// model (the hole is invisible to the polygonal arrangement); the boolean declines to its fallbacks.
func imprintTouchesHole(imprints [][2]math.Point3, holes []curvedLoop) bool {
	for _, h := range holes {
		for _, e := range h.edges {
			c, isCircle := e.curve.(geom.Circle)
			if !isCircle {
				return true // unexpected hole kind: conservative
			}
			for _, seg := range imprints {
				if onCirclePad(seg[0], c) || onCirclePad(seg[1], c) {
					return true
				}
			}
		}
	}
	return false
}

// onCirclePad reports p within the cull pad of the circle's rim.
func onCirclePad(p math.Point3, c geom.Circle) bool {
	return stdmath.Abs(float64(c.Center.DistanceTo(p))-c.Radius) <= facePairCullPad
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
		box := inflateBox(p.uvBox[i], facePairCullPad)
		if boxesOverlapAny(box, other.uvBox) {
			return nil, false // uv×uv pairs have no imprint pairing yet: decline
		}
		for j := range other.planarFull {
			if !box.Intersects(paddedFaceBox(other.planar[j])) {
				continue
			}
			segs, ok := uvPairSegments(uf, other.planarFull[j])
			if !ok {
				return nil, false
			}
			for _, s := range segs {
				out[i] = append(out[i], geom.NewLineSegment(s[0], s[1]))
				otherImp[j] = append(otherImp[j], s)
			}
		}
	}
	return out, true
}

// uvPairSegments is the exact shared imprint of one (uv face, planar face) pair: the plane∩plane
// line clipped to the polygonal face's intervals AND the uv face's exact conic intervals. ok=false
// for a coplanar pair (flush contact, unmodelled) or a failed exact clip.
func uvPairSegments(uf, of curvedFace) ([][2]math.Point3, bool) {
	p0, dir, ok := geom.PlanePlaneLine(facePlane(uf), facePlane(of))
	if !ok {
		return nil, !coplanar(uf, of)
	}
	toolIv := faceLineIntervals(of, p0, dir)
	if len(toolIv) == 0 {
		return nil, true
	}
	uvIv, exact := curvedFaceLineIntervals(uf, p0, dir)
	if !exact {
		return nil, false
	}
	var segs [][2]math.Point3
	for _, iv := range intersectIntervals(toolIv, uvIv) {
		if iv[1]-iv[0] > 1e-9 { // tol:calibrated — planar imprint overlap length (see arrange2d arrTol)
			segs = append(segs, [2]math.Point3{p0.TranslateBy(dir.Scale(math.Scalar(iv[0]))), p0.TranslateBy(dir.Scale(math.Scalar(iv[1])))})
		}
	}
	return segs, true
}

// uvSplitFaces trims each exact-frame face by its imprints through the shared (u,v) trimmer,
// classifying cells by the boolean's keep table over the other operand's membership oracle. A face
// with no imprints passes through whole (the pass-through classification). A kept Difference tool
// face reverses into the cavity. ok=false declines: a grazing contact, or a trim error.
func uvSplitFaces(p facePartition, imprints [][]geom.Curve3, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	var out []curvedFace
	for i, uf := range p.uv {
		faces, ok := uvSplitOne(uf, p.uvBox[i], imprints[i], other, op, isB)
		if !ok {
			return nil, false
		}
		out = append(out, faces...)
	}
	return out, true
}

// uvSplitOne trims one exact-frame face (or classifies it whole when it has no imprints).
func uvSplitOne(uf curvedFace, box math.Box, imprint []geom.Curve3, other insideOracle, op Op, isB bool) ([]curvedFace, bool) {
	if len(imprint) == 0 {
		return passThroughKept([]curvedFace{uf}, other, op, isB)
	}
	c, ok := newPlaneFaceUV(uf, geom.ResolutionForBox(box))
	if !ok || !planeFaceContactOK(c, imprint) {
		return nil, false
	}
	keepAt := func(pt math.Point3) bool { return keep(op, isB, other.inside(pt)) }
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

// boxesOverlapAny reports box intersecting any of the listed boxes.
func boxesOverlapAny(box math.Box, boxes []math.Box) bool {
	for _, b := range boxes {
		if box.Intersects(b) {
			return true
		}
	}
	return false
}
