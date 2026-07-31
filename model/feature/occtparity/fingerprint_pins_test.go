// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/topo"
)

// TestByteIdentityFingerprints pins the canonical mesh fingerprint + volume of the prior-green corner
// bodies the CN-C8 brief names for do-no-harm (B3, N7, C2, C6, D5, D9, E4) plus C8 itself. CN-C8 changes no
// shared weld/retrim/geometry code (it is verification + a forensic receipt), so every one of these must
// stay bit-identical to the base worktree 6123169d. These VALUES were captured on base 6123169d and re-run
// on the CN-C8 HEAD; if a future shared-code edit perturbs any of them, this fails loud with the delta. The
// RAW volumes match the brief's references exactly (B3 190756.470897507, N7 963883.383205631).
func TestByteIdentityFingerprints(t *testing.T) {
	for _, tc := range byteIdentityPins() {
		t.Run(tc.name, func(t *testing.T) {
			fp := bodyMeshFingerprint(pinnedBody(t, tc.grid, tc.name))
			if fp.Hash != tc.hash || fp.Triangles != tc.tris {
				t.Fatalf("%s fingerprint drifted: hash=%#x tris=%d, want hash=%#x tris=%d (shared geometry changed)",
					tc.name, fp.Hash, fp.Triangles, tc.hash, tc.tris)
			}
			// RAW volume compared at rel volTolFor(name) — 1e-9 for a well-conditioned body (re-run noise
			// ~1e-13, ~4 decades of margin). A body with heavy signed-tetra cancellation (M4's off-origin
			// bore) is order-sensitive at ~1e-4: TessellateBody yields the SAME triangle SET (hash+tris stay
			// EXACT) in a map-order-dependent SEQUENCE and FP addition is non-associative — a pre-existing
			// kernel property, not a fillet defect (N3/N9, centred boss geometry, stay bit-stable at 1e-9).
			if rel := relErr(fp.Volume, tc.vol); rel > volTolFor(tc.name) {
				t.Fatalf("%s volume %.12f != base %.12f (rel %.3g, tol %.1g)", tc.name, fp.Volume, tc.vol, rel, volTolFor(tc.name))
			}
		})
	}
}

// volTolFor is the raw-volume relative tolerance for a pinned body: the bit-stable 1e-9 default, loosened
// for the two cases whose signed-tetra volume sum is TessellateBody-order-sensitive (their mesh — hash+tris
// — is still pinned exactly): M4's off-origin bore (~1e-4) and D3's host-sphere near-degenerate zone winding
// (~1e-3 swing; 3e-3 leaves ~3× margin over the observed spread).
func volTolFor(name string) float64 {
	switch name {
	case "M4":
		return 5e-4
	case "D3":
		return 3e-3
	}
	return 1e-9
}

// fingerprintPin is one body's captured order-independent fingerprint. grid is the corpus grid the case
// lives in ("" means "simple"); it disambiguates cases whose name collides across grids (e.g. A2/A3 exist
// in BOTH simple and bfuseblend).
type fingerprintPin struct {
	name string
	vol  float64
	tris int
	hash uint64
	grid string // "" means the "simple" grid; set for cross-grid name collisions (A2/A3 in bfuseblend)
}

// pinnedBody resolves a pinned case's result body in its grid (defaulting to "simple"), running the real
// fillet feature — the grid-aware counterpart of caseResultBody.
func pinnedBody(t *testing.T, grid, name string) *topo.Body {
	t.Helper()
	if grid == "bfuseblend" {
		return bfuseblendResultBody(t, name)
	}
	return caseResultBody(t, name)
}

// byteIdentityPins are the base-6123169d fingerprints, captured by running bodyMeshFingerprint in a git
// worktree at 6123169d and confirmed bit-identical on the CN-C8 HEAD (see cnc8-report.md §byte-identity)
// and again on the CN6 HEAD (cn6-report.md §byte-identity; CN6 is a test-harness-only change).
//
// CROSS-PLATFORM RISK (record only — do NOT fix here; for the pre-PR cross-platform-stability pass,
// CLAUDE.md hard-coded-values rule): these are hard-coded 64-bit hashes of quantized tessellated
// coordinates. arm64 FMA/libm rounding could shift a vertex across a 1e-6·scale quantization boundary
// and flip a hash → a FALSE red off linux/amd64. Fine for the corpus's linux/amd64 runs. The pre-PR
// pass must re-capture on arm64 and either confirm identical or widen the quantum / drop the hash pins
// to the (rounding-robust) volume+triangle-count pins there. No arm64 re-capture is attempted here.
//
// RE-CAPTURED for the far-end wall trim + the loop-arc alignment pass (fillet_farend_trim.go and
// fillet_survivor_rim.go's alignCarriedArcsToSegments; farend-runon-report.md §5): ten pins moved —
// A6 A8 B1 B9 D3 E1 E2 I3 M2 N5 — because their fillet band's far end had been squared off at the flank's
// parametric extreme instead of trimmed against the curved host wall, and because a carried rim arc could
// sweep past the loop's own vertex. Every one was PROVEN improved against DRAWEXE 8.0.0 before re-capture.
// Summed per-face |Δ| vs OCCT's own `sprops` receipts: B1 539.17→0.31, B9 1074.57→0.46, D3 2564.08→15.81,
// E1 529.57→13.15, E2 2396.65→12.91, M2 2.29→0.12, A6 10.17→0.15, A8 55.83→0.14, I3 818.72→0.36,
// N5 124.70→0.44. Whole-body area error: B1 −0.0071%→−0.0015%, B9 −0.349%→−0.0012%, D3 +0.290%→−0.0121%,
// E1 +0.367%→−0.0097%, E2 +0.163%→−0.0091%, A8 −0.191%→−0.0004%, I3 +0.825%→−0.00002%. Mesh VOLUME vs
// DRAWEXE `vprops` improved on nine — D3 −0.787%→−0.047%, N5 −0.115%→−0.0008%, A8 −0.256%→−0.001%,
// I3 +0.075%→−0.002%, A6 −0.027%→−0.0006%, B9 −0.054%→−0.008%, B1 −0.027%→−0.006%, E1 −0.038%→−0.037%,
// M2 flat at −0.000% — and moved from −0.0055% to −0.036% on E2 alone, whose per-face areas came 186×
// closer (its worst face was +1201.79 before, now −12.50), so that one delta is mesh-side, not geometry.
// ★ RE-CAPTURED for the patchGridCap density fix (patchgridcap-report.md): spherePatchMesh's silent
// 80-steps-per-axis grid clamp floored 25 shipped simple-grid cases at PropertyQuality; the grid now
// honours the chord tolerance up to the diagnosed patchGridCellBudget. NINETEEN pins moved — exactly
// the pinned ∩ saturated set (A6 A8 B1 B9 C6 C8 D3 D4 D5 D8 D9 E1 E2 E3 E4 K6 L4 L9 N1; every other
// pin bit-identical, which also proves the trimScan filter's bit-equivalence). Each was PROVEN
// improved against live DRAWEXE 8.0.0 (`sprops <face>/result` and `vprops result` at `1.e-12`) before
// re-capture; captured twice, bit-stable. Saturated sphere-face mesh area, clamped → honest, vs the
// DRAWEXE face row: A6 157.055→157.0745 (157.08), A8 195.094→195.1246 (195.13), B1/B9
// 157.058→157.0744 (157.08), D3 61118.28→61133.48 (61133.7) + corner→157.0744, D4 57830.73→57844.76
// (57845), D5 57523.39→57536.92 (57537.1) + 55.784→55.7868 (55.7891), D8 182430.71→182628.46
// (182633), D9 179109.48→179288.47 (179292), E1 65916.57→65929.07 (65929.3), E2 65834.80→65847.07
// (65847.3), E3 61729.18→61740.69 (61740.9), E4 57609.34→57645.61 (57646), L9 28.1939→28.1950
// (28.1971), N1 34.2318→34.2335 (34.236), K6/L4 39.2645→39.26726 (39.2699 = 25π/2, on top of the
// §region octant fix). Raw volume vs `vprops`, relative: D3 −4.7e-4→−8.9e-6, D4 −4.4e-4→−1.6e-5,
// D5 −4.2e-4→−1.0e-5, D8 −1.96e-3→−4.1e-5, D9 −1.79e-3→−3.6e-5, E1 −3.7e-4→−7.3e-6, E2
// −3.6e-4→−9.4e-6, E3 −3.4e-4→−1.2e-5, E4 −1.1e-3→−2.2e-5, A6/A8/B1/B9/L9/N1/K6/L4 toward vprops
// likewise. C6's corner ball (cone-host, OCCT ships a BSpline 145.583 approximation where ours is the
// exact sphere) moves toward ITS converged form 144.288→144.304; its body area 89364.91 vs sprops
// 89366.1 and volume −1.3e-4→−6.2e-5 corroborate. C8/D1 are the documented deviation cases (OCCT's
// own result flawed): D1's corner cap lands 238.479 on the exact ~238.5 spherical triangle and its
// body area 10078.811 on the recorded ExactArea 10078.8; C8's body area reads 9781.544 vs ExactArea
// 9781.45 (the recorded constant carries the old clamped sphere term — see the report's concerns).
// Triangle growth is the saturated faces' densification alone (e.g. D5 64630→481302: its two
// saturated faces' honest grids).
//
// It is assembled from two per-family records, split out so no single file carries the whole ledger
// (CLAUDE.md's 500-line rule): curvedWeldPins (fingerprint_pins_curved_test.go) and setbackObstaclePins
// (fingerprint_pins_setback_test.go). The two lists are disjoint and their union is the pin set.
func byteIdentityPins() []fingerprintPin {
	return append(curvedWeldPins(), setbackObstaclePins()...)
}
