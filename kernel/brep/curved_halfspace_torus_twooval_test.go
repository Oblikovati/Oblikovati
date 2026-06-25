// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane PARALLEL to the torus axis but offset INSIDE the inner tube radius passes through the central
// hole and cuts BOTH tube walls: the section is two ovals, and the kept solid is one tube-wrapping torus
// band closed by two planar oval-disk lids, watertight, no CSG (Oblikovati/Oblikovati#1375).
func TestHalfSpaceCutTorusTwoOvalBand(t *testing.T) {
	for _, tc := range []struct {
		name   string
		normal math.Vector3
	}{
		{"keep +y band", math.V3(0, -1, 0)}, // keep y≥2
		{"keep −y band", math.V3(0, 1, 0)},  // keep y≤2 (the wider band)
	} {
		t.Run(tc.name, func(t *testing.T) {
			tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
			plane, _ := geom.NewPlane(math.P3(0, 2, 0), tc.normal) // |offset|=2 < R−r=3 → two ovals
			res, err := HalfSpaceCut(tor, plane)
			if err != nil {
				t.Fatalf("HalfSpaceCut: %v", err)
			}
			assertWatertight(t, res)
			tori, planes := 0, 0
			for _, f := range res.Faces() {
				switch f.Geometry().(type) {
				case geom.Torus:
					tori++
				case geom.Plane:
					planes++
				}
			}
			if tori != 1 || planes != 2 {
				t.Errorf("two-oval band has %d torus + %d plane faces, want 1 + 2 (the band + two oval-disk lids)", tori, planes)
			}
			// Two ovals (each a closed spiric edge) + one bridging seam.
			if e := len(res.Edges()); e != 3 {
				t.Errorf("two-oval band has %d edges, want 3 (two ovals + a seam)", e)
			}
		})
	}
}

// torusTwoOvalBand admits only the axis-parallel offset strictly inside the inner tube radius; a single-oval
// offset (≥ R−r) or a perpendicular cut belongs elsewhere.
func TestTorusTwoOvalBandGuards(t *testing.T) {
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, tc := range []struct {
		name   string
		origin math.Point3
		normal math.Vector3
		want   bool
	}{
		{"two-oval (offset 2 < R−r)", math.P3(0, 2, 0), math.V3(0, 1, 0), true},
		{"single-oval (offset 6 in (R−r,R+r))", math.P3(0, 6, 0), math.V3(0, 1, 0), false},
		{"perpendicular to axis", math.P3(0, 0, 1), math.V3(0, 0, 1), false},
		{"through the axis (offset 0)", math.P3(0, 0, 0), math.V3(0, 1, 0), false},
	} {
		plane, _ := geom.NewPlane(tc.origin, tc.normal)
		if got := torusTwoOvalBand(tor, plane); got != tc.want {
			t.Errorf("%s: torusTwoOvalBand = %v, want %v", tc.name, got, tc.want)
		}
	}
}
