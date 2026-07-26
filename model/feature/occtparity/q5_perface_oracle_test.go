// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"
)

// TestQ5ShippedFacesAgainstDrawexe rank-pairs simple/Q5's SHIPPED per-face mesh areas against DRAWEXE
// 8.0.0's own `explode result F` + `sprops result_i 1e-6`.
//
// WHY THIS CASE, AND WHY THE SHIPPED MESH. simple/Q5 is GREEN on whole-body area (−0.0918% of
// DRAWEXE's 3.46388e8, well inside its 1% deps), and the whole-body gate is structurally blind to what
// is actually happening inside it: ONE face — the r=2500 fillet band — is 5.85% short, and two others
// over-read by +0.15% / +0.18% on a much larger base, so the errors partly cancel. That is the
// compensating-pair class this file exists for.
//
// The shortfall's root is NOT the tessellator: the band's boundary, developed onto its own cylinder,
// SELF-INTERSECTS, so its (u,v) trim has no well-defined area and no correct triangulation exists.
// What the mesher can do is pick the better of its candidates, and it does — the cross-face conformance
// re-mesh ships 7.6459e6 where the plain per-face mesh would ship 6.5576e6 (−19.3%). That is why the
// coverage certificate in kernel/ops/cdt_coverage.go CHOOSES rather than vetoes, and this test is the
// gate on that decision: make the certificate refuse a non-covering re-mesh outright, or promote an
// uncertified one, and rank 10 falls to −19% and this fails loud.
//
// RANK PAIRING IS SOUND HERE: DRAWEXE's ten face areas are separated by ≥4.3% (the tightest adjacent
// pair is 4.93687e7 / 4.7039e7), which is wider than every per-face error except rank 10's — and rank
// 10 is the smallest face by 29%, so nothing can be mistaken for it.
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
		// Mesh quantization only: nine of the ten faces sit inside 0.18%.
		perFaceTol: 0.0025,
	}
	// Rank 10 (the r=2500 fillet band) carries a measured 5.85% shortfall whose root is its
	// self-intersecting developed boundary, not the mesher. Ratchet at 1.11x — it may shrink freely.
	debt := map[int]float64{9: 0.065}
	body, ok := shippedCaseBody(q5Record(t), CorpusFixtureDir())
	if !ok {
		t.Fatal("simple/Q5 produced no shipped body")
	}
	assertShippedPerFaceAgainstDrawexe(t, tc, body, debt, 0.002)
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
