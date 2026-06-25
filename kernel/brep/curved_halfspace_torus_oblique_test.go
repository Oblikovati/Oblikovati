// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane PARALLEL to a torus axis, offset between the inner and outer tube radii, slices off a single
// outer-tube oval cap: one analytic torus face (the surface inside the spiric oval) plus one planar oval
// lid, watertight, no CSG — the first exact OBLIQUE torus cut (Oblikovati/Oblikovati#1375).
func TestHalfSpaceCutTorusAxisParallelOvalCap(t *testing.T) {
	tor, err := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	if err != nil {
		t.Fatalf("SolidTorus: %v", err)
	}
	// Keep the cap y≥6: HalfSpaceCut keeps the negative side, so the cut normal points −y (into the torus).
	plane, _ := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, -1, 0))
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
	if tori != 1 || planes != 1 {
		t.Errorf("cap has %d torus + %d plane faces, want exactly 1 + 1 (analytic cap + oval lid)", tori, planes)
	}
	// The cap is one bigon loop (two spiric branches meeting at the two oval pinch vertices).
	if e := len(res.Edges()); e != 2 {
		t.Errorf("cap has %d edges, want 2 (the two spiric branches)", e)
	}
	if vtx := len(res.Vertices()); vtx != 2 {
		t.Errorf("cap has %d vertices, want 2 (the oval's v-extreme pinches)", vtx)
	}
}

// torusSingleOvalCap admits only the axis-parallel, kept-cap, single-oval geometry; every other plane
// (perpendicular, the big-complement side, a clearing offset) must defer so it falls to the band path or CSG.
func TestTorusSingleOvalCapGuards(t *testing.T) {
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, tc := range []struct {
		name   string
		origin math.Point3
		normal math.Vector3
		want   bool
	}{
		{"axis-∥ cap (kept side)", math.P3(0, 6, 0), math.V3(0, -1, 0), true},
		{"perpendicular to axis", math.P3(0, 0, 1), math.V3(0, 0, 1), false},
		{"big-complement side (K≥0)", math.P3(0, 6, 0), math.V3(0, 1, 0), false},
		{"clears the tube (offset > R+r)", math.P3(0, 8, 0), math.V3(0, -1, 0), false},
		{"two-oval offset (< R−r)", math.P3(0, 2, 0), math.V3(0, -1, 0), false},
	} {
		plane, _ := geom.NewPlane(tc.origin, tc.normal)
		if got := torusSingleOvalCap(tor, plane); got != tc.want {
			t.Errorf("%s: torusSingleOvalCap = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// clampUnitF folds arccos arguments into [−1, 1] (the oval's v-extremes can nudge past by FP error).
func TestClampUnitF(t *testing.T) {
	for _, tc := range []struct{ in, want float64 }{{1.0001, 1}, {-1.0001, -1}, {0.3, 0.3}} {
		if got := clampUnitF(tc.in); got != tc.want {
			t.Errorf("clampUnitF(%g) = %g, want %g", tc.in, got, tc.want)
		}
	}
}
