// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestMixedTorusCornerWatertight is the per-case whole-body gate for the three OCCT tests/blend/simple
// cases the P3 mixed-sense corner-setback pass touches: K9 and M2 (RED→GREEN) and L6 (the one green-by-
// tolerance sibling legitimately re-welded). Each is a box+boss/box+cyl whose single trihedral corner
// joins TWO CONCAVE + ONE CONVEX fillet at three mutually-orthogonal planar faces — a MIXED-sense
// corner, whose rolling ball pivots around the convex edge and sweeps a TORUS (major R=2r=10, minor
// r=5, patch area (25π/2)(π−1)=84.10), NOT the sphere (area 274.35) solveBlend forced. It asserts —
// WITHOUT relying on the area-only TestOCCTBlendSimple — that each result is a watertight fold-free
// solid carrying exactly one R=2r torus corner of area ≈84.10 (a regression that re-forces the sphere,
// or drops the setback, fails loud here) and OCCT's whole-body area.
func TestMixedTorusCornerWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name  string
		faces int
		area  float64 // OCCT checkprops whole-body reference area
	}{
		{"K9", 15, 66160.6},
		{"M2", 14, 64088.4},
		{"L6", 15, 65867.9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertWatertight(t, tc.name, body, tc.faces)
			assertWholeBodyFoldFree(t, tc.name, body)
			assertWholeBodyArea(t, tc.name, body, tc.area)
			assertMixedCornerTorus(t, tc.name, body)
		})
	}
}

// assertMixedCornerTorus checks the body carries EXACTLY ONE corner torus and that it is the derived
// mixed-sense R=2r patch: major radius 2r=10, minor r=5, mesh area ≈84.10 — the crux the P3 pass fixes.
// A regression that re-forces the sphere (274.35) leaves zero tori and fails on the count; a mis-sized
// torus fails on the radii or the 84.10 area.
func assertMixedCornerTorus(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	found := 0
	for _, f := range body.Faces() {
		tor, ok := f.Geometry().(geom.Torus)
		if !ok {
			continue
		}
		found++
		eps := ops.ResolutionForBody(body).Weld() // model-relative coincidence tolerance (M35), not a bare epsilon
		if stdmath.Abs(tor.MajorRadius-10) > eps || stdmath.Abs(tor.MinorRadius-5) > eps {
			t.Fatalf("%s corner torus radii major=%.6f minor=%.6f, want R=2r=10 minor=r=5 (a mis-sized/degenerate pivot?)",
				name, tor.MajorRadius, tor.MinorRadius)
		}
		if a := faceMeshArea2(f); stdmath.Abs(a-84.10) > 0.02*84.10 {
			t.Fatalf("%s corner torus area %.4f, want ≈84.10 =(25π/2)(π−1) (a sphere would read 274.35)", name, a)
		}
	}
	if found != 1 {
		t.Fatalf("%s has %d corner tori, want exactly 1 R=2r mixed-sense patch (a re-forced sphere reads 0)", name, found)
	}
}
