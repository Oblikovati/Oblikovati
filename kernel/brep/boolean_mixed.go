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
	planar  []curvedFace
	pass    []curvedFace
	passBox []math.Box
	body    *topo.Body
}

// partitionFaces flattens b and buckets each face for per-face dispatch.
func partitionFaces(b *topo.Body) facePartition {
	p := facePartition{body: b}
	topoFaces := b.Faces()
	for i, cf := range facesOfAny(b) {
		if polygonalPlanar(cf) {
			p.planar = append(p.planar, cf)
			continue
		}
		p.pass = append(p.pass, cf)
		p.passBox = append(p.passBox, topoFaces[i].RangeBox())
	}
	return p
}

// polygonalPlanar reports whether a face can take the exact polygonal planar pipeline: a plane
// surface whose every boundary edge is straight (a curved edge — a boss seat's rim circle — has no
// faithful point-ring form there).
func polygonalPlanar(f curvedFace) bool {
	if _, isPlane := f.surface.(geom.Plane); !isPlane {
		return false
	}
	for _, l := range f.loops {
		for _, e := range l.edges {
			switch e.curve.(type) {
			case geom.LineSegment, geom.Line:
			default:
				return false
			}
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

// allFaces is the partition's full flattened face list (planar then pass), for the membership oracle.
func (p facePartition) allFaces() []curvedFace {
	return append(append([]curvedFace{}, p.planar...), p.pass...)
}

// passDisjointFrom reports whether every pass-through face's box is clear of every face box of the
// other body (padded by the pipeline's cull slack) — the scope gate: only then can no imprint,
// membership boundary, or T-junction touch a pass-through face.
func passDisjointFrom(p facePartition, other *topo.Body) bool {
	pad := math.Scalar(facePairCullPad)
	for _, pb := range p.passBox {
		inflated := math.NewBox(pb.Min.TranslateBy(math.V3(-pad, -pad, -pad)), pb.Max.TranslateBy(math.V3(pad, pad, pad)))
		for _, of := range other.Faces() {
			if inflated.Intersects(of.RangeBox()) {
				return false
			}
		}
	}
	return true
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
	if !passDisjointFrom(pa, b) || !passDisjointFrom(pb, a) {
		return nil, false, ErrNonPlanar
	}
	pra, prb := newInsideOracle(a, pa.allFaces()), newInsideOracle(b, pb.allFaces())
	pairs := crossingFaceCandidates(pa.planar, pb.planar)
	impA, impB, prov := imprintCandidates(pa.planar, pb.planar, pairs)
	var kept []subFace
	kept = append(kept, selectFaces(pa.planar, impA, prb, pb.planar, pairs.bForA, op, false, prov)...)
	kept = append(kept, selectFaces(pb.planar, impB, pra, pa.planar, pairs.aForB, op, true, prov)...)
	passA, okA := passThroughKept(pa.pass, prb, op, false)
	passB, okB := passThroughKept(pb.pass, pra, op, true)
	if !okA || !okB {
		return nil, false, ErrNonPlanar
	}
	return stitch(kept, append(passA, passB...), prov)
}
