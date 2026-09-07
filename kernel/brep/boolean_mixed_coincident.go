// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// The DEGENERATE OVERLAP: two faces of the two operands lying on the SAME surface, so their contact is
// a two-dimensional region rather than a curve (ADR-0045). The planar boolean has always had it — two
// flush faces are covered once, from one side, by the ON/ON table — and the rule is not about planes.
// Two coaxial cylinders of one radius overlap exactly so, and the mixed pipeline declined them on the
// only evidence it looked for: an intersector asked for a crossing between two identical surfaces
// answers that it cannot, and "cannot" was read as "unsupported" (ADR-0061 stage 4).
//
// The rule has two halves, and they are the same two the crossing case has: an IMPRINT, which for a
// coincident pair is the other face's own boundary rather than a section curve, and a CLASSIFICATION,
// which is the ON/ON table rather than the membership oracle.

// coincidentCover reports whether a point of f's surface is covered by a face of the other operand
// lying on the SAME surface, and if so whether that face's outward normal agrees with f's — shared
// contact — or opposes it. It is the general form of the coplanar cover: identity of the surfaces
// through geom.SurfacesCoincide, containment through each face's own trim.
func coincidentCover(f curvedFace, p math.Point3, others []curvedFace) (covered, sameNormal bool) {
	res := geom.ResolutionForBox(faceLoopBox(f))
	for _, o := range others {
		if !geom.SurfacesCoincide(f.surface, o.surface, res) || !faceHoldsPoint(o, p) {
			continue
		}
		return true, float64(outwardNormalAt(f, p).Dot(outwardNormalAt(o, p))) > 0
	}
	return false, false
}

// faceHoldsPoint is a face's trim containment at a point already on its surface: the exact conic/polygon
// test for a planar face (the verdict the planar boolean has always used, unchanged), the developed
// (u,v) trim for every other surface.
func faceHoldsPoint(f curvedFace, p math.Point3) bool {
	if _, isPlane := planeOf(f); isPlane {
		return faceContainsExact(f, p)
	}
	return pointInTrimUV(f, p)
}

// outwardNormalAt is the face's OUTWARD normal at a point of its surface — the surface normal, turned
// round for a reversed face, whose surface normal points into the material.
func outwardNormalAt(f curvedFace, p math.Point3) math.Vector3 {
	n := geom.SurfaceNormalAt(f.surface, p)
	if f.reversed {
		return n.Scale(-1)
	}
	return n
}

// coincidentKeepAt is the keep test every chart shares: a point covered by a face of the other operand
// on the same surface follows the ON/ON table (coplanarKeep — the shared region is emitted once, as A's
// copy); every other point follows the keep table over the membership oracle. Asking the oracle there
// instead is what a coincident overlap makes meaningless: the point is ON the other solid's boundary,
// where inside and outside are not the question.
func coincidentKeepAt(f curvedFace, others []curvedFace, other insideOracle, op Op, isB bool) func(math.Point3) bool {
	return func(pt math.Point3) bool {
		if covered, same := coincidentCover(f, pt, others); covered {
			return coplanarKeep(op, isB, same)
		}
		return keep(op, isB, other.inside(pt))
	}
}

// coincidentWallImprint is the shared imprint of two walls on ONE ruled surface: each band's rim
// circles, kept where they fall strictly inside the other's band. That is the whole of the contact —
// two coincident surfaces cross nowhere, and what divides them is where one band ends inside the other.
// A rim on the receiving chart's OWN frame is dropped there by ruledFaceUV.admits, so returning both
// sides' rims together keeps the "solve once, write to both" rule the crossing pairs follow.
func coincidentWallImprint(ra, rb ruledSide) []geom.Curve3 {
	out := rimsInsideBand(ra, rb)
	return append(out, rimsInsideBand(rb, ra)...)
}

// rimsInsideBand returns the rim circles of `of` that lie strictly inside `in`'s band. The rims are
// built from the band's own world anchors and radii rather than read off it: a band's v is measured in
// its OWN frame, so two coincident walls both report [0, h] and only the world points place them
// against one another.
func rimsInsideBand(of, in ruledSide) []geom.Curve3 {
	var out []geom.Curve3
	for _, rim := range [2]struct {
		at math.Point3
		r  float64
	}{{of.band.bottom, of.band.rBot}, {of.band.top, of.band.rTop}} {
		v := bandV(rim.at, in.axis, in.band)
		if inside, _ := bandPlacement(v, v, in.band); !inside || rim.r <= 0 {
			continue
		}
		// The receiving chart's own Ref fixes the circle's angle 0, so the imprint and the frame it is
		// arranged against read one azimuth.
		out = append(out, geom.Circle{Center: rim.at, Normal: in.frame.Axis.AsUnit(),
			RefDir: in.frame.Ref.AsUnit(), Radius: rim.r})
	}
	return out
}

// sectionOnFaceBoundary reports whether a section curve runs ALONG one of the receiving face's own
// boundary edges. It is that face's half of the rule sectionOnWallEdge states for the wall: the section
// a coaxial cylinder's wall cuts from the plane of the cap that closes the other IS that cap's own rim,
// and no face is split by its own boundary (ADR-0061 stage 4).
//
// Every sample of the section must lie on ONE edge's own span. A section that merely touches the
// boundary, or that runs past its end, is a genuine imprint and is left to the island rule — which is
// why the edge's span is asked about and not just the conic it lies on.
func sectionOnFaceBoundary(cv geom.Curve3, f curvedFace, res geom.Resolution) bool {
	for _, l := range f.loops {
		for _, e := range l.edges {
			if curveRunsAlongEdge(cv, e, res) {
				return true
			}
		}
	}
	return false
}

// curveRunsAlongEdge reports whether every sample of cv lies on the edge's own parameter span.
func curveRunsAlongEdge(cv geom.Curve3, e loopEdge, res geom.Resolution) bool {
	lo, hi := cv.Domain()
	for i := 0; i <= sectionContactSamples; i++ {
		p := cv.PointAt(lo + (hi-lo)*float64(i)/sectionContactSamples)
		if _, ok := curveParamWithin(e.curve, e.t0, e.t1, p, res); !ok {
			return false
		}
	}
	return true
}

// sectionContactSamples is how densely a section is walked when asking whether it runs along an edge.
// The two are analytic curves and the question is metric, so this only has to be dense enough that a
// section leaving the edge is caught at some station; a section that never leaves it agrees everywhere.
const sectionContactSamples = 16

// mergeCoincidentFaces joins result faces that lie on ONE surface and share an entire boundary loop.
//
// The union of two coaxial cylinders is one cylinder, not two bands meeting at a rim: the two operands'
// walls are the same surface, the classification keeps one band from each, and the circle where they
// meet bounds nothing — the surface is smooth across it. Leaving it there invents an edge the model
// does not have, which a user can select and a downstream fillet or chamfer would try to round. This is
// the curved form of the coplanar merge the planar boolean has always done (ADR-0057, ADR-0061 stage 4).
//
// The scope is the shared loop taken WHOLE: two faces whose common boundary is one complete loop of
// each merge, dropping it and keeping the rest. A partial overlap would need a two-dimensional union of
// the trims, which is a different operation, and is left alone rather than approximated.
func mergeCoincidentFaces(faces []curvedFace) []curvedFace {
	for merged := true; merged; {
		merged = false
		for i := 0; i < len(faces) && !merged; i++ {
			for j := i + 1; j < len(faces) && !merged; j++ {
				joined, ok := joinOnSharedLoop(faces[i], faces[j])
				if !ok {
					continue
				}
				faces[i] = joined
				faces = append(faces[:j], faces[j+1:]...)
				merged = true
			}
		}
	}
	return faces
}

// joinOnSharedLoop merges two faces on one surface across the boundary they share: a whole loop of
// each, which simply drops from both, or a single edge, where the loops SPLICE — the other face's chain
// takes the shared edge's place, which is what dissolving an edge between two faces means. ok=false
// when they are not on one surface, differ in sense, or share neither.
func joinOnSharedLoop(a, b curvedFace) (curvedFace, bool) {
	if a.reversed != b.reversed || !geom.SurfacesCoincide(a.surface, b.surface, geom.ResolutionForBox(faceLoopBox(a))) {
		return curvedFace{}, false
	}
	res := geom.ResolutionForBox(faceLoopBox(a).Union(faceLoopBox(b)))
	loops, ok := joinedLoops(a, b, res)
	if !ok {
		return curvedFace{}, false
	}
	out := a
	out.loops = loops
	out.chart = nil // the merged trim is neither operand's chart; its loops determine it
	if len(b.lineage.Key()) > 0 {
		out.aliasKeys = append(append([][]byte(nil), a.aliasKeys...), b.lineage.Key())
		out.aliasKeys = append(out.aliasKeys, b.aliasKeys...)
	}
	return out, true
}

// joinedLoops is the merged face's boundary, by whichever of the two shared-boundary forms applies.
func joinedLoops(a, b curvedFace, res geom.Resolution) ([]curvedLoop, bool) {
	if ia, ib, ok := sharedLoopPair(a, b, res); ok {
		return append(loopsExcept(a.loops, ia), loopsExcept(b.loops, ib)...), true
	}
	return spliceOnSharedEdge(a, b, res)
}

// spliceOnSharedEdge merges two faces whose only common boundary is ONE edge. The two traverse it in
// opposite senses — they lie on opposite sides of it — so removing it from each leaves two chains that
// join end to end, and the merged loop is A's loop with that edge replaced by B's chain.
//
// The union of two coaxial cylinder bands is the case it was written for: each band's boundary is a
// single loop carrying its seam twice and its two rims, and the rim they share is one edge of each.
// ok=false unless exactly one edge is shared, which is what makes the splice unambiguous.
func spliceOnSharedEdge(a, b curvedFace, res geom.Resolution) ([]curvedLoop, bool) {
	ia, ea, ib, eb, n := sharedEdgePair(a, b, res)
	if n != 1 {
		return nil, false
	}
	spliced := curvedLoop{edges: append(append(append([]loopEdge{},
		a.loops[ia].edges[:ea]...), rotatedChain(b.loops[ib].edges, eb)...),
		a.loops[ia].edges[ea+1:]...)}
	return append(append(loopsExcept(a.loops, ia), spliced), loopsExcept(b.loops, ib)...), true
}

// rotatedChain is a loop's edges with the one at k removed, rotated to start just after it — the chain
// that runs from that edge's end back round to its start.
func rotatedChain(edges []loopEdge, k int) []loopEdge {
	out := make([]loopEdge, 0, len(edges)-1)
	for i := 1; i < len(edges); i++ {
		out = append(out, edges[(k+i)%len(edges)])
	}
	return out
}

// sharedEdgePair returns the first edge the two faces share and how many they share in all.
func sharedEdgePair(a, b curvedFace, res geom.Resolution) (ia, ea, ib, eb, n int) {
	ia, ea, ib, eb = -1, -1, -1, -1
	for i, la := range a.loops {
		for j, e := range la.edges {
			for k, lb := range b.loops {
				for m, o := range lb.edges {
					if !edgesRunTogether(e, o, res) {
						continue
					}
					if n == 0 {
						ia, ea, ib, eb = i, j, k, m
					}
					n++
				}
			}
		}
	}
	return ia, ea, ib, eb, n
}

// edgesRunTogether reports whether two loop edges trace the same stretch of the same curve.
func edgesRunTogether(a, b loopEdge, res geom.Resolution) bool {
	sa, sb := geom.SubCurve(a.curve, a.t0, a.t1), geom.SubCurve(b.curve, b.t0, b.t1)
	return curveRunsAlongEdge(sa, b, res) && curveRunsAlongEdge(sb, a, res)
}

// sharedLoopPair finds one loop of a and one of b that are the same closed boundary.
func sharedLoopPair(a, b curvedFace, res geom.Resolution) (ia, ib int, ok bool) {
	for i, la := range a.loops {
		for j, lb := range b.loops {
			if loopsRunTogether(la, lb, res) {
				return i, j, true
			}
		}
	}
	return 0, 0, false
}

// loopsRunTogether reports whether two loops trace the same closed boundary: every edge of each runs
// along an edge of the other, which is direction-free and needs no matching of the edges' order.
func loopsRunTogether(a, b curvedLoop, res geom.Resolution) bool {
	return len(a.edges) > 0 && len(b.edges) > 0 &&
		everyEdgeRunsAlong(a, b, res) && everyEdgeRunsAlong(b, a, res)
}

// everyEdgeRunsAlong reports whether every edge of one loop lies on some edge of the other.
func everyEdgeRunsAlong(of, in curvedLoop, res geom.Resolution) bool {
	for _, e := range of.edges {
		on := false
		for _, o := range in.edges {
			if curveRunsAlongEdge(geom.SubCurve(e.curve, e.t0, e.t1), o, res) {
				on = true
				break
			}
		}
		if !on {
			return false
		}
	}
	return true
}

// loopsExcept returns the loops with index k removed.
func loopsExcept(loops []curvedLoop, k int) []curvedLoop {
	out := make([]curvedLoop, 0, len(loops))
	for i, l := range loops {
		if i != k {
			out = append(out, l)
		}
	}
	return out
}
