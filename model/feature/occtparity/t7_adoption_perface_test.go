// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestT7PerFaceRankPairsAgainstNexplodeOracle is simple/T7's per-face DRAWEXE reconciliation — the
// NINTH fully-proven case of the nil-curve-offer adoption (fingerprint_pins_setback_test.go's
// receipt covers the other eight).
//
// ★ T7 was EXCLUDED from the adoption slice's rank-pair evidence on a mis-run oracle: the fixture
// `OCCT/tests/blend/simple/T7` picks its blend edge with `nexplode s E`, the slice ran `explode s E`,
// and the silently renumbered pick blended a different edge — an 11-face body reading 7507.53 that
// was then committed as "a pre-existing topology divergence, nothing to rank-pair". Run with the
// fixture's own `nexplode` (perface_oracle_test.go's ★★ explode-variant note), OCCT's blend is a
// valid 15-face solid, `sprops result 1.e-12` = 7482.15 (rel err 9.3e-16), and it rank-pairs
// 15-to-15 against ours. There is no topology divergence and no oracle-internal disagreement.
//
// Evidence (fix wave, live DRAWEXE 8.0.0, per-face `sprops result_i 1.e-12`; ours =
// ops.MeshArea ∘ ops.TessellateFace at PropertyQuality, shipped == solo on this body): summed
// per-face |Δ| falls 7.0374 (base, adoption reverted) → 4.7997 (HEAD), 1.47×, NO face regressing.
// The faces that move are exactly the adopted-boundary ones: the mid runout BSpline 65.588873 →
// 64.186993 against 64.1795 (188×), the r=8 boss wall 602.613513 → 603.044883 against 603.186, the
// r=6 fillet twins 87.593199/87.603925 → 87.532003/87.532004 against 87.536 each, the BSpline twins
// 33.275269/33.337713 → 33.210881/33.210883 against 33.2141 each, and the extrusion wall
// 2380.841215 → 2380.950144. Rank pairing is sound: the closest DISTINCT pair (471.239 vs 492.274)
// is 4.3% apart while the largest per-face residual is 0.30%, and the three equal-area pairs are
// mirror twins, interchangeable under a size ordering by the fixture's own symmetry.
func TestT7PerFaceRankPairsAgainstNexplodeOracle(t *testing.T) {
	body := pinnedBody(t, "simple", "T7")
	assertShippedPerFaceAgainstDrawexe(t, t7NexplodeFaceCase(), body, t7PerFaceDebt(), 3.06e-4)
}

// t7NexplodeFaceCase is DRAWEXE 8.0.0's per-face record for simple/T7, captured live in the fix
// wave with the fixture's own commands (`restore CFI_6_f56fhc.rle s ; tscale s 0 0 0 1 ;
// nexplode s E ; blend result s 6 s_8`), faces exploded from the RESULT and each read at
// `sprops result_i 1.e-12` (per-face rel err ≤ 9.4e-16), identified by `mksurface ; dump`,
// descending: the oblique elliptic-prism extrusion wall, two untouched box planes (1000 / 800), the
// r=8 survivor boss cylinder, the two 492.274 box-side planes, a 471.239 box plane, the reshaped
// front (399.178) and top (333.086) host planes, the r=8 boss end cap, the two r=6 fillet-segment
// cylinder twins, and the three rational-BSpline runout patches (64.1795 mid + 33.2141 twins).
// Σ faces = 7482.1487, agreeing with the whole-body 7482.15.
func t7NexplodeFaceCase() drawexeFaceCase {
	return drawexeFaceCase{
		name: "T7", totalArea: 7482.15, perFaceTol: 3e-4,
		drawexe: []float64{2384.17, 1000, 800, 603.186, 492.274, 492.274, 471.239, 399.178,
			333.086, 201.062, 87.536, 87.536, 64.1795, 33.2141, 33.2141},
	}
}

// t7PerFaceDebt is the per-rank ratchet (1.1× each measured residual; shrink-only) for the three
// faces carrying a measured, separately-rooted shortfall the adoption does not own:
//
//   - rank 1, the trimmed oblique elliptic EXTRUSION wall (ours 2380.950144, −1.35e-3): DRAWEXE's
//     own quadrature is UNCONVERGED on this surface family — T6's untrimmed twin of the same
//     prism reads 2381.68 @1e-6 vs 2384.17 @1e-12 against a 2381.677340 closed form
//     (perface_oracle_test.go), so the oracle's last digits are its quadrature, not the face;
//   - ranks 8/9, the reshaped front/top HOST planes (ours 399.537566 +9.0e-4, 334.081647 +3.0e-3):
//     the pre-existing plane-trim sliver class (curve-adoption-report.md concern 2 — T3's 1204.6
//     plane is the same family), byte-identical base → HEAD, untouched by the adoption.
func t7PerFaceDebt() map[int]float64 {
	return map[int]float64{0: 1.49e-3, 7: 9.92e-4, 8: 3.29e-3}
}
