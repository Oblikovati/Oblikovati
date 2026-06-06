// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// TestProjectOntoPlaneUsesPartOrigin: a sketch plane on a face is origined at the part origin
// projected onto the face, not the face surface's parametric origin. A planar surface whose
// origin sits at a rim point (3,0,1) must still project the world origin to the centre (0,0,1)
// — so a sketch on the plane is positioned relative to the part frame, not an arbitrary corner.
// (Regression: the work-plane-on-face origin landed on a rim vertex, placing geometry off the body.)
func TestProjectOntoPlaneUsesPartOrigin(t *testing.T) {
	pl, err := geom.NewPlane(math.P3(3, 0, 1), math.V3(0, 0, 1))
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	got := projectOntoPlane(math.P3(0, 0, 0), pl)
	if !got.IsEqualTo(math.P3(0, 0, 1), 1e-9) {
		t.Errorf("projected origin = %v, want the disc centre (0,0,1)", got)
	}

	// An off-axis world point projects to its foot on the plane (the normal component removed).
	if got := projectOntoPlane(math.P3(2, -1, 9), pl); !got.IsEqualTo(math.P3(2, -1, 1), 1e-9) {
		t.Errorf("projection = %v, want (2,-1,1)", got)
	}
}
