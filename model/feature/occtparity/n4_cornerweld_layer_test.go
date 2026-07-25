// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"strings"
	"testing"
)

// The N4 gate for the general corner-weld layer (corner-weld-layer-design.md slice 1). N4 =
// tests/blend/simple/N4: a 100³ box with a full r=20 × h=50 cylinder standing on its vertical corner (270°
// of it protrudes), filleted r=5 on three edges meeting at one trihedral vertex — the concave boss-wall ∧
// box-wall edge, the CONVEX boss cap-rim arc, and the concave band where the cap plane meets the box wall.
// OCCT's oracle (DRAWEXE 8.0.0, `restore CFI_e5678fil.rle s ; tscale s 0 0 0 10 ; explode s e ;
// blend result s 5 s_4 5 s_13 5 s_2`) is a valid SOLID of 14 faces / 14 wires / 22 vertices / 34 edges,
// area 64287.2.
//
// Slice 1a routes N4 through the shared layer at its EXISTING decline point, so the corpus is unchanged;
// this test pins that decline so the routing cannot silently regress into a different (or a partial) result.

// n4WeldDecline drives the real feature path on N4 and returns the fillet health reason.
func n4WeldDecline(t *testing.T) string {
	t.Helper()
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == "N4" {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Skipf("N4 import-divergence (not a fillet defect): %v", err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Skip("N4 picks could not be located on the imported body")
	}
	_, okFillet, reason := runFillet(body, sets)
	if okFillet {
		t.Fatal("N4 now welds — update this test to the per-case watertight/area gate and bump the rollup")
	}
	return reason
}

// TestN4LayerReachesTheRimContinuationGate pins WHERE N4 stops. The layer must carry N4 all the way through
// classification, the coons4 corner solve, both terminating arms' bundles and the lateral torus arm's near
// rail — and stop at exactly one place: the torus arm's far end, which is a G1 SEAM on the boss wall (the
// rim continues past it on the same pair of host surfaces), so the far-runout engine's admission gate finds
// ZERO transverse capping faces. A decline anywhere earlier means the layer lost ground the bespoke weld had.
func TestN4LayerReachesTheRimContinuationGate(t *testing.T) {
	reason := n4WeldDecline(t)
	for _, want := range []string{"torus arm", "far runout", "0 non-host transverse faces"} {
		if !strings.Contains(reason, want) {
			t.Fatalf("N4 declined with %q, want the torus-arm rim-continuation gate (missing %q)", reason, want)
		}
	}
}

// TestN4LayerFloorsCleanly is the do-no-harm floor: a corner the layer cannot close must yield NO body at
// all (the op errors), never a partial or cracked solid.
func TestN4LayerFloorsCleanly(t *testing.T) {
	if reason := n4WeldDecline(t); reason == "" {
		t.Fatal("N4 declined with an EMPTY reason — the floor must always name the obstruction")
	}
}
