// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// I9 (simple grid) is a plain solid cylinder (r=10) whose top rim is rounded by a single closed-rim
// fillet — the cylinder/cap toroidal band FilletCylinderRim already builds, EXCEPT the picked rim edge
// is (like every STEP-imported full circle) a geom.Arc3d with SweepAngle≈2π, not a geom.Circle. Before
// the rim-fillet pick gate widen (fillet_rim.go's isClosedCircularEdge), loneRimPick/resolveRim's
// geom.Circle-only check rejected this edge, the pick fell through to loneArcPick, and cylSideEdgeAt
// declined the self-closed vertex — I9 (and 7 siblings sharing the same mis-dispatch: K1, R8, U6, W6,
// W8, W9, Z1) all read FAIL(faulty) "arc end is not a cylinder/side tangent vertex". I9 is the ONE of
// the 8 whose rim solver has no other gap (a plain cylinder rim, no crossing boss, no concave bore lip,
// no boss-root cap topology), so it is the one that greens; the other 7 land on their own,
// separately-tracked gaps (recon: .superpowers/sdd/cylside-arcend-recon-report.md).
const i9WholeArea = 61439.0 // OCCT tests/blend/simple/I9 whole-result area
const i9CorpusDeps = 0.01   // the I9 corpus record's deps tolerance

// TestI9Arc3dRimWatertight asserts the widened gate routes I9's Arc3d rim to FilletCylinderRim and
// builds a watertight manifold solid with the oracle's 4 faces (receded cylinder, both caps, torus
// band) — every edge 2-incident, valid + closed + holes-contained + IsSolid.
func TestI9Arc3dRimWatertight(t *testing.T) {
	t.Parallel()
	assertWatertight(t, "I9", caseResultBody(t, "I9"), 4)
}

// TestI9Arc3dRimTessellationFoldGate meshes every I9 face, asserts each is fold-free with a finite
// positive area, that exactly one face is the geom.Torus band (not a fallback mesh), and that the
// summed mesh area equals OCCT's whole-result area within deps.
func TestI9Arc3dRimTessellationFoldGate(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "I9")
	meshTotal, torusBands := 0.0, 0
	for _, f := range body.Faces() {
		m := tessellate.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		meshTotal += area
		assertI9FaceSane(t, f, m, area)
		if _, ok := f.Geometry().(geom.Torus); ok {
			torusBands++
		}
	}
	if torusBands != 1 {
		t.Fatalf("I9 has %d torus-band faces, want exactly 1 (the rim band)", torusBands)
	}
	if rel := stdmath.Abs(meshTotal-i9WholeArea) / i9WholeArea; rel > i9CorpusDeps {
		t.Fatalf("I9 total mesh area %.4f, want OCCT %.1f within deps %.2f (rel %.6f)", meshTotal, i9WholeArea, i9CorpusDeps, rel)
	}
	if vol := query.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; vol <= 0 {
		t.Fatalf("I9 result volume %.4f, want positive", vol)
	}
}

// assertI9FaceSane fails if an I9 face meshes to a non-finite / non-positive area or carries a fold.
func assertI9FaceSane(t *testing.T, f *topo.Face, m *ops.Mesh, area float64) {
	t.Helper()
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("I9 %T face meshed to %.4f, want a finite positive area", f.Geometry(), area)
	}
	assertFaceFoldFreeAtEveryQuality(t, "I9", f, m)
}
