// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"testing"
)

func TestForceEdgeQuadFlip(t *testing.T) {
	// A unit square as two CCW triangles sharing diagonal 0-2; force the other
	// diagonal 1-3. The two triangles are the cavity; they flip to share 1-3.
	trg := &Triangulation{
		verts: []Point{pt([3]float64{0, 0, 0}), pt([3]float64{4, 0, 0}), pt([3]float64{4, 4, 0}), pt([3]float64{0, 4, 0})},
		tris:  [][3]int{{0, 1, 2}, {0, 2, 3}},
		axis:  2,
	}
	before := totalDet(trg)
	trg.ForceEdge(1, 3)

	if !trg.edgeExists(1, 3) {
		t.Fatal("forced diagonal 1-3 is not present")
	}
	if trg.edgeExists(0, 2) {
		t.Fatal("old diagonal 0-2 should be gone after the flip")
	}
	if len(trg.tris) != 2 {
		t.Fatalf("got %d triangles, want 2", len(trg.tris))
	}
	assertMeshValid(t, trg, before)
}

func TestForceEdgeCorridor(t *testing.T) {
	// A 3-column strip (8 vertices, 6 triangles) over the rectangle [0,6]x[0,2].
	// The long diagonal 0->7 crosses several internal edges; forcing it exercises a
	// multi-triangle cavity.
	trg := &Triangulation{
		verts: []Point{
			pt([3]float64{0, 0, 0}), pt([3]float64{2, 0, 0}), pt([3]float64{4, 0, 0}), pt([3]float64{6, 0, 0}),
			pt([3]float64{0, 2, 0}), pt([3]float64{2, 2, 0}), pt([3]float64{4, 2, 0}), pt([3]float64{6, 2, 0}),
		},
		tris: [][3]int{
			{0, 1, 5}, {0, 5, 4},
			{1, 2, 6}, {1, 6, 5},
			{2, 3, 7}, {2, 7, 6},
		},
		axis: 2,
	}
	before := totalDet(trg)
	nverts := len(trg.verts)
	trg.ForceEdge(0, 7)

	if !trg.edgeExists(0, 7) {
		t.Fatal("forced long diagonal 0-7 is not present")
	}
	if len(trg.verts) != nverts {
		t.Fatalf("ForceEdge added vertices (%d -> %d); it must not", nverts, len(trg.verts))
	}
	assertMeshValid(t, trg, before)
}

func TestForceEdgeNoOps(t *testing.T) {
	trg := NewTriangulation(tri([3]float64{0, 0, 0}, [3]float64{4, 0, 0}, [3]float64{0, 4, 0}))
	before := totalDet(trg)
	trg.ForceEdge(0, 1) // already an edge
	trg.ForceEdge(2, 2) // same vertex
	if len(trg.tris) != 1 || totalDet(trg).Cmp(before) != 0 {
		t.Fatal("no-op ForceEdge changed the mesh")
	}
}

func TestNewTriangulationNormalizesToCCW(t *testing.T) {
	// A face whose vertices are clockwise in the xy projection must be stored CCW.
	cw := tri([3]float64{0, 0, 0}, [3]float64{0, 4, 0}, [3]float64{4, 0, 0})
	trg := NewTriangulation(cw)
	tt := trg.tris[0]
	if orient2(trg.verts[tt[0]], trg.verts[tt[1]], trg.verts[tt[2]], trg.axis) <= 0 {
		t.Fatal("NewTriangulation did not normalize a clockwise face to CCW storage")
	}
}

// TestForceEdgeDefensiveContracts pins the guards that keep ForceEdge safe on
// malformed input: a bounded (non-hanging) cycle trace, a missing-vertex lookup, a
// two-vertex chain that adds nothing, and a segment that crosses no triangle.
func TestForceEdgeDefensiveContracts(t *testing.T) {
	if indexIn([]int{1, 2, 3}, 9) != -1 {
		t.Fatal("indexIn of a missing vertex must be -1")
	}
	if got := traceCycle(map[int]int{0: 1, 1: 2, 2: 1}, 0); len(got) == 0 || len(got) > 4 {
		t.Fatalf("traceCycle must terminate on a malformed successor map, got %v", got)
	}

	trg := NewTriangulation(tri([3]float64{0, 0, 0}, [3]float64{1, 0, 0}, [3]float64{0, 1, 0}))
	n := len(trg.tris)
	trg.retriangulate([]int{0, 1}) // two-vertex chain: the forced edge is already the boundary
	if len(trg.tris) != n {
		t.Fatal("retriangulate of a two-vertex chain must add no triangles")
	}

	// An orphan vertex whose segment to a corner (along x+y=0) crosses no triangle.
	trg.verts = append(trg.verts, pt([3]float64{10, -10, 0}))
	before := totalDet(trg)
	trg.ForceEdge(3, 0)
	if totalDet(trg).Cmp(before) != 0 || len(trg.tris) != n {
		t.Fatal("ForceEdge with no crossed triangle must be a no-op")
	}
}

// TestForceEdgeDegenerateCavityBails pins the crash-safety guard: if the segment
// crosses a triangle that does not contain its endpoints (a degenerate cavity whose
// boundary omits ui/vi — which pathological near-tangent input can produce),
// ForceEdge must bail without panicking, not index a -1 into the cycle.
func TestForceEdgeDegenerateCavityBails(t *testing.T) {
	trg := &Triangulation{
		verts: []Point{
			pt([3]float64{0, 0, 0}),  // 0 = ui (not in any triangle)
			pt([3]float64{10, 0, 0}), // 1 = vi (not in any triangle)
			pt([3]float64{5, 1, 0}),  // 2
			pt([3]float64{4, -1, 0}), // 3
			pt([3]float64{6, -1, 0}), // 4
		},
		tris: [][3]int{{2, 3, 4}}, // crossed by segment 0->1 but contains neither endpoint
		axis: 2,
	}
	trg.ForceEdge(0, 1) // must not panic
	if len(trg.tris) != 1 {
		t.Fatalf("degenerate-cavity ForceEdge changed the mesh (%d tris); it must bail", len(trg.tris))
	}
}

// assertMeshValid checks the mesh still tiles the face exactly (area unchanged) and
// every triangle is non-degenerate and CCW.
func assertMeshValid(t *testing.T, trg *Triangulation, wantArea *big.Rat) {
	t.Helper()
	if got := totalDet(trg); got.Cmp(wantArea) != 0 {
		t.Fatalf("area not conserved: got %s, want %s", got.RatString(), wantArea.RatString())
	}
	for i, tt := range trg.tris {
		if orient2Val(trg.verts[tt[0]], trg.verts[tt[1]], trg.verts[tt[2]], trg.axis).Sign() <= 0 {
			t.Fatalf("triangle %d is degenerate or clockwise after ForceEdge", i)
		}
	}
}
