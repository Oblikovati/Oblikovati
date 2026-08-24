// SPDX-License-Identifier: GPL-2.0-only

package renderer

import "testing"

// scatteredTriangles returns n non-degenerate, non-overlapping unit triangles spread
// along the x axis — enough to force BuildBVH to split repeatedly.
func scatteredTriangles(n int) []Triangle {
	tris := make([]Triangle, n)
	for i := range tris {
		x := float32(i) * 2
		tris[i] = Triangle{
			V0: [3]float32{x, 0, 0}, V1: [3]float32{x + 1, 0, 0}, V2: [3]float32{x, 1, 0},
			InstanceID: 1, PrimitiveID: uint32(i),
		}
	}
	return tris
}

func TestBuildBVHEmptyIsNil(t *testing.T) {
	if got := BuildBVH(nil); got != nil {
		t.Errorf("BuildBVH(nil) = %+v, want nil", got)
	}
}

// TestBuildBVHTriangleOrderIsAPermutation checks every original triangle index appears
// exactly once across TriangleOrder — the BVH must partition, never drop or duplicate.
func TestBuildBVHTriangleOrderIsAPermutation(t *testing.T) {
	tris := scatteredTriangles(37) // an odd, non-power-of-two count exercises uneven splits
	bvh := BuildBVH(tris)
	seen := make([]bool, len(tris))
	for _, idx := range bvh.TriangleOrder {
		if idx < 0 || int(idx) >= len(tris) {
			t.Fatalf("TriangleOrder has out-of-range index %d", idx)
		}
		if seen[idx] {
			t.Fatalf("triangle %d appears more than once in TriangleOrder", idx)
		}
		seen[idx] = true
	}
	for i, ok := range seen {
		if !ok {
			t.Errorf("triangle %d missing from TriangleOrder", i)
		}
	}
}

// TestBuildBVHLeafBoundsContainTheirTriangles checks every leaf's AABB actually contains
// every vertex of every triangle assigned to it.
func TestBuildBVHLeafBoundsContainTheirTriangles(t *testing.T) {
	tris := scatteredTriangles(37)
	bvh := BuildBVH(tris)
	for _, n := range bvh.Nodes {
		if n.TriCount == 0 {
			continue // internal node
		}
		for _, triIdx := range bvh.TriangleOrder[n.LeftFirst : n.LeftFirst+n.TriCount] {
			for _, v := range [3][3]float32{tris[triIdx].V0, tris[triIdx].V1, tris[triIdx].V2} {
				for a := 0; a < 3; a++ {
					if v[a] < n.Min[a]-1e-6 || v[a] > n.Max[a]+1e-6 {
						t.Fatalf("leaf bounds [%v,%v] do not contain vertex %v (axis %d) of triangle %d", n.Min, n.Max, v, a, triIdx)
					}
				}
			}
		}
	}
}

// TestBuildBVHInternalBoundsContainChildren checks the defining BVH property: every
// internal node's AABB contains both of its children's AABBs.
func TestBuildBVHInternalBoundsContainChildren(t *testing.T) {
	tris := scatteredTriangles(37)
	bvh := BuildBVH(tris)
	for i, n := range bvh.Nodes {
		if n.TriCount != 0 {
			continue // leaf
		}
		left := bvh.Nodes[n.LeftFirst]
		right := bvh.Nodes[n.LeftFirst+1]
		for _, child := range []BVHNode{left, right} {
			for a := 0; a < 3; a++ {
				if child.Min[a] < n.Min[a]-1e-6 || child.Max[a] > n.Max[a]+1e-6 {
					t.Errorf("node %d bounds [%v,%v] do not contain child bounds [%v,%v] (axis %d)",
						i, n.Min, n.Max, child.Min, child.Max, a)
				}
			}
		}
	}
}

// TestBuildBVHLeavesRespectThreshold checks no leaf exceeds bvhLeafThreshold triangles.
func TestBuildBVHLeavesRespectThreshold(t *testing.T) {
	tris := scatteredTriangles(37)
	bvh := BuildBVH(tris)
	for _, n := range bvh.Nodes {
		if n.TriCount > bvhLeafThreshold {
			t.Errorf("leaf has %d triangles, want ≤ %d", n.TriCount, bvhLeafThreshold)
		}
	}
}

// TestBuildBVHSmallSceneIsSingleLeaf checks a scene at or below the threshold builds a
// single-node tree (no unnecessary splitting).
func TestBuildBVHSmallSceneIsSingleLeaf(t *testing.T) {
	bvh := BuildBVH(scatteredTriangles(bvhLeafThreshold))
	if len(bvh.Nodes) != 1 {
		t.Fatalf("len(Nodes) = %d, want 1 (single root leaf)", len(bvh.Nodes))
	}
	if bvh.Nodes[0].TriCount != bvhLeafThreshold {
		t.Errorf("root TriCount = %d, want %d", bvh.Nodes[0].TriCount, bvhLeafThreshold)
	}
}
