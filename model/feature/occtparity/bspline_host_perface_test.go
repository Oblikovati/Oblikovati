// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"
)

// Per-face DRAWEXE reconciliation for the wave-G BSpline-host fillet engine
// (kernel/ops/fillet_bspline_host*.go — the edge-anchored section-plane Newton march, canal
// loft, and prolong-then-trim runout/rim weld). The whole-body area gate is structurally blind
// to compensating per-face errors (proven three times in this project), so every green here is
// rank-paired against OCCT 8.0.0's own `explode result F` + `sprops result_i 1.e-12` numbers,
// captured by running each case's OWN tests/blend script verbatim through DRAWEXE (SCALE1=10,
// the corpus grid's own scale — matches every case's radius: G5/I5/I7/J9 r=1*SCALE1=10, G7/G9
// r=0.5*SCALE1=5, V6 r=2*SCALE1=20, B2 r=1*SCALE1=10). All eight use `explode` (none are in the
// nexplode set). checkshape was silent (valid) on every one; scratchpad/wg/*.tcl + *.out hold
// the driver scripts and raw DRAWEXE transcripts this table was read from.
//
// Rank pairing is sound for all eight: G7 has one genuine symmetry-twin pair (3711.21/3711.21,
// the prism's front/back caps — interchangeable by construction); V6 similarly (57179.3 twice);
// I5/I7 each carry two side-wall strips within 0.03% of each other (9975.47/9972.52,
// 9975.99/9975.96) — close but NOT tied, and our own per-face mesh areas land within the SAME
// pair ordering at the SAME separation, so the rank pairing never crosses. The worst measured
// deviation across all 40 rank-paired faces is G5's smallest face at −0.070% (mesh quantization
// on the runout's double-bitten cap wedge); every other face is under 0.03%.
const bsplineHostPerFaceTol = 1e-3

// bsplineHostPerFaceCases is the pinned population, DRAWEXE 8.0.0 receipts at sprops 1.e-12
// (captured 2026-07-31, scripts scratchpad/wg/*.tcl; checkshape valid on all eight).
func bsplineHostPerFaceCases() []cylCylSeamCase {
	return []cylCylSeamCase{
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "G5", totalArea: 106867, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{36275.4, 23000, 22403.4, 19335.1, 5853.26}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "G7", totalArea: 33988.4, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{16082.6, 9569.8, 3711.21, 3711.21, 909.873}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "G9", totalArea: 33865.5, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{15777.2, 10181.3, 3720, 2839.91, 1339.86}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "I5", totalArea: 81863.7, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{18100.9, 14581.2, 13776.9, 13038, 9975.47, 9972.52, 2418.69}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "I7", totalArea: 81743.8, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{16290.8, 15307.7, 14581.2, 12787.1, 9975.99, 9975.96, 2824.98}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "J9", totalArea: 86937.1, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{69495.6, 7853.98, 5040.39, 4544.46}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "V6", totalArea: 263402, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{95197.1, 57179.3, 57179.3, 49718, 4128.2}}},
		{grid: "bfuseblend", drawexeFaceCase: drawexeFaceCase{name: "B2", totalArea: 431283, perFaceTol: bsplineHostPerFaceTol,
			drawexe: []float64{90000, 78655.6, 69394.6, 45000, 45000, 45000, 45000, 7853.98, 5365.89}}},
	}
}

// TestBsplineHostPerFaceParity is the oracle gate for the wave-G BSpline-host fillet family.
// The on-face budget is read from the SAME corpus-wide ratchet TestEveryLoopSegmentLiesOnItsFace
// gates (offSurfaceDebtIndex, loopseg_onface_test.go) rather than a second hardcoded number, so
// the two can never silently disagree: six of the eight cases hold the default 1e-6 (every rail
// is the loft's own iso-curve and every trim is the band's own marched crossing, so boundary and
// surface agree by construction); J9 and B2 carry the documented CLOSED-rim chorded-seam debt
// (knownOffSurfaceDebt's wave-G entries) — a general swept B-spline wall gives the rolling
// ball's contact point no guarantee of landing on the wall's OWN pre-existing structural seam,
// unlike the axisymmetric hosts the shared rim rebuild was proven exact on.
func TestBsplineHostPerFaceParity(t *testing.T) {
	debt := offSurfaceDebtIndex()
	for _, tc := range bsplineHostPerFaceCases() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := pinnedBody(t, tc.grid, tc.name)
			assertPerFaceAgainstDrawexe(t, tc.drawexeFaceCase, body)
			assertLoopSegmentsOnFaces(t, Record{Grid: tc.grid, Case: tc.name}, body, debt[tc.grid+"/"+tc.name])
		})
	}
}

// TestBsplineHostWatertight pins the eight results as watertight manifold solids at the
// DRAWEXE face-count join: every edge exactly 2-incident, the oracle's face count, 0 free
// edges + no retracing/self-crossing loops at BOTH gate qualities, and every face fold-free at
// both (a wrong-winding or gapped band cannot pass any of these).
func TestBsplineHostWatertight(t *testing.T) {
	for _, tc := range bsplineHostPerFaceCases() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := pinnedBody(t, tc.grid, tc.name)
			assertWatertight(t, tc.name, body, len(tc.drawexe))
			assertBodyIsWatertightAndSimple(t, body, tc.grid+"/"+tc.name)
			for _, f := range body.Faces() {
				assertFaceFoldFreeAtEveryQuality(t, tc.grid+"/"+tc.name, f, nil)
			}
		})
	}
}
