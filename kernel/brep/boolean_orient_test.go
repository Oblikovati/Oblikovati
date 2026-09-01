// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestReorientFacesMakesConsistentOutward is the unit regression for the boolean orientation
// repair (M20-F01): given a closed shell whose faces are wound inconsistently — the state the
// boolean classification can leave on complex geometry such as a helical coil unioned onto a
// cylinder — reorientFaces must flood-fill a consistent winding (every shared edge traversed in
// opposite directions by its two faces) and flip the whole shell outward (positive volume).
func TestReorientFacesMakesConsistentOutward(t *testing.T) {
	t.Parallel()
	verts := []math.Point3{
		math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0), math.P3(0, 1, 0),
		math.P3(0, 0, 1), math.P3(1, 0, 1), math.P3(1, 1, 1), math.P3(0, 1, 1),
	}
	// The six quad faces of the unit cube, wound HAPHAZARDLY (some CW, some CCW) so adjacent
	// faces disagree across their shared edges — the inconsistency the repair must resolve.
	rings := [][]int{
		{0, 1, 2, 3}, // bottom
		{4, 5, 6, 7}, // top
		{0, 1, 5, 4}, // y=0
		{3, 2, 6, 7}, // y=1
		{0, 3, 7, 4}, // x=0
		{1, 2, 6, 5}, // x=1
	}
	faces := make([]builtFace, len(rings))
	for i, r := range rings {
		faces[i] = builtFace{rings: [][]int{r}, normal: math.V3(0, 0, 1)}
	}

	reorientFaces(faces, verts)

	for e, us := range collectEdgeUses(faces) {
		if len(us) != 2 {
			t.Fatalf("edge %v used %d times, want 2 (a closed shell)", e, len(us))
		}
		if us[0].reversed == us[1].reversed {
			t.Errorf("edge %v still traversed the same way by both faces after reorient", e)
		}
	}
	if v := signedVolume(faces, verts); v <= 0 {
		t.Errorf("signed volume = %g after reorient, want > 0 (outward-facing shell)", v)
	}
}
