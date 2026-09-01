// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// gridMesh builds an n×n grid of unit quads on z=0 (two triangles each) — a mesh big enough to
// exercise the BVH's interior nodes.
func gridMesh(n int) ([]math.Point3, [][3]int32) {
	var pos []math.Point3
	idx := func(i, j int) int32 { return int32(i*(n+1) + j) }
	for i := 0; i <= n; i++ {
		for j := 0; j <= n; j++ {
			pos = append(pos, math.P3(float64(j), float64(i), 0))
		}
	}
	var tris [][3]int32
	for i := range n {
		for j := range n {
			tris = append(tris, [3]int32{idx(i, j), idx(i, j+1), idx(i+1, j)})
			tris = append(tris, [3]int32{idx(i, j+1), idx(i+1, j+1), idx(i+1, j)})
		}
	}
	return pos, tris
}

// bruteNearest is the O(n) reference the BVH must agree with.
func bruteNearest(pos []math.Point3, tris [][3]int32, o math.Point3, d math.Vector3) (int, float64, bool) {
	best, hit := stdmath.Inf(1), -1
	for i, t := range tris {
		if dist, ok := rayTriangleDist(o, d, pos[t[0]], pos[t[1]], pos[t[2]]); ok && dist < best {
			best, hit = dist, i
		}
	}
	return hit, best, hit >= 0
}

// TestMeshRayIndexMatchesBruteForce pins #1776: the BVH returns the same nearest triangle and
// distance as an exhaustive scan, over many rays aimed at a grid mesh — the correctness the
// hover-safe pick depends on.
func TestMeshRayIndexMatchesBruteForce(t *testing.T) {
	t.Parallel()
	pos, tris := gridMesh(16)
	idx := NewMeshRayIndex(pos, tris)
	if idx == nil {
		t.Fatal("NewMeshRayIndex returned nil for a non-empty mesh")
	}
	dir := math.V3(0, 0, -1)
	for gy := range 16 {
		for gx := range 16 {
			o := math.P3(float64(gx)+0.25, float64(gy)+0.25, 10) // straight down onto a known cell
			wantTri, wantT, wantOK := bruteNearest(pos, tris, o, dir)
			gotTri, gotT, gotOK := idx.Nearest(o, dir)
			if gotOK != wantOK || gotTri != wantTri || stdmath.Abs(gotT-wantT) > 1e-9 {
				t.Fatalf("ray at (%d,%d): BVH (%d,%.6f,%v) != brute (%d,%.6f,%v)", gx, gy, gotTri, gotT, gotOK, wantTri, wantT, wantOK)
			}
		}
	}
}

// TestMeshRayIndexMissAndEmpty: a ray that misses the mesh returns ok=false, and an empty mesh
// yields a nil index.
func TestMeshRayIndexMissAndEmpty(t *testing.T) {
	t.Parallel()
	pos, tris := gridMesh(4)
	idx := NewMeshRayIndex(pos, tris)
	if _, _, ok := idx.Nearest(math.P3(-5, -5, 10), math.V3(0, 0, -1)); ok {
		t.Error("a ray beside the grid should not hit")
	}
	if _, _, ok := idx.Nearest(math.P3(1, 1, 10), math.V3(0, 0, 1)); ok {
		t.Error("a ray pointing away from the grid should not hit")
	}
	if NewMeshRayIndex(nil, nil) != nil {
		t.Error("empty mesh should give a nil index")
	}
}
