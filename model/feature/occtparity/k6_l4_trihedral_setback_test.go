// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// TestK6L4TrihedralSetbackWatertight is the per-case whole-body gate for the two OCCT tests/blend/simple
// cases the P2 trihedral corner-setback pass greens: K6 and L4, each a box − pocket whose single
// trihedral corner joins THREE CONCAVE fillets at three mutually-orthogonal planar faces. Both were RED
// (K6 +1.19%, L4 +1.38% — the pre-P2 body placed the corner sphere on the MATERIAL side, the reflection
// of the true void sphere through the vertex, twisting the three bands and leaving un-retracted host
// tabs). Flipping the sphere to the VOID side retracts each band by r to the void tangent circle and
// re-trims the three hosts. It asserts — WITHOUT relying on the area-only TestOCCTBlendSimple — that each
// result is a watertight fold-free solid with the void sphere-octant patch and the retracted corner
// station, so a regression that drops the setback (or re-reflects the sphere) fails loud here.
func TestK6L4TrihedralSetbackWatertight(t *testing.T) {
	for _, tc := range []struct {
		name      string
		faces     int
		area      float64     // OCCT checkprops whole-body reference area
		sphereCtr math.Point3 // the VOID-side octant sphere centre (was the material-side reflection)
	}{
		{"K6", 15, 63733.6, math.P3(35, 15, 15)},
		{"L4", 13, 59733.6, math.P3(77.630973, 49.228366, 35)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertWholeBodyFoldFree(t, tc.name, body)
			assertWholeBodyArea(t, tc.name, body, tc.area)
			assertVoidCornerSphere(t, tc.name, body, tc.sphereCtr)
		})
	}
}

// assertVoidCornerSphere checks the body carries exactly one radius-r (=5) corner sphere and that its
// centre is the VOID-side octant point — the crux the P2 flip fixes. A regression that re-reflects the
// sphere to the material side moves the centre by 2·r·√3 and fails here.
func assertVoidCornerSphere(t *testing.T, name string, body *topo.Body, want math.Point3) {
	t.Helper()
	found := 0
	for _, f := range body.Faces() {
		sph, ok := f.Geometry().(geom.Sphere)
		if !ok || sph.Radius > 10 { // skip any large host sphere; the corner blend is r=5
			continue
		}
		found++
		if d := sph.Center.DistanceTo(want); d > 1e-6*100 {
			t.Fatalf("%s corner sphere centre %v, want void-side %v (off by %.4f — sphere re-reflected to the material side?)",
				name, sph.Center, want, d)
		}
	}
	if found != 1 {
		t.Fatalf("%s has %d corner spheres, want exactly 1 void-side octant patch", name, found)
	}
}
