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
// ★ Re-measured by the FOOTPRINT-RIM CURVE CARRY (t3-plane-sliver-report.md): the reshaped hosts and
// the elliptic wall were bounded by inscribed CHORDS of the boss footprint conics; those segments now
// carry exact sub-spans, and the residuals collapse — front host 399.537566 → 399.192709 (rel 3.7e-5),
// top host 334.081647 → 333.101499 (rel 4.7e-5), elliptic wall 2380.950144 → 2381.645098, which is
// 1.35e-5 of the ∮|C′(u)×v| du closed form 2381.677340 (the DRAWEXE row 2384.17 is that surface
// family's unconverged quadrature, perface_oracle_test.go). The whole-body budget tightened 3.06e-4 →
// 1.24e-5 against the closed-form-adjudicated total (t7NexplodeFaceCase).
func TestT7PerFaceRankPairsAgainstNexplodeOracle(t *testing.T) {
	body := pinnedBody(t, "simple", "T7")
	assertShippedPerFaceAgainstDrawexe(t, t7NexplodeFaceCase(), body, t7PerFaceDebt(), 1.24e-5)
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
//
// ★ totalArea is the CLOSED-FORM-ADJUDICATED sum 7482.15 − 2384.17 + 2381.677340 = 7479.657340:
// rank 1's `sprops … 1.e-12` reading 2384.17 is UNCONVERGED quadrature on the trimmed oblique
// quadric family (its untrimmed twin reads 2355.61 / 2393.32 / 2384.17 at 1e-6 / 1e-9 / 1e-12
// against the exact ∮|C′(u)×v| du = 2381.677340 — perface_oracle_test.go's caveat: adjudicate a
// disagreement against a CLOSED FORM, never another DRAWEXE tolerance). The old 7482.15 total only
// ever passed because our hosts' chord surplus (+1.36) partially cancelled the oracle's own wall
// error (+2.49); with the rim chords fixed the cancellation is gone and the honest target is the
// adjudicated one — against which the budget RATCHETS 3.06e-4 → 1.24e-5. The per-face rank rows
// stay DRAWEXE's own readings; rank 1's oracle error lives in its debt ceiling, not the row.
func t7NexplodeFaceCase() drawexeFaceCase {
	return drawexeFaceCase{
		name: "T7", totalArea: 7479.657340, perFaceTol: 3e-4,
		drawexe: []float64{2384.17, 1000, 800, 603.186, 492.274, 492.274, 471.239, 399.178,
			333.086, 201.062, 87.536, 87.536, 64.1795, 33.2141, 33.2141},
	}
}

// t7PerFaceDebt is the per-rank ratchet (1.1× each measured residual; shrink-only) for the three
// faces whose rows carry a named, separately-rooted residual:
//
//   - rank 1, the trimmed oblique elliptic EXTRUSION wall (ours 2381.645098, rel 1.06e-3 vs the
//     row): the residual is the ORACLE's — DRAWEXE's quadrature is UNCONVERGED on this surface
//     family (2355.61 / 2393.32 / 2384.17 at 1e-6 / 1e-9 / 1e-12 on the untrimmed twin vs the
//     2381.677340 closed form, perface_oracle_test.go); ours sits 1.35e-5 from the closed form.
//     Ceiling 1.49e-3 → 1.17e-3 with the rim-chord share now gone;
//   - ranks 8/9, the reshaped front/top HOST planes: the plane-trim sliver class CLOSED by the
//     footprint-rim curve carry (t3-plane-sliver-report.md) — 399.537566 → 399.192709 (rel
//     3.69e-5) and 334.081647 → 333.101499 (rel 4.65e-5). Ceilings 9.92e-4 → 4.06e-5 and
//     3.29e-3 → 5.12e-5: tighter than the 3e-4 default, kept as entries so a chord regression
//     on exactly these faces fails loud at its own measured scale.
func t7PerFaceDebt() map[int]float64 {
	return map[int]float64{0: 1.17e-3, 7: 4.06e-5, 8: 5.12e-5}
}
