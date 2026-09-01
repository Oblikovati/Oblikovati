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
	t.Parallel()
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
			// The unified (u,v)-arrangement trim emits the band as a SEAMLESS annulus — two closed spiric oval
			// edges, no bridging seam (the v-wrapping band's artificial seam folds and cancels). The analytic
			// builder bridged the ovals with a seam (3 edges); the seamless form is equally watertight (#1406).
			if e := len(res.Edges()); e != 2 {
				t.Errorf("two-oval band has %d edges, want 2 (the two ovals, seamless)", e)
			}
		})
	}
}

// At offset EXACTLY R−r the plane is tangent to the inner equator: the two ovals merge into a figure-eight
// pinched at the tangent point. The two-oval band path meshes it (the band's zero-width limit), watertight.
func TestHalfSpaceCutTorusFigureEight(t *testing.T) {
	t.Parallel()
	for _, n := range []math.Vector3{math.V3(0, -1, 0), math.V3(0, 1, 0)} {
		tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
		plane, _ := geom.NewPlane(math.P3(0, 3, 0), n) // |offset| = R−r = 3
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
			t.Errorf("figure-eight cut has %d torus + %d plane faces, want 1 + 2 (band + two touching lids)", tori, planes)
		}
	}
}

// torusAxisParallelFigureEight matches ONLY the degenerate inner-equator tangent (offset = R−r), the one
// spiric case kept analytic; a clean two-oval offset, a single-oval offset, and a perpendicular cut all route
// through the unified trimmer and so must NOT be claimed here (Oblikovati#1406).
func TestTorusAxisParallelFigureEightGuards(t *testing.T) {
	t.Parallel()
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, tc := range []struct {
		name   string
		origin math.Point3
		normal math.Vector3
		want   bool
	}{
		{"figure-eight tangent (offset 3 = R−r)", math.P3(0, 3, 0), math.V3(0, 1, 0), true},
		{"clean two-oval (offset 2 < R−r)", math.P3(0, 2, 0), math.V3(0, 1, 0), false},
		{"single-oval (offset 6 in (R−r,R+r))", math.P3(0, 6, 0), math.V3(0, 1, 0), false},
		{"perpendicular to axis", math.P3(0, 0, 1), math.V3(0, 0, 1), false},
	} {
		plane, _ := geom.NewPlane(tc.origin, tc.normal)
		if got := torusAxisParallelFigureEight(tor, plane); got != tc.want {
			t.Errorf("%s: torusAxisParallelFigureEight = %v, want %v", tc.name, got, tc.want)
		}
	}
}
