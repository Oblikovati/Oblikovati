// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Broad-phase face-pair culling for the planar boolean (#1607). The imprint, provenance and
// coplanar-cover scans only ever produce output for a face pair when some point lies ON both
// faces (an imprint/overlap segment, or a covered classification sample) within tolerances
// orders of magnitude below the pad here, so a pair whose padded AABBs are disjoint can be
// skipped without touching any result bit. The candidate set is computed ONCE per boolean pass
// via a [geom.BoxTree] (the kernel/ops self-intersection broad phase of #1411, shared through
// geom) and handed to every consumer — retiring both the O(Fa·Fb) scan and its 2–3×
// recomputation across imprintAll/provenanceOf.

// facePairCullPad inflates each face's AABB before the overlap test. It must exceed every
// tolerance under which the narrow phase still produces output for a near-touching pair —
// arrTol (1e-9), the 1e-7 weld/coplanar/boundary-imprint gaps, and planarStitchGrid (1e-6) —
// and at 10× the planar stitch grid it clears the largest by an order of magnitude (applied to
// BOTH boxes, doubling the slack). Culling only ever narrows which pairs are TESTED, never how
// a tested pair classifies.
const facePairCullPad = 10 * planarStitchGrid // tol:calibrated — box-cull slack over the planar weld family (see arrange2d arrTol)

// facePairs is the boolean pass's shared candidate set: for each face of A, the ascending
// indices of B faces whose padded boxes overlap it — and the same relation transposed — so
// every consumer iterates pairs in the exact (i, then j) order the retired brute scan used.
type facePairs struct {
	bForA [][]int // per A-face index: candidate B-face indices, ascending
	aForB [][]int // per B-face index: candidate A-face indices, ascending
}

// crossingFaceCandidates computes the candidate pair set once: a BoxTree over B's padded face
// boxes, queried with each of A's padded face boxes.
func crossingFaceCandidates(fa, fb []planarFace) facePairs {
	boxesB := make([]math.Box, len(fb))
	for j := range fb {
		boxesB[j] = paddedFaceBox(fb[j])
	}
	tree := geom.NewBoxTree(boxesB)
	p := facePairs{bForA: make([][]int, len(fa)), aForB: make([][]int, len(fb))}
	for i := range fa {
		tree.Query(paddedFaceBox(fa[i]), func(j int) bool {
			p.bForA[i] = append(p.bForA[i], j)
			return false
		})
		// The tree yields spatial order; consumers must see the brute scan's ascending-j order
		// (imprint slice order feeds the 2D arrangement, which is order-sensitive at tolerance).
		sort.Ints(p.bForA[i])
		for _, j := range p.bForA[i] {
			p.aForB[j] = append(p.aForB[j], i) // i ascends outer loop → already sorted
		}
	}
	return p
}

// paddedFaceBox returns the face's loop-point bounding box inflated by [facePairCullPad] on
// every side.
func paddedFaceBox(f planarFace) math.Box {
	box := math.EmptyBox()
	for _, ring := range f.loops {
		for _, p := range ring {
			box = box.ExtendPoint(p)
		}
	}
	pad := math.Scalar(facePairCullPad)
	box.Min = math.Point3{X: box.Min.X - pad, Y: box.Min.Y - pad, Z: box.Min.Z - pad}
	box.Max = math.Point3{X: box.Max.X + pad, Y: box.Max.Y + pad, Z: box.Max.Z + pad}
	return box
}

// imprintCandidates runs the narrow phase ONCE over the culled candidate pairs, producing both
// the per-face imprint segments (imprintAll's contract) and the tagged provenance
// (provenanceOf's contract) from a single pairImprints call per pair — the duplicate pairing
// the audit flagged (#1607). Pairs iterate (i asc, j asc), matching the retired brute scan, so
// impA/impB/prov are element-for-element identical to it.
func imprintCandidates(fa, fb []planarFace, pairs facePairs) (impA, impB [][][2]math.Point3, prov []imprintSeg) {
	impA = make([][][2]math.Point3, len(fa))
	impB = make([][][2]math.Point3, len(fb))
	for i := range fa {
		for _, j := range pairs.bForA[i] {
			onA, onB := pairImprints(fa[i], fb[j])
			impA[i] = append(impA[i], onA...)
			impB[j] = append(impB[j], onB...)
			prov = appendTagged(prov, onA, fa[i], fb[j])
			prov = appendTagged(prov, onB, fb[j], fa[i])
		}
	}
	return impA, impB, prov
}

// facesAt selects faces by index, preserving order: the coplanar-cover scan sees the same
// first-match order the full scan did, restricted to box-overlap candidates (sound — a cover
// witnesses a point on both faces, so its pair is always a candidate).
func facesAt(faces []planarFace, idx []int) []planarFace {
	out := make([]planarFace, len(idx))
	for k, j := range idx {
		out[k] = faces[j]
	}
	return out
}
