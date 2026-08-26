// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// gridTriangles returns an n×n grid of unit cells (2 triangles each) on the plane z=zPlane — a stand-in
// for a tessellated face dense enough to make the old O(Tₐ·T_b) inner loop hurt.
func gridTriangles(n int, zPlane float64) [][3]math.Point3 {
	z := math.Scalar(zPlane)
	tris := make([][3]math.Point3, 0, 2*n*n)
	for i := range n {
		for j := range n {
			x0, y0 := math.Scalar(i), math.Scalar(j)
			x1, y1 := x0+1, y0+1
			a := math.Point3{X: x0, Y: y0, Z: z}
			b := math.Point3{X: x1, Y: y0, Z: z}
			c := math.Point3{X: x1, Y: y1, Z: z}
			d := math.Point3{X: x0, Y: y1, Z: z}
			tris = append(tris, [3]math.Point3{a, b, c}, [3]math.Point3{a, c, d})
		}
	}
	return tris
}

// TestSelfIntersectBVHPrunesTriangleTests is acceptance criterion 1 of #1411: querying mesh A against a
// BVH over mesh B touches far fewer candidate triangles than the Tₐ·T_b all-pairs scan. Two parallel
// grids offset in z share no overlapping triangle boxes, so the BVH yields essentially nothing while
// the old loop would still run every pair.
func TestSelfIntersectBVHPrunesTriangleTests(t *testing.T) {
	a := gridTriangles(24, 0)
	b := gridTriangles(24, 5) // disjoint in z
	bvh := newTriBVH(b)

	candidates := 0
	for _, t1 := range a {
		box := math.BoxFromPoints(t1[0], t1[1], t1[2])
		bvh.query(box, func(int) bool { candidates++; return false })
	}
	naive := len(a) * len(b)
	if candidates*100 > naive {
		t.Errorf("BVH yielded %d candidate triangle tests; want << the %d all-pairs scan", candidates, naive)
	}
	t.Logf("disjoint grids: BVH %d candidate tests vs %d all-pairs (Tₐ=%d, T_b=%d)", candidates, naive, len(a), len(b))
}

// TestSelfIntersectBVHFindsRealCrossing is acceptance criterion 2 (no false negative): a triangle of A
// that genuinely pierces B is still reported through the BVH query, with a witness on the crossing.
func TestSelfIntersectBVHFindsRealCrossing(t *testing.T) {
	b := gridTriangles(10, 0) // a sheet on z=0 spanning x,y ∈ [0,10]
	bvh := newTriBVH(b)
	// A vertical triangle straddling z=0 inside the sheet's extent must cross it.
	crossing := [3]math.Point3{
		{X: 3, Y: 3, Z: -1}, {X: 6, Y: 3, Z: -1}, {X: 4.5, Y: 3.5, Z: 1},
	}
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{crossing}, bvh, nil, 1e-6, 0); !hit {
		t.Error("a triangle piercing the sheet was not detected through the BVH (false negative)")
	}
}

// TestSelfIntersectBVHIgnoresClearance is acceptance criterion 2 (no false positive): a triangle well
// clear of B reports nothing.
func TestSelfIntersectBVHIgnoresClearance(t *testing.T) {
	b := gridTriangles(10, 0)
	bvh := newTriBVH(b)
	clear := [3]math.Point3{{X: 3, Y: 3, Z: 5}, {X: 6, Y: 3, Z: 5}, {X: 4.5, Y: 3.5, Z: 6}}
	if _, hit := meshCrossesOffBoundary([][3]math.Point3{clear}, bvh, nil, 1e-6, 0); hit {
		t.Error("a triangle clear of the sheet was reported as crossing (false positive)")
	}
}

// TestSelfIntersectBVHEmpty covers the degenerate empty mesh: a BVH over no triangles yields no
// candidates and does not panic (a face that tessellated to nothing).
func TestSelfIntersectBVHEmpty(t *testing.T) {
	bvh := newTriBVH(nil)
	called := false
	bvh.query(math.BoxFromPoints(math.Point3{}, math.Point3{X: 1, Y: 1, Z: 1}), func(int) bool { called = true; return true })
	if called {
		t.Error("an empty BVH should yield no candidate triangles")
	}
}

// BenchmarkSelfIntersectBVH records the broad-phase cost on two dense disjoint grids — the #1411 win
// over the all-pairs scan (Tₐ·T_b ≈ 1.3M triangle tests here, the BVH a small multiple of Tₐ+T_b).
func BenchmarkSelfIntersectBVH(b *testing.B) {
	aTris, bTris := gridTriangles(26, 0), gridTriangles(26, 5)
	for i := 0; i < b.N; i++ {
		bvh := newTriBVH(bTris)
		meshCrossesOffBoundary(aTris, bvh, nil, 1e-6, 0)
	}
}
