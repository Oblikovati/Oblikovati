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

// TestHalfSpaceCutTorusObliqueOffMatrix is acceptance criterion 2 of #1406: an OFF-MATRIX oblique cut — a
// differently-proportioned upright torus (R=6, r=1.5) bitten by a 45°-tilted plane, a configuration in none
// of the analytic-builder test cases — is trimmed EXACTLY by the unified (u,v)-arrangement trimmer with NO
// bespoke builder: one analytic torus face + one planar oval lid, watertight, no faceted CSG fallback.
func TestHalfSpaceCutTorusObliqueOffMatrix(t *testing.T) {
	tor, _ := SolidTorus(math.P3(0, 0, 0), math.V3(0, 0, 1), 6, 1.5, "torus")
	plane, _ := geom.NewPlane(math.P3(0, 6, 1.5), math.V3(0, 0.7, 0.7)) // tilted ~45°, bites one oval
	res, err := HalfSpaceCut(tor, plane)
	if err != nil {
		t.Fatalf("HalfSpaceCut: %v", err)
	}
	assertWatertight(t, res)
	tori, planes, faces := 0, 0, len(res.Faces())
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Torus:
			tori++
		case geom.Plane:
			planes++
		}
	}
	if tori != 1 || planes != 1 {
		t.Errorf("off-matrix oblique oval has %d torus + %d plane faces, want 1 + 1 (exact cap + lid)", tori, planes)
	}
	if faces > 4 { // a CSG fallback would be hundreds of faceted triangles, with no analytic torus face
		t.Errorf("result has %d faces — that is faceted CSG, not the exact analytic path", faces)
	}
	if e := len(res.Edges()); e != 2 {
		t.Errorf("off-matrix oblique oval has %d edges, want 2 (the two spiric branches)", e)
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
	_ = torS
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

// The oblique oval-classification predicates (torusObliqueOval, torusTwoObliqueOval) were removed when the
// tilted single oval and figure-eight migrated to the unified (u,v)-arrangement trimmer (#1406); the
// integration tests above (TestHalfSpaceCutTorusObliqueOval/TwoObliqueOval) now exercise that path, and
// torusSpiricSection classifies the topology from the section instead of a predicate ladder.
