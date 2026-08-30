// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A planar face's triangulation must COVER the face exactly: no overlap, no gap, over exactly
// (outer − Σ holes). That is the watertightness the face needs to stitch to its neighbours, and AREA
// is the faithful proxy for it — a genuine coverage defect (overlapping or gapped triangles) shifts
// the triangulated area off the true face area, so we keep earcut on the hot path and fall back to the
// CDT only when earcut's area disagrees. earcut is a heuristic ear-clipper whose self-intersection-cure
// and brute-force fallback passes do NOT guarantee coverage on a hard multi-hole input — on the EDF
// duct's top cap (a complex outer loop + two small bore holes) it dropped a hole segment and left
// bridge edges single-sided, producing overlapping triangles that both crack the cap AND throw its area
// off. constrainedDelaunay treats every loop segment as a hard constraint and flood-fills the interior,
// so it cannot drop a boundary edge; with the exact predicates (see [orient2d]/[inCircle]) it is also
// robust on the near-cocircular points a sampled circular hole produces.
//
// AREA IS A PROXY, NOT THE LITERAL "every loop segment is a triangle edge" — and that distinction is
// deliberate, because two area-CONSERVING ways a boundary segment is legitimately NOT its own triangle
// edge must NOT trip the fallback:
//   - COLLINEAR SUBSUMPTION. earcut fans a straight strip with ONE long edge that runs along several
//     collinear boundary vertices — e.g. a plate with three in-line rectangular slots triangulates the
//     material below them with a single edge spanning all three slot bottoms, so each slot-bottom
//     segment is subsumed, not reproduced. This is watertight: the edge is collinear with those
//     segments, so the surface is flat and continuous, and the neighbour's extra vertices lie exactly
//     ON that edge (a coplanar T-vertex, no gap). removeTJunctions splits it out only where the WELDED
//     CSG cage needs a closed 2-manifold; the display mesh does not.
//   - SHARED-VOID EDGES. a segment shared by two loops (two abutting holes, or a hole touching the
//     outer) borders no material on either side, so no triangle need use it (#37's dog-bone slot).
// A literal per-segment guard would FALSE-POSITIVE on both — even three disjoint slots — and route
// thousands of clean faces to the O(n²) CDT (a 390-point cap on every recompute). Gating on area
// admits both benign cases and still catches the genuine coverage defect, which is the one that counts.
// (The theoretical hole — an overlap and a gap that cancel to the same total area — was searched for
// and not produced on realistic multi-hole faces; if one is ever found, the fix is a coverage check
// here, NOT a per-segment one.)

// planarTris triangulates a planar face's projected boundary. Indices address outer2D[i] for
// i<len(outer2D), then the holes concatenated after it — the same vertex ordering earcut uses, so
// callers build one 3D vertex buffer in that order regardless of which triangulator produced the
// indices.
//
// The fallback used to be capped at 256 boundary vertices because the CDT's flip-based
// constraint recovery was worse than O(n²) (~1.2 s at ~650 verts) — above the cap an
// area-WRONG earcut shipped silently, the exact defect class the repo's tessellation
// rule ranks above everything else. #1409's corridor-walk segment insertion removed the
// quadratic recovery, so the cap is retired (#1610): a mismatched face now always routes
// to the CDT, whatever its size.
func planarTris(outer2D []math.Point2, holes2D [][]math.Point2) [][3]int {
	var tris [][3]int
	if len(holes2D) == 0 {
		tris = bestSingleLoopTriangulation(outer2D)
	} else {
		tris = earcut(outer2D, holes2D)
	}
	if planarAreaMatches(tris, outer2D, holes2D) {
		return tris
	}
	// A self-intersecting (non-simple) boundary has no correct triangulation, and feeding it to the
	// CDT makes every boundary constraint fall into the O(T) flip-recovery fallback — O(n·T²), seconds
	// on a ~250-vertex face — which, on the synchronous per-frame pick path, starves the frame-loop
	// dispatcher an async add-in build waits on (a transient partially-constrained face gets revolved
	// and hover-picked mid-build). Such input only appears transiently; a bounded best-effort mesh is
	// fine (the real geometry replaces it next frame). The CDT's own recoverFlipWork budget backstops
	// the non-crossing degeneracies (collinear overlap, coincident vertices) this cheap check misses.
	if loopsSelfCross(outer2D, holes2D) {
		return tris
	}
	return planarCDT(outer2D, holes2D)
}

// loopsSelfCross reports whether any two boundary segments of the loops (outer + holes) PROPERLY
// cross — the defining signature of a non-simple boundary. It reuses the exact-predicate segmentsCross,
// which returns false for segments that merely share an endpoint (adjacent edges) or touch, so only a
// genuine transversal crossing trips it (no false positive on a valid simple boundary, whose faceting
// stays simple). O(n²) with early exit, run only on the already-off area-mismatch path where n is one
// face's facet count (<~150) — a few milliseconds worst case, and typically an immediate hit.
func loopsSelfCross(outer2D []math.Point2, holes2D [][]math.Point2) bool {
	var segs [][2][2]float64
	appendLoop := func(loop []math.Point2) {
		n := len(loop)
		for i := range n {
			p, q := loop[i], loop[(i+1)%n]
			segs = append(segs, [2][2]float64{{float64(p.X), float64(p.Y)}, {float64(q.X), float64(q.Y)}})
		}
	}
	appendLoop(outer2D)
	for _, h := range holes2D {
		appendLoop(h)
	}
	for i := 0; i < len(segs); i++ {
		for j := i + 1; j < len(segs); j++ {
			if segmentsCross(segs[i][0], segs[i][1], segs[j][0], segs[j][1]) {
				return true
			}
		}
	}
	return false
}

// planarAreaMatches reports whether tris covers exactly the face area (|outer| − Σ|holes|) — the
// coverage proxy the package comment justifies. A clean triangulation matches, INCLUDING the two
// area-conserving cases where a boundary segment is legitimately not its own triangle edge (collinear
// subsumption and shared-void edges); a defective one that overlaps or gaps does not. Empty tris never
// matches (forces the fallback).
func planarAreaMatches(tris [][3]int, outer2D []math.Point2, holes2D [][]math.Point2) bool {
	if len(tris) == 0 {
		return false
	}
	want := stdmath.Abs(signedArea(outer2D))
	for _, h := range holes2D {
		want -= stdmath.Abs(signedArea(h))
	}
	verts := append([]math.Point2(nil), outer2D...)
	for _, h := range holes2D {
		verts = append(verts, h...)
	}
	var got float64
	for _, t := range tris {
		got += triArea(verts[t[0]], verts[t[1]], verts[t[2]])
	}
	// Relative area bracket with a model-relative floor: the old absolute 1e-9 floor was
	// ~10% of a µm-scale face's area, neutering the defect check exactly where cracks
	// hurt most (#1610). The predicate itself is shared with the CDT's own coverage guard
	// (cdt_coverage.go), so the two cannot drift apart.
	return coverageAreaMatches(got, want, geom.ResolutionForPoints2D(outer2D).Area())
}

func triArea(a, b, c math.Point2) float64 {
	return stdmath.Abs(float64((b.X-a.X)*(c.Y-a.Y)-(c.X-a.X)*(b.Y-a.Y))) / 2
}

// recordUncoveredPlanarFace flags a planar face mesh whose triangulation does not cover the face's
// own domain — the same area coverage predicate planarTris gates on, run once more on the SHIPPED
// tris. planarTris returns a bounded best-effort covering for a transient non-simple boundary
// (loopsSelfCross) rather than blocking the pick path, but that covering is area-wrong: an overlap
// or a gap. Before #3388 it shipped with no signal, so a consumer (render, mass properties, export)
// saw a clean face where the mesh has a hole. The mesh still ships — a flagged partial beats a
// missing face — but the shortfall now travels with it as a Defect. A covering triangulation is a
// no-op. Reuses CodeTessellateDomainUncovered (the conformance path's code).
func recordUncoveredPlanarFace(m *Mesh, tris [][3]int, outer2D []math.Point2, holes2D [][]math.Point2) {
	if m == nil || planarAreaMatches(tris, outer2D, holes2D) {
		return
	}
	m.Diagnose(diag.Diagnostic{
		Code:     CodeTessellateDomainUncovered,
		Severity: diag.Defect,
		Detail: fmt.Sprintf("planar face triangulation (%d triangles over a %d-vertex outer + %d hole loop(s)) does not "+
			"cover the face area; the trim boundary self-crosses and the mesh carries a hole or overlap", len(tris), len(outer2D), len(holes2D)),
	})
}

// planarCDT re-triangulates the projected boundary with the constrained Delaunay, the manifold-
// guaranteeing fallback. It feeds constrainedDelaunay the SAME projected coordinates earcut used (not a
// surface-(u,v) re-projection), so the boundary points are bit-identical to the neighbour's — the
// reason this conforms instead of cascading.
func planarCDT(outer2D []math.Point2, holes2D [][]math.Point2) [][3]int {
	pts := make([][2]float64, 0, len(outer2D))
	for _, p := range outer2D {
		pts = append(pts, [2]float64{p.X, p.Y})
	}
	loops := [][]int{rangeIndices(0, len(outer2D))}
	off := len(outer2D)
	for _, h := range holes2D {
		for _, p := range h {
			pts = append(pts, [2]float64{p.X, p.Y})
		}
		loops = append(loops, rangeIndices(off, off+len(h)))
		off += len(h)
	}
	return constrainedDelaunay(pts, loops)
}

func rangeIndices(lo, hi int) []int {
	out := make([]int, hi-lo)
	for i := range out {
		out[i] = lo + i
	}
	return out
}
