// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"
)

// TestQ5ShippedFacesAgainstDrawexe rank-pairs simple/Q5's SHIPPED per-face mesh areas against DRAWEXE
// 8.0.0's own `explode result F` + `sprops result_i 1e-6`.
//
// WHY THIS CASE, AND WHY THE SHIPPED MESH. simple/Q5 used to be GREEN on whole-body area (−0.0918% of
// DRAWEXE's 3.46388e8, well inside its 1% deps) while the whole-body gate was structurally blind to what
// was happening inside it: ONE face was 5.85% short and two others over-read by +0.147% / +0.181% on a
// much larger base, so the errors partly cancelled. That is the compensating-pair class this file exists
// for, and this is the case that showed it can hide a 937-free-edge mesh leak as well as an area error.
//
// ★ TWO INHERITED ATTRIBUTIONS IN THIS FILE'S OWN COMMENT WERE FALSIFIED BY MEASUREMENT, and both are
// corrected here rather than deleted, because the correction is the finding:
//
//  1. Rank 10 is NOT "the r=2500 fillet band". It is a piece of the radius-3000 host WALL — the face the
//     band's terminal section stops on — lineage import:step#16:face#5. The band itself is rank 8
//     (import:step#16:edge#15/fillet:cyl#0, 1.28376e7) and measured −0.00082% of its own closed form
//     throughout, before and after. Nothing was ever wrong with the band.
//  2. The shortfall's root was the FAR-END TRIM, not the tessellator's choice among candidates. The
//     multi-face split (kernel/ops/fillet_farend_chain.go) used to require BOTH of a fillet's terminal
//     sections to split; Q5's splits at one end only, so the whole split was declined, nine of the
//     section's 33 stations were slid past the small wall face's own ruling, and that face's developed
//     boundary self-crossed — which is why no correct triangulation of it existed. With the split
//     admitted, the boundary is a simple polygon again and rank 10 goes 7645850.16 → 8121160.60 against
//     a closed form of 8121170.18 (−5.85% → −0.00012%). The whole body goes to −0.000194%.
//
// TestQ5FarEndSplitIsAtomicAndHitsItsClosedForms carries the closed forms and the atomicity statement;
// this test is the independent DRAWEXE-side check on the same four faces.
//
// RANK PAIRING IS SOUND HERE: DRAWEXE's ten face areas are separated by ≥4.3% (the tightest adjacent
// pair is 4.93687e7 / 4.7039e7), which is far wider than every per-face error, all of which are now
// inside 0.0014%.
func TestQ5ShippedFacesAgainstDrawexe(t *testing.T) {
	tc := drawexeFaceCase{
		name: "simple/Q5",
		// DRAWEXE 8.0.0, descending: `restore 4_G1(test22).draw s; tscale s 0 0 0 1000; explode s E;
		// blend result s 2500 s_17; explode result F; sprops result_i 1e-6`.
		drawexe: []float64{
			7.2e7, 5.78628e7, 4.93687e7, 4.7039e7, 3.6e7,
			3.46587e7, 1.8e7, 1.28376e7, 1.05e7, 8.12117e6,
		},
		totalArea: 3.46388e8,
		// Mesh quantization only. ALL TEN faces now sit inside 0.0014% (the worst is rank 4's −0.00135%
		// on the big wall piece), so this is a ~20x margin over the measurement rather than the 0.25% it
		// needed while one face was 5.85% short. TIGHTENED from 0.0025 — never widened.
		perFaceTol: 0.0003,
	}
	// ★ THE PER-RANK DEBT IS RETIRED, not re-ceilinged. Rank 10 carried {9: 0.065} for a measured 5.85%
	// shortfall; it now measures −0.00012% and is gated by perFaceTol like every other face. The total
	// budget is tightened with it, 0.002 → 0.00005 against a measured −0.000194%.
	body, ok := shippedCaseBody(q5Record(t), CorpusFixtureDir())
	if !ok {
		t.Fatal("simple/Q5 produced no shipped body")
	}
	assertShippedPerFaceAgainstDrawexe(t, tc, body, nil, 0.00005)
}

// q5Record is the corpus record for simple/Q5.
func q5Record(t *testing.T) Record {
	t.Helper()
	for _, r := range Corpus() {
		if r.Grid == "simple" && r.Case == "Q5" {
			return r
		}
	}
	t.Fatalf("simple/Q5 not in the corpus (fixtures at %s)", filepath.Clean(CorpusFixtureDir()))
	return Record{}
}
