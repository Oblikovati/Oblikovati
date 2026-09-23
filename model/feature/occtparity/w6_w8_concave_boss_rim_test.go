// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/math"
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
	t.Parallel()
	body := caseResultBody(t, "W6")
	assertWatertight(t, "W6", body, 9)
	assertRimTorusMajor(t, "W6", body, 25)
}

// TestW8ConcaveBossBaseRim asserts W8 (cylinder radius 30, r=5) builds a watertight 10-face solid whose
// torus band major radius is R+r=35.
func TestW8ConcaveBossBaseRim(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "W8")
	assertWatertight(t, "W8", body, 10)
	assertRimTorusMajor(t, "W8", body, 35)
}

// TestA1ConcaveBossBaseRim asserts the bfuseblend-grid sibling A1 (cylinder radius 100, r=10) builds a
// watertight 9-face solid whose torus band major radius is R+r=110 — the same fix across grids.
func TestA1ConcaveBossBaseRim(t *testing.T) {
	t.Parallel()
	body := bfuseblendResultBody(t, "A1")
	assertWatertight(t, "A1", body, 9)
	assertRimTorusMajor(t, "A1", body, 110)
}

// TestConcaveBossBaseRimArea meshes every face of W6/W8, asserts each is fold-free with a finite positive
// area, and that the summed mesh area equals OCCT's whole-result area within the corpus deps — the
// footprint now MATCHES OCCT (the whole point of the fix), mirroring TestK1Z1TessellationFoldGate.
func TestConcaveBossBaseRimArea(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		want float64
	}{{"W6", 60998.1}, {"W8", 64789.3}} {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			meshTotal := 0.0
			for _, f := range body.Faces() {
				m := tessellate.TessellateFace(f, ops.PropertyQuality())
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

// TestSpillingConcaveRimsStayDeep pins that R8 and W9 — the concave boss-base rims whose R+r cove
// spills past the cap onto the side walls (the deep #2012 boss-root weld) — are NOT claimed by
// solveConcaveBossRim's cap-fit gate and stay FAIL on the unchanged solveRim ladder, never a false
// green. They score FAIL(faulty), not FAIL(area): #2079 found the spilling cove drives the plate face
// and the side wall straight THROUGH each other (interpenetration 3-15 model units), so the body was
// never the clean "valid solid, wrong footprint" the old topology-only gate reported — the hardened
// isWatertightSolid now names it faulty. A future fix frees them by DELETING this pin, not by widening.
func TestSpillingConcaveRimsStayDeep(t *testing.T) {
	t.Parallel()
	byCase := map[string]Record{}
	for _, r := range Corpus() {
		if r.Grid == "simple" {
			byCase[r.Case] = r
		}
	}
	dir := CorpusFixtureDir()
	for _, id := range []string{"R8", "W9"} {
		if got := ScoreCase(byCase[id], dir); got != FailFaulty {
			t.Errorf("simple/%s scored %v, want FAIL(faulty) (deep #2012 boss-root; its spilling cove self-intersects, #2079)", id, got)
		}
	}
}

// TestW8CylinderWallHoldsItsChord is the per-face row the whole-body area sum cannot give, and it
// exists because a regression hid behind exactly that sum (#3517 review 3 C1).
//
// W8's face 2 is a quarter cylinder of radius exactly 10 and height 100, so its area is exactly 500π
// and its chord error is exactly readable: geom.Cylinder.ParamAt is the metric nearest point, so the
// distance from a mesh edge's midpoint to the surface is the sagitta that edge carries. When the
// classification handed this face to the covering — which its trim does not need, because it develops
// into one (u,v) branch — the mesh read 36.87× PropertyQuality's 1e-3 mm tolerance with 6554
// triangles, 49× the error for 51× the triangles, and the body's summed area still passed its 1 %
// deps. Nothing in the repo could see it.
//
// So this row asks the two questions the sum cancels: does the face meet the chord it was ASKED for,
// and is its area the analytic one less a chord deficit.
func TestW8CylinderWallHoldsItsChord(t *testing.T) {
	t.Parallel()
	q := ops.PropertyQuality()
	f := caseResultBody(t, "W8").Faces()[2]
	cyl, ok := f.Geometry().(geom.Cylinder)
	if !ok || cyl.Radius != 10 {
		t.Fatalf("W8 face 2 is a %T of radius %v, want the radius-10 cylinder this row reads", f.Geometry(), cyl.Radius)
	}
	m := tessellate.TessellateFace(f, q)
	if ratio := worstChordRatio(m, cyl, q); ratio > 1 {
		t.Errorf("W8 face 2 meshes to %.3f× its own chord tolerance with %d triangles, want within it "+
			"— a face whose trim develops onto the structured grid was meshed from the covering",
			ratio, m.TriangleCount())
	}
	want := 500 * stdmath.Pi
	if rel := (ops.MeshArea(m) - want) / want; rel > 0 || rel < -1e-4 {
		t.Errorf("W8 face 2 meshes %.6f mm², want the analytic %.6f less a chord deficit (rel %.4g)",
			ops.MeshArea(m), want, rel)
	}
}

// worstChordRatio is the largest distance from a mesh edge's midpoint to the surface the mesh
// approximates, as a multiple of the quality's own chord tolerance. 1 or less is a mesh that delivered
// the chord it promised.
//
// It measures through geom.ClosestPointOnSurface, which is the METRIC nearest point on every surface
// kind — not through ParamAt, whose own contract says it is the metric nearest point "for the plane,
// cylinder and sphere but not exactly for the cone or torus". This helper read ParamAt first and
// over-stated four cone/elliptical faces into firing while they were inside tolerance (#3517 review 5
// C1). It must stay the same oracle the production report uses (worstEdgeChord,
// kernel/ops/tessellate/face_chord_achieved.go), or a pin here asserts a different question from the
// diagnostic beside it.
func worstChordRatio(m *tessellate.Mesh, s geom.Surface, q tessellate.Quality) float64 {
	worst := 0.0
	for i := 0; i+2 < len(m.Indices); i += 3 {
		for k := range 3 {
			a, b := m.Positions[m.Indices[i+k]], m.Positions[m.Indices[i+(k+1)%3]]
			mid := math.P3((a.X+b.X)/2, (a.Y+b.Y)/2, (a.Z+b.Z)/2)
			_, _, foot := geom.ClosestPointOnSurface(s, mid)
			worst = stdmath.Max(worst, float64(mid.DistanceTo(foot)))
		}
	}
	return worst / q.Tol()
}
