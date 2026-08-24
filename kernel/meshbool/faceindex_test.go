// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "testing"

func TestFaceGridEmptyAndDegenerate(t *testing.T) {
	// Empty mesh: any query returns nothing, no panic.
	if got := newFaceGrid(nil).candidates(faceAABB(tri([3]float64{0, 0, 0}, [3]float64{1, 0, 0}, [3]float64{0, 1, 0}))); len(got) != 0 {
		t.Fatalf("empty grid returned %d candidates, want 0", len(got))
	}
	// Zero-size bounding box (all vertices coincident): cell size must floor, not
	// divide by zero.
	degen := [][3]Point{{pt([3]float64{2, 2, 2}), pt([3]float64{2, 2, 2}), pt([3]float64{2, 2, 2})}}
	g := newFaceGrid(degen)
	if len(g.candidates(faceAABB(degen[0]))) != 1 {
		t.Fatal("degenerate mesh grid did not return its own face")
	}
}

// TestFaceGridNoFalseNegatives is the correctness contract: every face that truly
// overlaps the query box must be among the candidates (the grid may over-report but
// must never miss a real overlap, or the exact test would skip a real pair).
func TestFaceGridNoFalseNegatives(t *testing.T) {
	mesh := boxMesh([3]float64{0, 0, 0}, [3]float64{4, 4, 4})
	grid := newFaceGrid(mesh)

	// A box far outside touches nothing.
	far := aabb{lo: [3]float64{10, 10, 10}, hi: [3]float64{11, 11, 11}}
	if got := grid.candidates(far); len(got) != 0 {
		t.Fatalf("far query returned %d candidates, want 0", len(got))
	}

	// For every face, the brute-force overlap set must be a subset of the grid's
	// candidate set.
	for i := range mesh {
		q := faceAABB(mesh[i])
		cand := map[int]bool{}
		for _, c := range grid.candidates(q) {
			cand[c] = true
		}
		for j := range mesh {
			if faceAABB(mesh[j]).overlaps(q) && !cand[j] {
				t.Fatalf("grid missed overlapping face %d for query face %d", j, i)
			}
		}
	}
}
