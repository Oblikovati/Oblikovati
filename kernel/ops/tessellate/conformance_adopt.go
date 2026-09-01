// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// THE ADOPTION CRITERION for the cross-face conformance repair (conformCylConeFaces).
//
// The repair swaps a face's mesh for a boundary-faithful re-mesh so it stops cracking against its
// neighbour. That swap is only a REPAIR when the replacement is at least as FAITHFUL to the face as
// the mesh it replaces — a crack is a hairline T-junction on geometry that is otherwise right, and it
// is strictly better to ship it than to close it with a mesh that no longer describes the face.
//
// Fidelity has TWO independent failure modes, and the criterion must gate BOTH:
//
//  1. FOLDS — the re-mesh creases over itself, double-counting area and destroying the enclosed
//     volume. The metric-(u,v) CDT could fold a partial cone whose crack-repair boundary it cannot
//     triangulate cleanly (I3's two host cones: 30659 fold0 → 59056 fold4).
//
//  2. LOST AREA — the re-mesh tiles LESS of the face than the mesh it replaces. Measured across the
//     whole OCCT blend-parity corpus (151 adoption decisions on the 20 cases that run the repair),
//     NINE adoptions destroyed area while passing the fold-only gate, in two distinct ways:
//     • the CDT did not COVER its own parametric domain — complex/D8's r=24 corner cylinder came out
//     38.9% short in (u,v) (3392.39 → 2003.66 in 3D), complex/F2's walls 34%/39% short,
//     simple/Y4's plane 26.5% short;
//     • the CDT covered the domain EXACTLY yet the 3D realisation chorded across the surface's
//     curvature — complex/D8's r=30 fillet band covers 23340 of 23340 in metric (u,v) but ships
//     21339.8 in 3D (−8.57%), because a BOUNDARY-ONLY triangulation has no interior node between
//     the band's two straight axial rulings, so a single triangle spans the full 90° arc and
//     realises it as its chord (2sin45°/(π/2) = 0.900, i.e. exactly the 8.57% seen).
//     That single face was the whole of complex/D8's −0.90% shipped-vs-closed-form area gap.
//
// The area condition is a strict do-no-harm, not a threshold to tune: both meshes inscribe the SAME
// trimmed face (every vertex is an exact point ON the surface) and neither may fold, so both
// under-report the face's true area and the one with MORE area is the one closer to it. A re-mesh may
// therefore be adopted when it GAINS area (simple/Q5's wall gains 16.6%, closing 74% of its gap to
// the exact 8.036e6 metric-(u,v) area) but never when it loses it.
//
// "Never loses it" needs one derived allowance, because the repair legitimately CHANGES the boundary
// polygon: it restores the near-collinear boundary points the absorbing mesher dropped. A point the
// mesher was entitled to drop lies within the chordal tolerance q.Tol() of the segment it splits, so
// restoring it moves the enclosed area by at most ½·q.Tol()·(that segment's length); summed over the
// whole boundary that is bounded by q.Tol() × the face's boundary length. Nothing else about a
// faithful re-mesh may move the area. The bound scales with the model (a length × a tolerance, both
// in model units), so a µm part and a 1000× fixture get the same test — and it is provably not fitted
// to the corpus: every one of the 141 faithful adoptions sits ≥10 DECADES inside it (worst |loss| /
// bound = 2e-11, i.e. float round-off), while all nine defective ones exceed it by 70×–5000×.

// conformingMeshIsFaithful reports whether the conformance re-mesh m may replace old on face f: it
// must add no fold edges and lose no surface area beyond conformAreaSlack. See the file comment for
// the derivation; a re-mesh that fails either test is dropped and the crack is left as a hairline.
//
// Example: conformingMeshIsFaithful(remesh, current, face, PropertyQuality()) == false when the
// boundary-only CDT chords across a cylinder it has no interior node to follow.
func conformingMeshIsFaithful(m, old *Mesh, f *topo.Face, q Quality) bool {
	if validate.FoldEdgeCount(m) > validate.FoldEdgeCount(old) {
		return false
	}
	return validate.MeshArea(m) >= validate.MeshArea(old)-conformAreaSlack(f, q)
}

// conformAreaSlack is the most area a FAITHFUL conformance re-mesh of f can lose: the chordal
// tolerance times the face's own discretized boundary length, the bound on how far restoring the
// dropped near-collinear boundary points can move the enclosed area (see the file comment).
func conformAreaSlack(f *topo.Face, q Quality) float64 {
	return q.Tol() * faceBoundaryLength(f, q)
}

// faceBoundaryLength is the total length of f's discretized boundary rings — outer plus every hole —
// at quality q, the same polylines the conforming meshers triangulate between.
func faceBoundaryLength(f *topo.Face, q Quality) float64 {
	total := ringLength(FaceOuterBoundary(f, q))
	for _, h := range faceHoleBoundaries(f, q) {
		total += ringLength(h)
	}
	return total
}

// ringLength is the perimeter of a closed point ring (the segment back from the last point to the
// first included).
func ringLength(ring []math.Point3) float64 {
	total := 0.0
	for i := range ring {
		total += float64(ring[i].DistanceTo(ring[(i+1)%len(ring)]))
	}
	return total
}
