// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/math"
)

// TestReconstructBooleanPlanarFacesAreOutward: a reconstructed Difference must store every PLANAR
// face with its surface normal pointing OUTWARD (reversed=false) — the convention brep.BooleanDiag
// upholds and downstream consumers rely on (#2247). The Difference cut faces come from the tool and
// would otherwise arrive reversed=true.
func TestReconstructBooleanPlanarFacesAreOutward(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "box")
	if err != nil {
		t.Fatal(err)
	}
	tool, err := brep.SolidBlock(math.P3(1, 1, 1), math.P3(3, 3, 3), "tool") // bites the (2,2,2) corner
	if err != nil {
		t.Fatal(err)
	}
	body, ok := reconstructBoolean(box, tool, meshbool.Difference, DefaultQuality())
	if !ok {
		t.Fatal("reconstructBoolean declined the corner Difference")
	}
	if r := Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("reconstructed body is not a valid solid: %+v", r.Issues)
	}
	for _, f := range body.Faces() {
		if _, planar := f.Geometry().(geom.Plane); planar && f.Reversed() {
			p := f.Geometry().NormalAt(0, 0)
			t.Errorf("planar face is reversed=true (raw surface normal %v points inward); "+
				"reconstruction must canonicalize it to outward", p)
		}
	}
}
