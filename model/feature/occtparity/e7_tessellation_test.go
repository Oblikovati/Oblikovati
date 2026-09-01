// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// E7 whole-body gate on the real OCCT blend/simple/E7 STEP body — the convex LATITUDE-cut Torus∧Plane rim
// whose exact torus arm (torusArmEdge / fillet_torusarm.go) welds through the single-arm runout. It asserts
// the weld is a watertight fold-free solid (every edge 2-incident, valid+closed+holes-contained+solid, the
// oracle 6-face count), that the exact ARM torus carries the derived major M_a = 166.4395, and that the
// whole body meshes to OCCT's area 104612 within the corpus deps (0.01). Tessellation correctness is the
// repo's highest priority (CLAUDE.md), so a folded or mis-wound arm/host — which mis-counts area — is caught
// here, not shipped. A regression that perturbs the byte-exact mesh is also caught by the E7 fingerprint pin.
const (
	e7OracleArea = 104612.0 // OCCT checkprops -s whole-result surface area (corpus expectedArea)
	e7ArmMajor   = 166.4395 // the derived arm-torus major R + √(b²−η²) (b=90, η=60.7107)
	e7Faces      = 6        // host torus + arm torus + 4 planes (2 radial cuts, bottom cap, top-tube cap)
)

func TestE7WholeBodyWatertightFoldFree(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "E7")
	assertWatertight(t, "E7", body, e7Faces)
	assertE7VolumePositive(t, body)
	meshTotal := assertE7FacesFoldFree(t, body)
	assertE7ArmTorus(t, body)
	assertE7TotalArea(t, meshTotal)
}

// assertE7VolumePositive asserts the welded solid encloses a positive volume (an inside-out or collapsed
// weld reads ≤ 0).
func assertE7VolumePositive(t *testing.T, body *topo.Body) {
	t.Helper()
	if v := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; v <= 0 {
		t.Fatalf("E7 body volume %.4f, want positive (a valid enclosed solid)", v)
	}
}

// assertE7FacesFoldFree asserts every face meshes to a finite positive area with ZERO fold edges (E7 is a
// clean latitude arm — unlike the oblique E4, it carries no residual fold), returning the summed mesh area.
func assertE7FacesFoldFree(t *testing.T, body *topo.Body) float64 {
	t.Helper()
	total := 0.0
	for _, f := range body.Faces() {
		m := tessellate.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
			t.Fatalf("E7 %T face meshed to %.4f, want a finite positive area", f.Geometry(), area)
		}
		assertFaceFoldFreeAtEveryQuality(t, "E7", f, m)
		total += area
	}
	return total
}

// assertE7ArmTorus asserts the body carries EXACTLY one exact arm torus (the derived major 166.4395, minor
// r=10), distinct from the retrimmed host torus (major 100), each meshing to a finite positive area.
func assertE7ArmTorus(t *testing.T, body *topo.Body) {
	t.Helper()
	arms := 0
	for _, f := range body.Faces() {
		tor, ok := f.Geometry().(geom.Torus)
		if !ok || stdmath.Abs(tor.MajorRadius-e7ArmMajor) > 0.01*e7ArmMajor {
			continue
		}
		arms++
		if a := faceMeshArea2(f); a <= 0 || stdmath.IsInf(a, 0) || stdmath.IsNaN(a) {
			t.Fatalf("E7 arm torus meshed to %.4f, want a finite positive area", a)
		}
	}
	if arms != 1 {
		t.Fatalf("E7 has %d arm-torus faces (major≈%.4f), want exactly 1", arms, e7ArmMajor)
	}
}

// assertE7TotalArea asserts the summed per-face mesh area equals OCCT's whole-result area within deps 0.01.
func assertE7TotalArea(t *testing.T, meshTotal float64) {
	t.Helper()
	if rel := stdmath.Abs(meshTotal-e7OracleArea) / e7OracleArea; rel > 0.01 {
		t.Fatalf("E7 total mesh area %.2f, want OCCT %.0f within deps 0.01 (rel %.5f)", meshTotal, e7OracleArea, rel)
	}
}
