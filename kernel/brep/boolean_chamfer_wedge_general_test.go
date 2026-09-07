// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Chamfering the top rim of a revolved stepped shaft cuts a ring WEDGE whose outer wall is the shaft's
// own wall and whose cone crosses that wall exactly at the wedge's own lower rim. The crossing comes
// back from the intersector as a ruled arc — which carries no section plane — so the guard that drops
// an imprint coinciding with a frame edge did not see it, the arrangement carried that rim twice a
// rounding apart, and the cone's boundary came out as ninety alternating fragments that welded to
// nothing (ADR-0061 stage 4).
//
// The result is five faces: the two caps, the taper cone, the shortened wall, and the chamfer cone.
func TestChamferWedgeCutBuildsThroughTheGeneralPath(t *testing.T) {
	t.Parallel()
	// (r, z): a bottom cap, an oblique taper, a straight wall, a top cap — the #1689 meridian.
	shaft, err := SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), []math.Point2{
		math.P2(0, 0), math.P2(0.8, 0), math.P2(1, 1), math.P2(1, 2), math.P2(0, 2),
	}, "shaft")
	if err != nil || shaft == nil {
		t.Fatalf("shaft: %v", err)
	}
	// The chamfer's ring wedge: 0.06 down the wall and 0.06 in along the cap.
	wedge, err := SolidOfRevolution(math.P3(0, 0, 0), math.V3(0, 0, 1), []math.Point2{
		math.P2(1, 1.94), math.P2(1, 2), math.P2(0.94, 2),
	}, "wedge")
	if err != nil || wedge == nil {
		t.Fatalf("wedge: %v", err)
	}
	res, err := Boolean(Difference, shaft, wedge)
	if err != nil {
		t.Fatalf("Boolean(Difference) on the chamfer wedge: %v", err)
	}
	assertWatertight(t, res)
	assertEveryFaceWinds(t, res)
	cones, cyls, planes := 0, 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cones++
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		}
	}
	if cones != 2 || cyls != 1 || planes != 2 {
		t.Errorf("chamfered shaft has %d cones, %d cylinders and %d planes; want 2, 1 and 2 "+
			"(the taper and the chamfer, the shortened wall, the two caps)", cones, cyls, planes)
	}
}
