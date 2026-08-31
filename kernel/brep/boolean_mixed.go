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
// and welded by the same unified stitch. Scope is conservative and declines loudly (ErrNonPlanar →
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
	return append(append([]curvedFace{}, p.planarFull...), p.pass...)
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
// It returns ErrNonPlanar whenever the conservative scope gate declines, so the caller falls to the
// curved/CSG paths exactly as before.
func booleanMixed(op Op, a, b *topo.Body) (*topo.Body, bool, error) {
	pa, pb := partitionFaces(a), partitionFaces(b)
	if !passClearOf(pa, pb) || !passClearOf(pb, pa) {
		return nil, false, ErrNonPlanar
	}
	pra, prb := newInsideOracle(a, pa.allFaces()), newInsideOracle(b, pb.allFaces())
	pairs := crossingFaceCandidates(pa.planar, pb.planar)
	if coplanarCurvedContact(pa, pb, pairs) {
		return nil, false, ErrNonPlanar // a flush contact on a curved-loop face is not modelled here
	}
	// Imprints clip against the TRUE trims (planarFull): a face's detached holes are void, so no
	// imprint is minted inside them (the phantom that broke the detached-hole premise).
	impA, impB, prov := imprintCandidates(pa.planarFull, pb.planarFull, pairs)
	keptA, okHA := selectFacesDetached(pa.planar, impA, prb, pb.planar, pairs.bForA, op, false, prov, pa.planarHoles)
	keptB, okHB := selectFacesDetached(pb.planar, impB, pra, pa.planar, pairs.aForB, op, true, prov, pb.planarHoles)
	if !okHA || !okHB {
		return nil, false, ErrNonPlanar
	}
	kept := append(append([]subFace{}, keptA...), keptB...)
	passA, okA := passThroughKept(pa.pass, prb, op, false)
	passB, okB := passThroughKept(pb.pass, pra, op, true)
	if !okA || !okB {
		return nil, false, ErrNonPlanar
	}
	return stitch(kept, append(passA, passB...), prov)
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
