// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// J1 is the convex CLOSED cone-cap rim band (Miter-A1): a single convex closed circular edge where the
// host cone meets its perpendicular cap plane rounds into a FULL torus band — no corner, no runout. The
// demux (closedRimPick) routes it out of solveMiter, and the closed-band arm assembly
// (fillet_curved_closed_rim.go) welds the arm's exact torus bounded by its two receded-host contact circles.
// This gate pins, WITHOUT relying only on the whole-body area, that the result is a watertight fold-free
// solid whose four faces (receded cone, receded cap disk, untouched bottom cap, torus band) each mesh to a
// finite positive area, and that the summed mesh area equals OCCT's 135648 within the corpus deps (0.01).

const j1WholeArea = 135648.0 // OCCT tests/blend/simple/J1 whole-result area
const j1CorpusDeps = 0.01    // the J1 corpus record's deps tolerance

// TestJ1ClosedRimWatertight asserts J1 welds a watertight manifold solid with the oracle's 4 faces — every
// edge 2-incident, valid + closed + holes-contained + IsSolid — WITHOUT tessellating (a cracked band seam
// or a mis-receded host would fail here loud and fast). Reuses the shared assertWatertight harness.
func TestJ1ClosedRimWatertight(t *testing.T) {
	assertWatertight(t, "J1", caseResultBody(t, "J1"), 4)
}

// TestJ1ClosedRimTessellationFoldGate meshes every J1 face, asserts each is fold-free with a finite positive
// area, that the band is a geom.Torus (the arm surface, not a fallback), and that the summed mesh area equals
// OCCT's whole-result area within deps — the fold-free / correct-region total (a full-donut fallback mesh on
// the torus band, or a folded receded host, would inflate it well past the 1% bound).
func TestJ1ClosedRimTessellationFoldGate(t *testing.T) {
	body := caseResultBody(t, "J1")
	meshTotal, torusBands := 0.0, 0
	for _, f := range body.Faces() {
		m := ops.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		meshTotal += area
		assertJ1FaceSane(t, f, m, area)
		if _, ok := f.Geometry().(geom.Torus); ok {
			torusBands++
		}
	}
	if torusBands != 1 {
		t.Fatalf("J1 has %d torus-band faces, want exactly 1 (the closed arm band)", torusBands)
	}
	if rel := stdmath.Abs(meshTotal-j1WholeArea) / j1WholeArea; rel > j1CorpusDeps {
		t.Fatalf("J1 total mesh area %.2f, want OCCT %.1f within deps %.2f (rel %.5f)", meshTotal, j1WholeArea, j1CorpusDeps, rel)
	}
	if vol := ops.BodyGeometryProperties(body, ops.PropertyQuality()).Volume; vol <= 0 {
		t.Fatalf("J1 result volume %.4f, want positive (a wound-out band would read ≤0)", vol)
	}
}

// assertJ1FaceSane fails if a J1 face meshes to a non-finite / non-positive area or carries ANY fold edge —
// the closed rim has no known residual fold, so the budget is zero.
func assertJ1FaceSane(t *testing.T, f *topo.Face, m *ops.Mesh, area float64) {
	t.Helper()
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("J1 %T face meshed to %.4f, want a finite positive area", f.Geometry(), area)
	}
	assertFaceFoldFreeAtEveryQuality(t, "J1", f, m)
}
