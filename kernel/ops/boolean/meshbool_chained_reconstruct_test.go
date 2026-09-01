// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	"testing"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/meshbool"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/math"
)

// TestChainedReconstructionCutStaysValid is the #2247 chained-reconstruction regression: cutting a
// RECONSTRUCTED body with a tool flush against a face the first cut created must stay a valid solid.
// The second cut's floor face sits at the same z as the first cut's step floor; the meshbool point-in-
// solid classification used to reject every ray from that coplanar-feature height and drop the cut
// face, leaving an open body (fixed in kernel/meshbool raycast: a coplanar-but-outside ray endpoint is
// a clean miss, not a degeneracy).
func TestChainedReconstructionCutStaysValid(t *testing.T) {
	t.Parallel()
	box, err := brep.SolidBlock(math.P3(0, 0, 0), math.P3(4, 2, 2), "box")
	if err != nil {
		t.Fatal(err)
	}
	tool1, _ := brep.SolidBlock(math.P3(3, 0, 1), math.P3(4, 2, 3), "t1") // step cut: creates faces at x=3 and z=1
	target1, ok := reconstructBoolean(box, tool1, meshbool.Difference, DefaultQuality())
	if !ok || !Validate(target1).ValidSolid() {
		t.Fatalf("first cut on a fresh box must reconstruct to a valid solid (ok=%v)", ok)
	}

	tool2, _ := brep.SolidBlock(math.P3(2, 0, 1), math.P3(3, 2, 3), "t2") // flush against cut1's x=3 wall and z=1 floor
	target2, ok := reconstructBoolean(target1, tool2, meshbool.Difference, DefaultQuality())
	if !ok {
		t.Fatal("second cut on the RECONSTRUCTED body declined (the chained-reconstruction raycast bug)")
	}
	if r := Validate(target2); !r.ValidSolid() {
		t.Fatalf("chained cut is not a valid solid: manifold=%v closed=%v euler=%v issues=%v", r.Manifold, r.Closed, r.EulerConsistent, r.Issues)
	}
	// The combined cut leaves an L-prism of volume 4*2*2 - 2*2*1 = 12.
	if v := query.BodyGeometryProperties(target2, PropertyQuality()).Volume; v < 11.9 || v > 12.1 {
		t.Errorf("chained cut volume = %.3f, want ~12", v)
	}
}
