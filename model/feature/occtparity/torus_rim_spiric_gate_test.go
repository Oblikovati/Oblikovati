// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// Wave-E gates for the five torus-rim greens (capability-breadth wave, wave-report-E.md):
//
//   - simple/J5, bfuseblend/A5, bfuseblend/A6 — the CONCAVE closed-rim cove band on a TORUS host with
//     a latitude cap (fillet_torusarm_concave.go): the void ball rides a coaxial spine circle at
//     η = capH + r·n̂z with tube offset a + σ·s·r, where the tangency SIDE follows the tube's material
//     side (J5/A5 external a+r; A6 internal a−r — DRAWEXE decodes A6's plate hole at exactly
//     R − √((a−r)² − η²) = 136.7544).
//   - simple/J3, bfuseblend/A4 — the SPIRIC closed-rim canal (fillet_spiric_spine.go /
//     fillet_spiric_rim.go): a MERIDIAN cap (plane containing the torus axis) has no exact-torus arm;
//     the ball-centre spine is the closed spiric loop (offset plane ∩ offset torus), stationed in
//     closed form, lofted with geom.LoftCanalStations, welded through the same host-agnostic rim
//     rebuild as the elliptic band, and its host torus meshes through torusTubeBandLoftMesh (the
//     tube-wrapping transpose of the axis-wrapping closed band loft).
//
// Each green is gated on DRAWEXE 8.0.0 PER-FACE areas (sprops result_i 1e-12; fixtures run verbatim
// with their own explode variant — all five use plain `explode`), never on the whole-body area alone
// (per-face-oracle rule: whole-body gates admit wrong solids by cancellation). Rank pairing is sound:
// within every case the distinct face areas are separated by ≥8%, and the only equal entries are the
// box walls (exactly 40000/180000), where any pairing is the identity on the value.

// weTorusRimCase is one wave-E green's oracle table.
type weTorusRimCase struct {
	grid string
	tc   drawexeFaceCase
}

// weTorusRimCases carries the DRAWEXE per-face tables. Values are the converged `sprops result_i
// 1.e-12` masses (stable to all printed digits at 1e-10..1e-14 — the plateau was swept, not assumed).
func weTorusRimCases() []weTorusRimCase {
	return []weTorusRimCase{
		{"simple", drawexeFaceCase{
			name: "simple/J5",
			// ptorus 20 5 0 150 ×SCALE1; blend r=10 on the crater rim: base disc, host torus, shrunk
			// crater floor (π·151.266²), cove band.
			drawexe:    []float64{196350, 167422, 71884.1, 9271.61},
			totalArea:  444927.7, // Σ of the 1e-12 per-face masses (checkprops' 444925 is the looser 1e-4 read)
			perFaceTol: 1e-4,     // measured worst face: −0.0015% (J5 #4); ~6× margin, 100× inside OCCT's 1%
		}},
		{"simple", drawexeFaceCase{
			name: "simple/J3",
			// ptorus 20 5 270 ×SCALE1; blend r=10 on the u=270° cap-disc rim: host torus, full u=0 cap
			// disc (π·50²), spiric-trimmed cap (5033.06), spiric canal band.
			drawexe:    []float64{292961, 7853.98, 5033.06, 4562.47},
			totalArea:  310410.5,
			perFaceTol: 1e-4, // measured worst face: −0.0044% (the band, rel 4.4e-5) — ~2× margin
		}},
		{"bfuseblend", drawexeFaceCase{
			name: "bfuseblend/A4",
			// box + ptorus 20 5 270 fused at the box wall (wall ∥ torus axis): CONCAVE spiric cove.
			drawexe:    []float64{292920, 40000, 40000, 40000, 40000, 40000, 28675.1, 7853.98, 5322.75},
			totalArea:  534771.8,
			perFaceTol: 1e-4,
		}},
		{"bfuseblend", drawexeFaceCase{
			name: "bfuseblend/A5",
			// box + ptorus 20 5 0 90 (quarter ring on the plate): external-tangency cove, plate hole
			// grows 250 → 259.16.
			drawexe:    []float64{360000, 180000, 180000, 180000, 180000, 148997, 125664, 101265, 22327.9},
			totalArea:  1478253.9,
			perFaceTol: 1e-4,
		}},
		{"bfuseblend", drawexeFaceCase{
			name: "bfuseblend/A6",
			// box + treversed ptorus 20 10 90 180 (flared funnel): INTERNAL-tangency cove (a−r), plate
			// hole grows 100 → 136.754.
			drawexe:    []float64{360000, 301247, 180000, 180000, 180000, 180000, 125664, 105830, 43830.2},
			totalArea:  1656571.2,
			perFaceTol: 1e-4,
		}},
	}
}

// TestWaveETorusRimPerFaceAgainstDrawexe rank-pairs each wave-E green's SHIPPED per-face mesh areas
// against DRAWEXE and requires the face-count join, per-face agreement, and the summed total.
func TestWaveETorusRimPerFaceAgainstDrawexe(t *testing.T) {
	for _, c := range weTorusRimCases() {
		t.Run(c.tc.name, func(t *testing.T) {
			body := weTorusRimBody(t, c.grid, c.tc.name)
			assertShippedPerFaceAgainstDrawexe(t, c.tc, body, nil, 5e-5)
		})
	}
}

// TestWaveETorusRimWatertight requires each wave-E green to be a watertight manifold solid at
// DRAWEXE's exact face count, with positive volume and ZERO free edges at BOTH gate qualities.
func TestWaveETorusRimWatertight(t *testing.T) {
	for _, c := range weTorusRimCases() {
		t.Run(c.tc.name, func(t *testing.T) {
			body := weTorusRimBody(t, c.grid, c.tc.name)
			assertWatertight(t, c.tc.name, body, len(c.tc.drawexe))
			assertPositiveVolume(t, c.tc.name, body)
			for _, gq := range gateQualities() {
				if n := ops.FreeEdgeCount(ops.CalculateBodyFacets(body, gq.q).Mesh); n != 0 {
					t.Errorf("%s: %d free edges at %s quality, want 0", c.tc.name, n, gq.name)
				}
			}
		})
	}
}

// weTorusRimBody resolves one wave-E case's result body through the real fillet feature, failing
// (not skipping) on any import/locate/fillet gap — these are GREEN cases.
func weTorusRimBody(t *testing.T, grid, qualified string) *topo.Body {
	t.Helper()
	name := qualified[len(grid)+1:]
	var rec Record
	for _, r := range Corpus() {
		if r.Grid == grid && r.Case == name {
			rec = r
		}
	}
	body, err := importInput(filepath.Join(CorpusFixtureDir(), rec.InputStep))
	if err != nil {
		t.Fatalf("%s import-divergence: %v", qualified, err)
	}
	sets, ok := scoreLocate(rec, body)
	if !ok {
		t.Fatalf("%s picks could not be located on the imported body", qualified)
	}
	res, okFillet, reason := runFillet(body, sets)
	if !okFillet || len(res) != 1 || res[0] == nil {
		t.Fatalf("%s fillet unhealthy: ok=%v reason=%q results=%d", qualified, okFillet, reason, len(res))
	}
	return res[0]
}
