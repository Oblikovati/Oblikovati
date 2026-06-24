// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// TestRemoveTJunctionsClosesCage guards acceptance #3 of the curved-boolean umbrella
// (Oblikovati/Oblikovati#1336, #1320 #3): T-junction removal must close the cage with NO face budget.
// The old M20-F01 #470 pass bailed above tjunctionFaceBudget and returned the mesh unchanged (open);
// the single-pass centroid fan eliminates every T-junction regardless of size. A vertex sitting on a
// triangle's edge is split out; a vertex off the edges leaves the triangle alone.
func TestRemoveTJunctionsClosesCage(t *testing.T) {
	// Vertex 3 sits on edge 0→1 of triangle (0,1,2): a T-junction. The split must use vertex 3, so the
	// triangle becomes a fan (>1 face) and every undirected edge ends up shared an even number of times.
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 2, 0), math.P3(1, 0, 0)}
	lineTol := ResolutionForPoints(verts).Plane()

	_, faces := removeTJunctions(verts, [][3]int{{0, 1, 2}}, lineTol)
	if len(faces) < 2 {
		t.Errorf("T-junction at vertex 3 not split: got %d faces, want ≥2", len(faces))
	}
	if uses := edgeUseCounts(faces); uses[edgeKeyOf(0, 3)] == 0 || uses[edgeKeyOf(3, 1)] == 0 {
		t.Errorf("split did not subdivide edge 0→1 at the on-edge vertex 3: edge uses %v", uses)
	}
}

// TestRemoveTJunctionsNoFalseSplit: a vertex clear of every edge leaves the triangle whole.
func TestRemoveTJunctionsNoFalseSplit(t *testing.T) {
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 2, 0), math.P3(3, 3, 0)}
	lineTol := ResolutionForPoints(verts).Plane()

	_, faces := removeTJunctions(verts, [][3]int{{0, 1, 2}}, lineTol)
	if len(faces) != 1 {
		t.Errorf("triangle with no on-edge vertex changed: got %d faces, want 1", len(faces))
	}
}

// TestRemoveTJunctionsScalesPastOldBudget: a T-junction inside a mesh larger than the old 4000-face budget
// must STILL be split. The old M20-F01 #470 pass bailed unchanged above the budget — leaving the cage open
// — so this is the regression that proves the budget is gone (#1336 #3). The triangle (0,1,2) carries an
// on-edge vertex 3; thousands of small, disjoint pad triangles push the count over the old cap without
// touching it.
func TestRemoveTJunctionsScalesPastOldBudget(t *testing.T) {
	tj := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 2, 0), math.P3(1, 0, 0)}
	lineTol := ResolutionForPoints(tj).Plane() // small-scale tolerance, before the far pad blows up the bbox

	verts := append([]math.Point3{}, tj...)
	faces := [][3]int{{0, 1, 2}}
	for i := 0; i < tjunctionPadCount; i++ {
		b, x := len(verts), float64(100+i)
		verts = append(verts, math.P3(x, 0, 0), math.P3(x+1, 0, 0), math.P3(x, 1, 0))
		faces = append(faces, [3]int{b, b + 1, b + 2})
	}

	_, out := removeTJunctions(verts, faces, lineTol)
	uses := edgeUseCounts(out)
	if uses[edgeKeyOf(0, 3)] == 0 || uses[edgeKeyOf(3, 1)] == 0 {
		t.Fatalf("T-junction in a %d-face mesh not split (old budget would have bailed): edge uses around vertex 3 = %d, %d",
			len(faces), uses[edgeKeyOf(0, 3)], uses[edgeKeyOf(3, 1)])
	}
}

// tjunctionPadCount pushes the test mesh past the retired 4000-face budget.
const tjunctionPadCount = 4001
