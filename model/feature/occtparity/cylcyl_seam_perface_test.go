// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/topo"
)

// Per-face DRAWEXE reconciliation for the Cylinder∧Cylinder SSI-seam canal engine
// (kernel/ops/fillet_cylcyl_seam*.go) — the CLOSED-loop family: convex bore-exit rims (K2/K3 the
// orthogonal bore's two exits, K4 the oblique over-radius cutter) and concave fused valleys
// (B5 the oblique fuse, B4 the orthogonal fuse's entry+exit loops in ONE op). The whole-body area
// gate is structurally blind to compensating per-face errors (proven three times in this project),
// so every green here is rank-paired against OCCT 8.0.0's own `explode result F` +
// `sprops result_i 1.e-12` numbers, captured with each fixture's OWN explode variant (K2/K3 use
// `nexplode` — running `explode` silently renumbers the picks).
//
// Rank pairing is sound for all five: within each body the only equal-area pairs are exact
// symmetry twins (the two Ø100 caps of K2/K3, B4's mirror-symmetric band/stub/cap pairs), which
// pair among themselves; the closest non-twin pair differs by >39%.

// cylCylSeamPerFaceTol is the per-face RELATIVE budget. Every station column of the band is exact
// on the true envelope and both host retrims ride the loft's own iso-curve rails, so the residual
// against DRAWEXE is mesh quantization along the curved rail strip (the band and its two receded
// hosts trade a fraction of a unit² across the shared rail). Measured at 1e-9 (RED, receipts in
// wave-report-B.md): worst K4 receded-bore face +0.1297% / band −0.1150%, K2/K3 band −0.0923%,
// every analytic face ≤0.0025%. 2e-3 leaves ~1.5× headroom over the worst while staying two
// decades under the +50.9%/−73.4% compensating-error class this gate exists to catch.
const cylCylSeamPerFaceTol = 2e-3

// TestCylCylSeamPerFaceParity is the oracle gate for the closed cyl∧cyl seam canal family.
func TestCylCylSeamPerFaceParity(t *testing.T) {
	t.Parallel()
	for _, tc := range cylCylSeamPerFaceCases() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := cylCylSeamResultBody(t, tc.grid, tc.name)
			assertPerFaceAgainstDrawexe(t, tc.drawexeFaceCase, body)
			assertLoopSegmentsOnFaces(t, Record{Grid: tc.grid, Case: tc.name}, body, cylCylSeamOnFaceTol)
		})
	}
}

// cylCylSeamOnFaceTol is the "every boundary edge lies on the face it bounds" budget relative to
// the bounding diagonal — the corpus-wide 1e-6 ratchet bar, held with NO debt entry (the rails are
// the loft's own iso-curves, so band boundary and band surface agree by construction).
const cylCylSeamOnFaceTol = 1e-6

// cylCylSeamCase names one closed-seam case and its DRAWEXE receipts.
type cylCylSeamCase struct {
	grid string
	drawexeFaceCase
}

// cylCylSeamPerFaceCases is the pinned population, DRAWEXE 8.0.0 receipts at sprops 1.e-12
// (captured 2026-07-31, scripts scratchpad/wb/*.tcl; checkshape valid on all five).
func cylCylSeamPerFaceCases() []cylCylSeamCase {
	return []cylCylSeamCase{
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "K2", totalArea: 72801.7, perFaceTol: cylCylSeamPerFaceTol,
			drawexe: []float64{39433.3, 15502.8, 7853.98, 7853.98, 2157.55}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "K3", totalArea: 72801.7, perFaceTol: cylCylSeamPerFaceTol,
			drawexe: []float64{39433.3, 15502.8, 7853.98, 7853.98, 2157.55}}},
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "K4", totalArea: 19943.5, perFaceTol: cylCylSeamPerFaceTol,
			drawexe: []float64{13605, 2827.43, 1847.67, 1663.41}}},
		{grid: "bfuseblend", drawexeFaceCase: drawexeFaceCase{name: "B5", totalArea: 64858.3, perFaceTol: cylCylSeamPerFaceTol,
			drawexe: []float64{36385.3, 14407.8, 5026.55, 5026.55, 2827.43, 1184.74}}},
		{grid: "bfuseblend", drawexeFaceCase: drawexeFaceCase{name: "B4", totalArea: 82814.4, perFaceTol: cylCylSeamPerFaceTol,
			drawexe: []float64{39703, 9628.81, 9628.81, 7853.98, 7853.98, 2827.43, 2827.43, 1245.5, 1245.5}}},
		// The equal-parallel valley LINE seam (exact cylinder arm, cylCylParallelValleyArmEdge +
		// the existing single-arm runout weld). P1's 14234.3 pair are symmetry twins. complex/E8
		// (the unequal-radius parallel valley) is deliberately ABSENT: its R10-wall foot ruling
		// lands 0.03125 rad past the host's split ruling, the runout weld clamps at the face
		// boundary, and the shipped solid keeps strips totalling +19.98 area vs OCCT's per-face
		// receipts (a would-be false green at +0.074% whole-body) — so the arm DECLINES it
		// (cylCylValleyFeetOnHosts) until a neighbour-recede/sew capability exists.
		{grid: "simple", drawexeFaceCase: drawexeFaceCase{name: "P1", totalArea: 98604.8, perFaceTol: cylCylSeamPerFaceTol,
			drawexe: []float64{34550.7, 17596.5, 16954.3, 14234.3, 14234.3, 1034.66}}},
	}
}

// cylCylSeamResultBody resolves a case's result body in ANY grid — pinnedBody's grid coverage is
// simple/bfuseblend only, and E8 lives in complex.
func cylCylSeamResultBody(t *testing.T, grid, name string) *topo.Body {
	t.Helper()
	if grid == "simple" || grid == "bfuseblend" {
		return pinnedBody(t, grid, name)
	}
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == grid && r.Case == name {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Fatalf("%s/%s import-divergence: %v", grid, name, err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Fatalf("%s/%s picks could not be located on the imported body", grid, name)
	}
	res, okFillet, reason := runFillet(body, sets)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("%s/%s fillet unhealthy: ok=%v reason=%q results=%d", grid, name, okFillet, reason, len(res))
	}
	return res[0]
}

// TestCylCylSeamWatertight pins the closed-band results as watertight manifold solids at the
// DRAWEXE face-count join: every edge exactly 2-incident, the oracle's face count, 0 free edges +
// no retracing/self-crossing loops at BOTH gate qualities, and every face fold-free at both
// (a wrong-winding or gapped band cannot pass any of these).
func TestCylCylSeamWatertight(t *testing.T) {
	t.Parallel()
	for _, tc := range cylCylSeamPerFaceCases() {
		t.Run(tc.grid+"/"+tc.name, func(t *testing.T) {
			body := cylCylSeamResultBody(t, tc.grid, tc.name)
			assertWatertight(t, tc.name, body, len(tc.drawexe))
			assertBodyIsWatertightAndSimple(t, body, tc.grid+"/"+tc.name)
			for _, f := range body.Faces() {
				assertFaceFoldFreeAtEveryQuality(t, tc.grid+"/"+tc.name, f, nil)
			}
		})
	}
}
