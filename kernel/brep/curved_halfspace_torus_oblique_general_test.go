// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A plane oblique to the torus axis (tilted torus, axis-aligned cut) bites a single asymmetric spiric oval:
// the kept solid is the small contractible CAP or its genus-1 COMPLEMENT, watertight, no CSG (Oblikovati#1375).
func TestHalfSpaceCutTorusObliqueOval(t *testing.T) {
	for _, tc := range []struct {
		name   string
		normal math.Vector3
	}{
		{"keep cap (small)", math.V3(0, 0, -1)}, // z≥3.6 keeps the small cap
		{"keep complement", math.V3(0, 0, 1)},   // z≤3.6 keeps the genus-1 complement
	} {
		t.Run(tc.name, func(t *testing.T) {
			tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 5, 2, "torus")
			plane, _ := geom.NewPlane(math.P3(0, 0, 3.6), tc.normal)
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
				t.Errorf("oblique oval cut has %d torus + %d plane faces, want 1 + 1", tori, planes)
			}
		})
	}
}

// torusObliqueOvalRange finds the oval's tube-angle interval and pinch sign from the closed-form w=±1
// crossings, exactly where the sampled section starts/ends.
func TestTorusObliqueOvalRange(t *testing.T) {
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 5, 2)
	plane, _ := geom.NewPlane(math.P3(0, 0, 3.6), math.V3(0, 0, 1))
	_, m, k, c := geom.TorusSectionCoeffs(tor, plane)
	v0, v1, pinch, ok := torusObliqueOvalRange(tor, m, k, c)
	if !ok {
		t.Fatal("expected a single oval")
	}
	// |w|=1 at both ends, |w|<1 strictly inside.
	if w := stdmath.Abs(torusW(tor, m, k, c, v0)); stdmath.Abs(w-1) > 1e-9 {
		t.Errorf("|w(v0)|=%g, want 1", w)
	}
	if w := stdmath.Abs(torusW(tor, m, k, c, (v0+v1)/2)); w > 1 {
		t.Errorf("|w(mid)|=%g, want <1 (inside the oval)", w)
	}
	if pinch != 1 && pinch != -1 {
		t.Errorf("pinch=%g, want ±1", pinch)
	}
}

// A tilted plane through the central hole cuts TWO oblique ovals (the section is valid at every tube angle):
// the kept solid is a v-wrapping band + two oval lids, watertight, no CSG (Oblikovati#1375).
func TestHalfSpaceCutTorusTwoObliqueOval(t *testing.T) {
	tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 5, 2, "torus")
	torS, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 5, 2)
	plane, _ := geom.NewPlane(math.P3(0, 0, 0.5), math.V3(0, 0, 1)) // tilted, through the hole
	if !torusTwoObliqueOval(torS, plane) {
		t.Fatal("expected a tilted two-oval section")
	}
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
		t.Errorf("tilted two-oval cut has %d torus + %d plane faces, want 1 + 2 (band + two oval lids)", tori, planes)
	}
}

// torusTwoObliqueOval admits only a tilted cut through the hole (two ovals); a tilted single-oval bite (with
// w=±1 crossings) or an upright cut is not it.
func TestTorusTwoObliqueOvalGuards(t *testing.T) {
	torS, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0.6, 0.8), 5, 2)
	for _, tc := range []struct {
		name   string
		origin math.Point3
		want   bool
	}{
		{"tilted through hole (two ovals)", math.P3(0, 0, 0.5), true},
		{"tilted figure-eight (z=1.0, near-full-wrap)", math.P3(0, 0, 1.0), true},
		{"tilted single-oval bite (z=3.6)", math.P3(0, 0, 3.6), false},
	} {
		plane, _ := geom.NewPlane(tc.origin, math.V3(0, 0, 1))
		if got := torusTwoObliqueOval(torS, plane); got != tc.want {
			t.Errorf("%s: torusTwoObliqueOval = %v, want %v", tc.name, got, tc.want)
		}
		// The figure-eight and two-oval cases must NOT also be claimed by the single-oval (cap/complement) path.
		if got := torusObliqueOval(torS, plane); got == tc.want && tc.want {
			t.Errorf("%s: torusObliqueOval also claimed it (overlap with the band path)", tc.name)
		}
	}
	// An upright (axis-aligned) torus cut is never the oblique two-oval case.
	upright, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	plane, _ := geom.NewPlane(math.P3(0, 2, 0), math.V3(0, 1, 0))
	if torusTwoObliqueOval(upright, plane) {
		t.Error("axis-parallel two-oval wrongly classified as oblique")
	}
}

// A perpendicular or axis-parallel plane is not the oblique case; a non-tilted cut must defer.
func TestTorusObliqueOvalGuards(t *testing.T) {
	tor, _ := geom.NewTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 5, 2)
	for _, tc := range []struct {
		name   string
		normal math.Vector3
	}{
		{"perpendicular", math.V3(0, 0, 1)},
		{"axis-parallel", math.V3(0, 1, 0)},
	} {
		plane, _ := geom.NewPlane(math.P3(0, 0, 1), tc.normal)
		if torusObliqueOval(tor, plane) {
			t.Errorf("%s plane wrongly classified as an oblique oval", tc.name)
		}
	}
}
