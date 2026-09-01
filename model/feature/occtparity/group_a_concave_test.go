// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// Group-A gate for the CONCAVE Cylinder∧Plane single-arm runout — OCCT blend/simple N3, M4, N9. Each is a
// reentrant axis-parallel LINE edge whose rolling ball sits in the void wedge, so the fillet ADDS the fill
// wedge (fillet_arm_concave.go: the void-side exact cylinder arm + the concave GROW retrim of both hosts
// and both perpendicular caps). Tessellation correctness is the repo's highest priority (CLAUDE.md): this
// pins each result to a watertight, fold-free, volume-POSITIVE solid meshing to OCCT's whole-body area
// within the corpus deps (0.01). A mis-wound arm band (negative volume) or a cracked concave-grow splice
// fails here loud, complementing the whole-grid TestOCCTBlendSimple/{N3,M4,N9} area gate.

// groupAConcaveCases are the three concave cylinder-arm runouts with their OCCT whole-body areas (corpus.json).
var groupAConcaveCases = []struct {
	name string
	area float64
}{
	{"N3", 64521.3},
	{"M4", 61060.1},
	{"N9", 65250.0},
}

// TestGroupAConcaveWatertight asserts each concave result is a closed manifold solid (every edge exactly
// 2-incident, valid + closed + holes-contained + IsSolid) with a POSITIVE signed volume — the winding gate
// that would catch an un-flipped (inside-out) concave arm band laundered into a self-intersecting shell.
func TestGroupAConcaveWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range groupAConcaveCases {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			assertConcaveSolid(t, tc.name, body)
			assertConcaveVolumePositive(t, tc.name, body)
		})
	}
}

// TestGroupAConcaveFoldFreeArea meshes every result face, asserts it is fold-free with a finite positive
// area, and that the summed area equals OCCT's whole-body oracle within the corpus deps (0.01).
func TestGroupAConcaveFoldFreeArea(t *testing.T) {
	t.Parallel()
	for _, tc := range groupAConcaveCases {
		t.Run(tc.name, func(t *testing.T) {
			body := caseResultBody(t, tc.name)
			total := 0.0
			for _, f := range body.Faces() {
				m := tessellate.TessellateFace(f, ops.PropertyQuality())
				total += assertConcaveFaceSane(t, tc.name, f, m)
			}
			if rel := stdmath.Abs(total-tc.area) / tc.area; rel > 0.01 {
				t.Fatalf("%s total mesh area %.2f, want OCCT %.1f within deps 0.01 (rel %.4f)", tc.name, total, tc.area, rel)
			}
		})
	}
}

// assertConcaveSolid checks the body is a closed 2-incident manifold solid (reuses the shared watertight
// gate without a face-count oracle — N3/M4/N9 result face counts are not pinned here, only manifoldness).
func assertConcaveSolid(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	for _, e := range body.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Fatalf("%s edge %d is %d-incident (%v→%v), want exactly 2 (watertight manifold)",
				name, e.ID(), n, e.StartVertex().Point(), e.EndVertex().Point())
		}
	}
	rep := ops.Validate(body)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !body.IsSolid() {
		t.Fatalf("%s not a watertight solid: valid=%v closed=%v holes=%v solid=%v issues=%v",
			name, rep.Valid, rep.Closed, rep.HolesContained, body.IsSolid(), rep.Issues)
	}
}

// assertConcaveVolumePositive fails when the concave arm band welds inside-out (negative signed volume) —
// the derivation's mandatory winding assertion for the material-outward-negated concave arm.
func assertConcaveVolumePositive(t *testing.T, name string, body *topo.Body) {
	t.Helper()
	if v := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; v <= 0 {
		t.Fatalf("%s signed volume %.4f ≤ 0 — the concave arm band is wound inside-out", name, v)
	}
}

// assertConcaveFaceSane fails when a face meshes to a non-finite / non-positive area or carries ANY fold
// edge, returning the face's mesh area for the whole-body sum.
func assertConcaveFaceSane(t *testing.T, name string, f *topo.Face, m *ops.Mesh) float64 {
	t.Helper()
	area := ops.MeshArea(m)
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("%s %T face meshed to %.4f, want a finite positive area", name, f.Geometry(), area)
	}
	assertFaceFoldFreeAtEveryQuality(t, name, f, m)
	return area
}
