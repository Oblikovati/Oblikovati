// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"sort"

	"oblikovati.org/kernel/topo"
)

// cornerKind distinguishes the two shared-corner shapes the setback engine treats: a 2-edge miter
// (in the miters map) versus a 3-edge trihedral blend (in the blends map). A vertex is exactly one
// kind — computeCorners emits either a cornerMiter or a cornerBlend for a vertex, never both — so the
// two id spaces are disjoint.
type cornerKind int

const (
	miterCorner cornerKind = iota // a 2-edge corner: two filleted edges sharing a face meet at a seam
	blendCorner                   // a 3-edge trihedral corner: a sphere/torus corner patch
)

// cornerRef names one shared corner by its vertex id and kind — the unit accumulate iterates over,
// sorted by vid so face/patch emission is deterministic regardless of Go map order (byte-identity R1).
type cornerRef struct {
	vid  uint64
	kind cornerKind
}

// cornerTreatment is the tag classifyCorner assigns a corner: the OCCT setback shape its converging
// fillets require (geometry-derivation §Candidate methods). decline is every corner the engine leaves
// on the baseline (convex miter, curved host, non-orthogonal, wrong valence, …).
type cornerTreatment int

const (
	treatDecline       cornerTreatment = iota
	treatDihedralMiter                 // 2 concave orthogonal planar → seam setback, no patch (P1: L1/L7/N5)
	treatConcaveSphere                 // 3 concave orthogonal planar → void-flipped sphere octant (P2: K6/L4)
	treatMixedTorus                    // mixed-sense orthogonal planar → torus R=2r (2cc+1cvx: K9/M2/L6; 1cc+2cvx: B5/C4/D7)
	treatConvexRunoff                  // 3 convex planar (body all-planar) → oblique run-off clip (P4: A8/A6)
	treatRadiusTorus                   // mixed-RADIUS convex orthogonal planar → torus R=rB−rS (A4/E3, fillet_corner_radiustorus.go)
)

// setbackCtx bundles the read-only inputs classifyCorner and accumulate consume: the already-solved
// fillet set, its corner maps, and the body (for the convex-wedge all-planar precondition). ends is
// miterCornerEnds(fils) precomputed once, so the per-corner classify/accumulate is O(1) in the map.
type setbackCtx struct {
	body   *topo.Body
	fils   []edgeFillet
	blends map[uint64]*cornerBlend
	miters map[uint64]*cornerMiter
	ends   map[uint64][]miterEnd
}

// classifyCorner maps one corner to its treatment tag, delegating ENTIRELY to the existing gate
// predicates (no new geometry). It is pure and owns no assembly — the seam where a future 5th corner
// type plugs in as one new tag + predicate. Example: classifyCorner(cornerRef{vid, blendCorner}, ctx).
func classifyCorner(c cornerRef, ctx setbackCtx) cornerTreatment {
	if c.kind == miterCorner {
		return classifyMiterCorner(c.vid, ctx)
	}
	return classifyBlendCorner(c.vid, ctx)
}

// classifyMiterCorner tags a 2-edge corner dihedralMiter when its two arms form the L1-class
// concave-orthogonal-planar miter (concaveOrthogonalDihedralMiter), else decline.
func classifyMiterCorner(vid uint64, ctx setbackCtx) cornerTreatment {
	pair, cm := ctx.ends[vid], ctx.miters[vid]
	if len(pair) != 2 || cm == nil {
		return treatDecline // a miter corner is shared by exactly two filleted edges
	}
	efA, efB := &ctx.fils[pair[0].fi], &ctx.fils[pair[1].fi]
	if concaveOrthogonalDihedralMiter(efA, efB, cm) {
		return treatDihedralMiter
	}
	return treatDecline
}

// classifyBlendCorner tags a 3-edge trihedral corner by its sense signature — the three predicates are
// mutually exclusive by sense (mixed 2+1, all-concave 3, all-convex 3), so their order is immaterial to
// the result; convexRunoff additionally requires the whole body planar (the P4 scope boundary).
func classifyBlendCorner(vid uint64, ctx setbackCtx) cornerTreatment {
	cb := ctx.blends[vid]
	if cb == nil || cb.vertex == nil {
		return treatDecline
	}
	if cb.radiusTorus != nil {
		// The mixed-radius torus corner was matched and solved eagerly at computeCorners time;
		// its transient blend is the routing marker (never a real sphere corner).
		return treatRadiusTorus
	}
	bands := cornerBandsAt(vid, ctx.fils)
	if pivot, pair, ok := splitMixedSense(bands); ok {
		if faces := mixedCornerFaces(pivot, pair); len(faces) == 3 && orthogonalPlanarTriple(faces) {
			return treatMixedTorus
		}
	}
	if _, ok := concaveTrihedralCornerFaces(vid, cb, ctx.fils); ok {
		return treatConcaveSphere
	}
	if _, ok := convexTrihedralCornerBands(vid, cb, ctx.fils); ok && allBodyFacesPlanar(ctx.body) {
		return treatConvexRunoff
	}
	return treatDecline
}

// sortedCornerRefs enumerates every shared corner (miters ∪ blends) sorted ascending by vertex id, so
// accumulate's channel emission is deterministic across runs regardless of Go map iteration order (R1).
func sortedCornerRefs(ctx setbackCtx) []cornerRef {
	refs := make([]cornerRef, 0, len(ctx.miters)+len(ctx.blends))
	for vid := range ctx.miters {
		refs = append(refs, cornerRef{vid: vid, kind: miterCorner})
	}
	for vid := range ctx.blends {
		refs = append(refs, cornerRef{vid: vid, kind: blendCorner})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].vid < refs[j].vid })
	return refs
}
