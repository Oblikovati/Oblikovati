// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// W6, W8 (simple grid) and A1 (bfuseblend grid) are the re-survey's CONCAVE BOSS-BASE rim family: a
// boss/pin (a cylinder standing on a plate) filleted at its base, where the wall meets the plate in a
// REENTRANT corner. Before fillet_rim_concave.go's solveConcaveBossRim they built a watertight solid with
// the DRAWEXE-exact face count but the WRONG footprint — solveRim's convex R−r round bit the corner
// inward (torus major R−r, plate hole shrinking to R−r, wall extended by r) instead of the concave cove
// OCCT builds (torus major R+r, plate hole OPENING to R+r, wall receding by r). The convex R−r
// rolling-ball probe lands inside the plate for a boss base, so the rim never fell through to the concave
// tier; solveConcaveBossRim detects the concave dihedral and offsets the tube centre AXIALLY toward the
// boss (the direction the wall rises), the one thing the bore-lip mirror (K1) got backwards. Stable
// over-sizes before the fix: W6 +3.8%, W8 +5.5% (area, Property==fine under 20× refinement).
//
// R8 (boss R8/h15 on a ±10 box, r=3 → R+r=11) and W9 (r=15 on W8's shape) are the DEEP #2012 members:
// their R+r cove spills past the cap onto the side walls (DRAWEXE 20F/12F), so solveConcaveBossRim's
// cap-fit gate returns errConcaveCoveSpills and they stay on the UNCHANGED solveRim ladder — asserted in
// TestSpillingConcaveRimsStayDeep so a future widening of the fix that wrongly claims them fails loud.
const w6w8Deps = 0.01 // the W6/W8/A1 corpus records' deps tolerance

// TestW6ConcaveBossBaseRim asserts W6 (cylinder radius 20, r=5) builds a watertight 9-face solid whose
// torus band major radius is R+r=25 (the concave cove), NOT solveRim's convex R−r=15.
func TestW6ConcaveBossBaseRim(t *testing.T) {
	body := caseResultBody(t, "W6")
	assertWatertight(t, "W6", body, 9)
	assertRimTorusMajor(t, "W6", body, 25)
}

// TestW8ConcaveBossBaseRim asserts W8 (cylinder radius 30, r=5) builds a watertight 10-face solid whose
// torus band major radius is R+r=35.
func TestW8ConcaveBossBaseRim(t *testing.T) {
	body := caseResultBody(t, "W8")
	assertWatertight(t, "W8", body, 10)
	assertRimTorusMajor(t, "W8", body, 35)
}

// TestA1ConcaveBossBaseRim asserts the bfuseblend-grid sibling A1 (cylinder radius 100, r=10) builds a
// watertight 9-face solid whose torus band major radius is R+r=110 — the same fix across grids.
func TestA1ConcaveBossBaseRim(t *testing.T) {
	body := bfuseblendResultBody(t, "A1")
	assertWatertight(t, "A1", body, 9)
	assertRimTorusMajor(t, "A1", body, 110)
}

// TestConcaveBossBaseRimArea meshes every face of W6/W8, asserts each is fold-free with a finite positive
// area, and that the summed mesh area equals OCCT's whole-result area within the corpus deps — the
// footprint now MATCHES OCCT (the whole point of the fix), mirroring TestK1Z1TessellationFoldGate.
func TestConcaveBossBaseRimArea(t *testing.T) {
	for _, tc := range []struct {
		name string
		want float64
	}{{"W6", 60998.1}, {"W8", 64789.3}} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			meshTotal := 0.0
			for _, f := range body.Faces() {
				m := ops.TessellateFace(f, ops.PropertyQuality())
				area := ops.MeshArea(m)
				meshTotal += area
				if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
					t.Fatalf("%s %T face meshed to %.4f, want a finite positive area", tc.name, f.Geometry(), area)
				}
				assertFaceFoldFreeAtEveryQuality(t, tc.name, f, m)
			}
			if rel := stdmath.Abs(meshTotal-tc.want) / tc.want; rel > w6w8Deps {
				t.Fatalf("%s total mesh area %.4f, want OCCT %.2f within deps %.2f (rel %.6f)", tc.name, meshTotal, tc.want, w6w8Deps, rel)
			}
		})
	}
}

// TestSpillingConcaveRimsStayDeep pins that R8 and W9 — the concave boss-base rims whose R+r cove spills
// past the cap onto the side walls (the deep #2012 boss-root weld) — are NOT claimed by
// solveConcaveBossRim's cap-fit gate: they must remain FAIL(area) on the unchanged solveRim ladder, not
// silently regress to FAIL(faulty) or a false green.
func TestSpillingConcaveRimsStayDeep(t *testing.T) {
	byCase := map[string]Record{}
	for _, r := range Corpus() {
		if r.Grid == "simple" {
			byCase[r.Case] = r
		}
	}
	dir := CorpusFixtureDir()
	for _, id := range []string{"R8", "W9"} {
		if got := ScoreCase(byCase[id], dir); got != FailArea {
			t.Errorf("simple/%s scored %v, want FAIL(area) (deep #2012 boss-root; cap-fit gate must reject its spilling cove)", id, got)
		}
	}
}
