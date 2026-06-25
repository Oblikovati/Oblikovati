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

// The same plane keeping the OTHER side leaves the genus-1 COMPLEMENT: the full torus minus the oval cap,
// kept as one torus face carrying the oval as a HOLE (no outer loop, it wraps the whole surface) plus the
// oval lid, watertight, no CSG (Oblikovati/Oblikovati#1375).
func TestHalfSpaceCutTorusAxisParallelComplement(t *testing.T) {
	tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2, "torus")
	// Keep y≤6 (the big complement): the cut normal points +y, leaving the rest of the torus on its negative side.
	plane, _ := geom.NewPlane(math.P3(0, 6, 0), math.V3(0, 1, 0))
	res, err := HalfSpaceCut(tor, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	tori, planes := 0, 0
	var torusFace = -1
	for i, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tori++
			torusFace = i
		case geom.Plane:
			planes++
		}
	}
	if tori != 1 || planes != 1 {
		t.Fatalf("complement has %d torus + %d plane faces, want exactly 1 + 1", tori, planes)
	}
	// The complement torus face carries the oval as a HOLE (an inner loop), with no outer loop.
	for _, l := range res.Faces()[torusFace].Loops() {
		if l.IsOuter() {
			t.Error("complement torus face has an outer loop; the oval must be a hole (so the face wraps the whole torus)")
		}
	}
}

// torusSingleOvalCap and ...Complement split the axis-parallel single-oval cut by which side is kept (the
// small cap, K<0, vs the genus-1 complement, K>0); every other plane (perpendicular, a clearing or two-oval
// offset) belongs to neither, so it defers to the band path or CSG.
func TestTorusSingleOvalCapGuards(t *testing.T) {
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, tc := range []struct {
		name     string
		origin   math.Point3
		normal   math.Vector3
		cap, cmp bool
	}{
		{"axis-∥ cap (kept side)", math.P3(0, 6, 0), math.V3(0, -1, 0), true, false},
		{"axis-∥ complement (kept side)", math.P3(0, 6, 0), math.V3(0, 1, 0), false, true},
		{"perpendicular to axis", math.P3(0, 0, 1), math.V3(0, 0, 1), false, false},
		{"clears the tube (offset > R+r)", math.P3(0, 8, 0), math.V3(0, -1, 0), false, false},
		{"two-oval offset (< R−r)", math.P3(0, 2, 0), math.V3(0, -1, 0), false, false},
	} {
		plane, _ := geom.NewPlane(tc.origin, tc.normal)
		if got := torusSingleOvalCap(tor, plane); got != tc.cap {
			t.Errorf("%s: torusSingleOvalCap = %v, want %v", tc.name, got, tc.cap)
		}
		if got := torusSingleOvalComplement(tor, plane); got != tc.cmp {
			t.Errorf("%s: torusSingleOvalComplement = %v, want %v", tc.name, got, tc.cmp)
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
