// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestReconstructPassThroughCylinder proves the assembly round-trips a whole solid: a
// cylinder whose every face is passed through must rebuild as a valid closed solid that
// keeps its analytic cylindrical wall (the periodic seam loop survives because the face
// is copied, not traced from facets).
func TestReconstructPassThroughCylinder(t *testing.T) {
	cyl, err := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 3, 10)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	var inputs []ReconInput
	for _, f := range cyl.Faces() {
		inputs = append(inputs, ReconInput{PassThrough: f})
	}
	body := ReconstructBooleanBody(inputs)
	if body == nil || !body.IsSolid() {
		t.Fatal("reconstructed cylinder is not a solid")
	}
	var wall int
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); ok {
			wall++
		}
	}
	if wall != 1 {
		t.Fatalf("reconstructed cylinder has %d analytic walls, want 1", wall)
	}
	if got := len(body.Faces()); got != 3 {
		t.Fatalf("reconstructed cylinder has %d faces, want 3 (wall + 2 caps)", got)
	}
}
