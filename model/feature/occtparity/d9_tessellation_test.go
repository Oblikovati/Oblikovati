// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/tessellate"
	"oblikovati.org/kernel/topo"
)

// D9-T2 tessellation-fold gate for OCCT blend/simple/D9 — the 270° REFLEX sphere-host trihedral corner.
// D9's reflex geometry produces NEW loop shapes the convex corpus never exercised: the ~262° MAJOR-arc top
// cap bite, the EXTENDED lon-0 capping face (spliced across its old axis-edge boundary through the reflex
// far-vertex connector), the >hemisphere host sphere region, and the major-arc reflex rim/meridian bottom
// cap. Tessellation correctness is the repo's highest priority (CLAUDE.md): a folded or wrong-region mesh
// on any of these must fail loud here. This pins EVERY face's Property-quality mesh area to OCCT's per-face
// oracle (captured live from the locally built DRAWEXE 8.0.0: run the case, `explode result f`, `sprops`
// each face — the nine areas sum to the case total 274895.5) and asserts NO folds anywhere.

// d9OracleAreas are OCCT's per-face sprops areas for D9 (DRAWEXE 8.0.0), keyed by a human label. Every
// result face must match its nearest entry within d9AreaRelTol, and every entry must be hit exactly once.
var d9OracleAreas = map[string]float64{
	"host sphere R150":    179292,  // the >hemisphere 270° zone (winding fix: not the 103387 complement)
	"bottom cap z-":       13203,   // reflex 270° sector, meridian-arm far bite (subArcMajor major rim)
	"meridian torus arm":  5110.61, // Rmajor 139.642 (convex — byte-identical contact arc)
	"lon-0 capping (y=0)": 33325.5, // EXTENDED across its axis edge through the reflex connector (Splice 2)
	"lon-270 (x=0)":       29339.5, // corner-host retrim, both outer ends on-loop
	"corner blend sphere": 55.78,   // R=10 spherical triangle
	"cylinder arm":        1124.31, // interior far-vertex station outer end (D9-T1)
	"rim torus arm":       1858.58, // Rmajor 72.27 REFLEX — MAJOR (262°) contact arc (armContactSweep)
	"top cap z+":          11586.2, // ~262° cap bite closed by the NEW connector edge (Splice 1)
}

const d9AreaRelTol = 0.005 // per-face property-mesh vs OCCT oracle: every D9 face lands ≤0.14% off

// TestD9TessellationFoldGate meshes every D9 result face, asserts it is fold-free with a finite positive
// area matching its OCCT per-face oracle, that every oracle face is realized exactly once, and that the
// summed mesh area equals OCCT's 274895.5 within the corpus deps (0.01).
func TestD9TessellationFoldGate(t *testing.T) {
	t.Parallel()
	body := caseResultBody(t, "D9")
	if got := len(body.Faces()); got != 9 {
		t.Fatalf("D9 result has %d faces, want the oracle's 9", got)
	}
	hits := map[string]int{}
	meshTotal := 0.0
	for _, f := range body.Faces() {
		m := tessellate.TessellateFace(f, ops.PropertyQuality())
		area := ops.MeshArea(m)
		meshTotal += area
		assertD9FaceSane(t, f, m, area)
		hits[nearestD9Oracle(t, f, area)]++
	}
	assertEachOracleHitOnce(t, hits)
	assertD9MeshTotal(t, meshTotal)
}

// assertD9FaceSane fails if a face meshes to a non-finite / non-positive area or carries ANY fold edge —
// D9 has no known residual (unlike E4's one documented torus-arm fold), so the budget is zero.
func assertD9FaceSane(t *testing.T, f *topo.Face, m *ops.Mesh, area float64) {
	t.Helper()
	if area <= 0 || stdmath.IsInf(area, 0) || stdmath.IsNaN(area) {
		t.Fatalf("D9 %T face meshed to %.4f, want a finite positive area", f.Geometry(), area)
	}
	assertFaceFoldFreeAtEveryQuality(t, "D9", f, m)
}

// nearestD9Oracle returns the oracle label whose area is closest to the face's mesh area, failing when the
// nearest is farther than d9AreaRelTol (a face meshing the wrong region — e.g. the host-sphere complement,
// 103387 vs 179292 — lands far from every oracle and fails here).
func nearestD9Oracle(t *testing.T, f *topo.Face, area float64) string {
	t.Helper()
	best, bestErr := "", stdmath.Inf(1)
	for label, o := range d9OracleAreas {
		if e := stdmath.Abs(area - o); e < bestErr {
			best, bestErr = label, e
		}
	}
	if bestErr > d9AreaRelTol*d9OracleAreas[best] {
		t.Fatalf("D9 %T face mesh area %.2f matches no OCCT oracle within %.1f%% (nearest %q %.2f, err %.3f%%)",
			f.Geometry(), area, d9AreaRelTol*100, best, d9OracleAreas[best], 100*bestErr/d9OracleAreas[best])
	}
	return best
}

// assertEachOracleHitOnce fails unless every oracle face is realized exactly once (guards against two
// faces collapsing onto one oracle area while a third is missing).
func assertEachOracleHitOnce(t *testing.T, hits map[string]int) {
	t.Helper()
	for label := range d9OracleAreas {
		if hits[label] != 1 {
			t.Fatalf("D9 oracle face %q was realized %d times, want exactly 1", label, hits[label])
		}
	}
}

// assertD9MeshTotal asserts the summed per-face mesh area equals OCCT's whole-result area within the corpus
// deps (0.01 relative) — the fold-free / correct-region total (a folded or complement-filled face inflates it).
func assertD9MeshTotal(t *testing.T, meshTotal float64) {
	t.Helper()
	const want = 274895.5
	if rel := stdmath.Abs(meshTotal-want) / want; rel > 0.01 {
		t.Fatalf("D9 total mesh area %.2f, want OCCT %.1f within deps 0.01 (rel %.4f)", meshTotal, want, rel)
	}
}

// TestD9WholeBodyWatertight asserts D9 welds a watertight manifold solid — every edge 2-incident, valid +
// closed + holes-contained + IsSolid, with the oracle's 9 faces — WITHOUT tessellating. The reflex splices
// (the new connector edge shared by the top cap and the extended lon-0 face, the far trim shared by the
// cyl arm and lon-0) must each weld 2-incident; a cracked splice fails here loud (reuses assertWatertight).
func TestD9WholeBodyWatertight(t *testing.T) {
	t.Parallel()
	assertWatertight(t, "D9", caseResultBody(t, "D9"), 9)
}
