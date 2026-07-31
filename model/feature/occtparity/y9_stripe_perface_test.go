// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestY9PerFaceRankPairsAgainstDrawexe is simple/Y9's per-face DRAWEXE reconciliation — the first
// green of the STRIPE-JUNCTION-CROSSING capability (kernel/ops/fillet_stripe_junction.go): a full
// circular rim split into 3 tangent arcs by a wall seam and two top-face chords, blended r=4 as one
// closed stripe whose blend faces are bounded at the SUPPORT-TRANSITION crossings, not the chain
// vertices.
//
// Evidence (live DRAWEXE 8.0.0, fixture's own `nexplode`, per-face `sprops result_i 1e-12`,
// identified by `mksurface; dump`, descending): the full r=50 bottom disk (7853.98), the receded
// big top plane (6076.77) and chord sliver (570.841 — the two top planes sum to π·46² exactly,
// 6647.61), the 2π·50·16 drum wall (5026.55), and THREE torus band faces 852.995 / 704.962 /
// 358.582 whose meridian spans decode as the CROSSING meridians (160.22° / 132.42° / 67.36° —
// chordA crossing 199.78°, chordB crossing 132.42°, wall seam 0°), NOT the chain-vertex meridians
// (198.13° / 128.35° / 0°). Ours rank-pairs 7-to-7 with every residual ≤ 4.1e-5 (largest: the
// 852.995 band, −0.0041%) — pure mesh-inscription deficit. Rank pairing is sound: the closest
// distinct pair (852.995 vs 704.962) is 21% apart, three decades above the residuals.
func TestY9PerFaceRankPairsAgainstDrawexe(t *testing.T) {
	body := pinnedBody(t, "simple", "Y9")
	assertShippedPerFaceAgainstDrawexe(t, y9StripeFaceCase(), body, nil, 3e-4)
}

// y9StripeFaceCase is DRAWEXE 8.0.0's per-face record for simple/Y9 (`restore CTO904_cts21349.rle a;
// encoderegularity a 1; nexplode a e; blend result a 4 a_5 4 a_1 4 a_6`), Σ faces = 21444.68,
// matching the fixture's checkprops 21444.7.
func y9StripeFaceCase() drawexeFaceCase {
	return drawexeFaceCase{
		name: "Y9", totalArea: 21444.68, perFaceTol: 3e-4,
		drawexe: []float64{7853.98, 6076.77, 5026.55, 852.995, 704.962, 570.841, 358.582},
	}
}

// TestY9WholeBodyWatertight asserts the stripe-assembled Y9 solid is watertight WITHOUT
// tessellating: DRAWEXE's 7 faces, every edge exactly 2-incident, and a valid closed solid. A
// junction-crossing regression (a section rebuilt at the chain vertex instead of the crossing, or a
// contact sliver routed to the wrong top face) breaks 2-incidence here loud and fast.
func TestY9WholeBodyWatertight(t *testing.T) {
	body := pinnedBody(t, "simple", "Y9")
	if got := len(body.Faces()); got != 7 {
		t.Fatalf("Y9 result has %d faces, want DRAWEXE's 7", got)
	}
	for _, e := range body.Edges() {
		if n := len(e.Uses()); n != 2 {
			t.Fatalf("Y9 result edge %d is %d-incident (%v→%v), want exactly 2 (a watertight manifold solid)",
				e.ID(), n, e.StartVertex().Point(), e.EndVertex().Point())
		}
	}
	rep := ops.Validate(body)
	if !rep.Valid || !rep.Closed || !rep.HolesContained || !body.IsSolid() {
		t.Fatalf("Y9 result not a watertight solid: valid=%v closed=%v holes=%v solid=%v issues=%v",
			rep.Valid, rep.Closed, rep.HolesContained, body.IsSolid(), rep.Issues)
	}
}

// stripeSpinePins pins the stripe-junction-crossing green (captured on this HEAD, stable across
// re-runs; same capture practice and arm64 cross-platform caveat as fingerprint_pins_test.go).
func stripeSpinePins() []fingerprintPin {
	return []fingerprintPin{
		{"Y9", 156017.105020648334, 84222, 0x47095f2bf347135b, ""},
	}
}
