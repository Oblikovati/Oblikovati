// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// TestRemoveTJunctionsBudget guards the CSG-fallback hang fix (M20-F01 #470): T-junction
// removal splits a triangle whose edge carries another vertex when the mesh is small, but a
// mesh larger than tjunctionFaceBudget must BAIL unchanged — rather than run the
// O(faces·verts) cascade-splitting pass that used to hang the boolean on the thousands of
// triangles a faceted curved wall tessellates into.
func TestRemoveTJunctionsBudget(t *testing.T) {
	// Vertex 3 sits on edge 0→1 of triangle (0,1,2): a T-junction that splits 1 tri into 2.
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 2, 0), math.P3(1, 0, 0)}
	lineTol := ResolutionForPoints(verts).Plane()

	if got := removeTJunctions(verts, [][3]int{{0, 1, 2}}, lineTol); len(got) != 2 {
		t.Errorf("under budget: got %d faces, want 2 (the T-junction is split out)", len(got))
	}

	big := make([][3]int, tjunctionFaceBudget+1)
	for i := range big {
		big[i] = [3]int{0, 1, 2}
	}
	if got := removeTJunctions(verts, big, lineTol); len(got) != len(big) {
		t.Errorf("over budget: got %d faces, want %d (must bail unchanged, no scan)", len(got), len(big))
	}
}
