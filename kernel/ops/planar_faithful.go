// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A planar face's triangulation must cover the face exactly: every loop segment (outer + each hole) is
// reproduced as a triangle edge, so the face stitches watertight to the neighbour that shares that
// segment (both discretize the edge identically via discretizeEdge). earcut is a heuristic ear-clipper
// whose self-intersection-cure and brute-force fallback passes do NOT guarantee that on a hard
// multi-hole input — on the EDF duct's top cap (a complex outer loop + two small bore holes) it dropped
// a hole segment and left bridge edges single-sided, producing OVERLAPPING triangles that crack the cap
// against its cylinder/torus neighbours. constrainedDelaunay treats every loop segment as a hard
// constraint and flood-fills the interior, so it cannot drop a boundary edge; with the exact predicates
// (see [orient2d]/[inCircle]) it is also robust on the near-cocircular points a sampled circular hole
// produces. We keep earcut on the hot path and fall back to the CDT only when earcut's triangulated
// AREA disagrees with the true face area — the signature of a real defect. We deliberately do NOT fall
// back on a mere collinear-point merge: earcut's filterPoints drops only EXACTLY-collinear (zero-area)
// boundary points, which leaves the area unchanged and never cracks (a straight edge is sampled the
// same — just two endpoints — by both of its faces). Gating on area keeps the O(n²) CDT off the
// thousands of clean faces (a 390-point cap would otherwise route to it on every recompute) and reserves
// it for the genuine defect.

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
	return planarCDT(outer2D, holes2D)
}

// planarAreaMatches reports whether tris covers exactly the face area (|outer| − Σ|holes|). A clean
// triangulation (incl. one that merged only zero-area collinear points) matches; a defective one that
// overlaps or leaves gaps does not. Empty tris never matches (forces the fallback).
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
	// hurt most (#1610).
	floor := geom.ResolutionForPoints2D(outer2D).Area()
	return stdmath.Abs(got-want) <= 1e-6*want+floor // tol:numeric (relative area fraction)
}

func triArea(a, b, c math.Point2) float64 {
	return stdmath.Abs(float64((b.X-a.X)*(c.Y-a.Y)-(c.X-a.X)*(b.Y-a.Y))) / 2
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
