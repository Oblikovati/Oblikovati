// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane perpendicular to a torus axis trims it to a tube arc: a single analytic torus band capped by a
// planar annulus, watertight, no CSG (the only torus cut with an analytic section, Oblikovati/Oblikovati#1375).
func TestHalfSpaceCutTorusPerpendicularBand(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		origin math.Point3
		normal math.Vector3
	}{
		{"keep below mid-plane", math.P3(0, 0, 0), math.V3(0, 0, 1)},
		{"keep above mid-plane", math.P3(0, 0, 0), math.V3(0, 0, -1)},
		{"keep below off-centre", math.P3(0, 0, 1), math.V3(0, 0, 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tor, err := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
			if err != nil {
				t.Fatalf("SolidTorus: %v", err)
			}
			plane, _ := geom.NewPlane(tc.origin, tc.normal)
			res, err := HalfSpaceCut(tor, plane)
			if err != nil {
				t.Fatalf("HalfSpaceCut: %v", err)
			}
			assertWatertight(t, res)
			tori := 0
			for _, f := range res.Faces() {
				if _, ok := f.Geometry().(geom.Torus); ok {
					tori++
				}
			}
			if tori != 1 {
				t.Errorf("result has %d torus faces, want exactly 1 (the band stays analytic)", tori)
			}
		})
	}
}

// A plane clear of the tube keeps the whole torus or empties it, by which side faces the kept half-space.
func TestHalfSpaceCutTorusClears(t *testing.T) {
	t.Parallel()
	tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	whole, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1)) // z<3 holds the whole torus (tube to z=2)
	res, err := HalfSpaceCut(tor, whole)
	if err != nil {
		t.Fatalf("HalfSpaceCut (clears): %v", err)
	}
	if len(res.Faces()) != 1 {
		t.Errorf("a plane clearing the torus should keep it whole (1 face), got %d", len(res.Faces()))
	}
}
