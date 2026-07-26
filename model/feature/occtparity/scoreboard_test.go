// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"sort"
	"testing"
)

// Scoreboard tallies a fake evaluator's verdicts.
func TestScoreboardTally(t *testing.T) {
	recs := []Record{{Grid: "simple"}, {Grid: "simple"}, {Grid: "simple", TODO: "x"}}
	got := Scoreboard(recs, func(r Record) Outcome {
		if r.TODO != "" {
			return SkipTODO
		}
		return Pass
	})
	if got[Pass] != 2 || got[SkipTODO] != 1 {
		t.Fatalf("tally: %+v", got)
	}
}

// TestOCCTBlendScoreboard prints a per-grid PASS/FAIL/SKIP table over the real corpus and pins the
// HONEST rollup at the isWatertightSolid bar (watertight.go): the simple grid's green count, the
// all-grid green rollup, and SkipQuarantine, so a future gate change or corpus regeneration that
// silently drifts the count fails loud here. It is non-gating on individual CASE outcomes (the
// per-grid table tests in package feature_test are the parity gate — they must never be loosened) but
// IS gating on these aggregate rollup numbers.
//
// Hardened at the #2007 audit (green-gate-validate-audit-report.md): the area-only gate counted 5
// malformed-hole-loop false greens (S6/S9/T3/U3/U4) as PASS; isWatertightSolid closes that gap and
// quarantine.go holds them out. The honest rollup dropped 91→86 simple / 93→88 all-grid, and
// SkipQuarantine rose 1→6 (H6 + the 5). The single-boss setback tiling (#2007) then genuinely GREENED
// S6/S9/T3 (watertight, footprint absorbed), restoring 86→89 simple / 88→91 all-grid and dropping
// SkipQuarantine 6→3 (H6 + U3/U4). The dipArcOrder obstacle-path fix (u3-dipspast-report.md) then
// GREENED U3 (dipsPast was testing the wrong of the two crossing-bounded arcs), restoring 89→90 simple /
// 91→92 all-grid and dropping SkipQuarantine 3→2 (H6 + U4, the residual dual-host engine gap). The U4-5
// dual-host multi-rail weld then GREENED U4, restoring 90→91 simple / 92→93 all-grid and dropping
// SkipQuarantine 2→1 (H6 only). The concave bore-lip rim mirror (fillet_rim_concave.go's
// rimWithCapOrientation) then GREENED K1 (a genuine bore lip, R+r) and Z1 (a plain convex rim whose cap
// face is stored bottom-up — the SAME cap-orientation fix resolves it at R−r), restoring 91→93 simple /
// 93→95 all-grid with SkipQuarantine unchanged at 1.
func TestOCCTBlendScoreboard(t *testing.T) {
	dir := CorpusFixtureDir()
	byGrid := map[string]map[Outcome]int{}
	for _, r := range Corpus() {
		if byGrid[r.Grid] == nil {
			byGrid[r.Grid] = map[Outcome]int{}
		}
		byGrid[r.Grid][ScoreCase(r, dir)]++
	}

	grids := make([]string, 0, len(byGrid))
	for g := range byGrid {
		grids = append(grids, g)
	}
	sort.Strings(grids)

	order := []Outcome{Pass, PassDeviation, FailArea, FailFaulty, SkipTODO, SkipImportDivergence, Incomplete, SkipQuarantine}
	total := map[Outcome]int{}
	t.Logf("OCCT blend parity scoreboard (non-gating):")
	for _, g := range grids {
		line := g
		for _, o := range order {
			n := byGrid[g][o]
			total[o] += n
			line += "  " + o.String() + "=" + itoa(n)
		}
		t.Log(line)
	}
	sum := 0
	tline := "TOTAL"
	for _, o := range order {
		tline += "  " + o.String() + "=" + itoa(total[o])
		sum += total[o]
	}
	t.Log(tline)
	passRollup := 0
	for o, n := range total {
		if o.IsPass() {
			passRollup += n
		}
	}
	t.Logf("PASS rollup (Pass + PASS(deviation)) = %d", passRollup)
	if sum != len(Corpus()) {
		t.Fatalf("scoreboard counted %d cases, want %d", sum, len(Corpus()))
	}
	assertHardenedRollup(t, byGrid, passRollup, total[SkipQuarantine])
}

// assertHardenedRollup pins the honest rollup at the isWatertightSolid bar: 104 green in the
// simple grid, 109 green across all grids, and SkipQuarantine=1 (H6 only). The single-boss setback tiling
// greened S6/S9/T3 (86→89 / 88→91, SkipQuarantine 6→3); the dipArcOrder obstacle-path fix then greened
// U3 (89→90 / 91→92, SkipQuarantine 3→2); the U4-5 dual-host multi-rail weld then greened U4 (90→91 /
// 92→93, SkipQuarantine 2→1). The concave bore-lip rim mirror (fillet_rim_concave.go's
// rimWithCapOrientation, K1/Z1) then greened both: K1 is a genuine bore lip (the R+r mirror, material
// outside the bore) and Z1 turned out to be a plain CONVEX rim whose cap face happens to be stored
// bottom-up (capF.Reversed()==true) — solveRim's raw pl.Normal() offset the torus centre to the wrong
// side of the cap, failing its R−r probe for a reason unrelated to concavity; the same cap-orientation fix
// resolves it at R−r, not R+r (91→93 / 93→95), leaving only H6 (concave open-torus, ROOT 2) held. The
// R+r bore/notch-wall trihedral corner (corner-blend-weld Pieces 1+2) then GREENED N1 (box − r20
// cylinder notch; the engine used to mirror the corner into the void at R−r, now solves at R+r and
// welds pass-through faces to their rebuilt neighbours), restoring 93→94 simple / 95→96 all-grid. The
// bore far-cap OUTWARD extension (Piece 3) then GREENED L9 (box − r30 quarter-cylinder notch: the R+r
// torus far cross-section reaches past the rim, growing the adjacent flat faces outward instead of
// biting inward), restoring 94→95 simple / 96→97 all-grid, SkipQuarantine unchanged at 1. The
// mixed-sense curved-host 2r-torus corner WELD (corner-blend-weld Slice-1b, fillet_curved_mixed_weld.go)
// then GREENED M8 (box + boss, one convex Cyl∧Plane arm + a concave cove torus arm + a planar arm meeting
// at a curved-host trihedral vertex: no single ball is tangent to the boss wall at both R−r and R+r, so
// OCCT builds an analytic 2r-torus corner patch — this weld trims the three arm faces at the corner arcs,
// retrims the five bitten hosts, and assembles the 14-face solid), restoring 95→96 simple / 97→98
// all-grid, SkipQuarantine unchanged at 1. The concave BOSS-BASE rim cove (fillet_rim_concave.go's
// solveConcaveBossRim) then GREENED W6/W8 (simple) and A1 (bfuseblend): a boss filleted at its base built a
// watertight, DRAWEXE-exact-face-count solid but with solveRim's convex R−r round biting the reentrant
// corner INWARD (a STABLE +3.8%/+5.5% footprint over-size) instead of OCCT's concave R+r cove — the fix
// offsets the tube centre toward the boss and opens the plate hole to R+r, matching OCCT to −0.000%; a
// cap-fit gate leaves the deep #2012 spillers (R8/W9) on the unchanged ladder — restoring 96→98 simple /
// 98→101 all-grid, SkipQuarantine unchanged at 1. The elliptic-prism ruling fillet (fillet_ellipticalarm.go)
// then GREENED F4 (simple): a convex STRAIGHT ruling edge where a plane meets a right EllipticalCylinder
// wall (an oblique-prism side) was FLAT-REFUSED (curvedAdjacentError) because an elliptic host offset is a
// non-analytic canal — but on a STRAIGHT (∥-axis) edge the rolling-ball spine is a straight line, so the
// fillet collapses to an EXACT right circular cylinder; it is built analytically and welded through the
// existing single-arm curved runout with an ellipse-aware host retrim (fillet_curved_retrim_ellipse.go),
// restoring 98→99 simple / 101→102 all-grid, SkipQuarantine unchanged at 1. The asymmetric planar-miter
// seam (fillet_miter_asymmetric.go) then GREENED P9/V9 (simple): two filleted box-top edges of DIFFERENT
// radii (r1/r0.5) sharing a face were guard-rejected ("must use one radius") because the equal-radius miter
// samples cyl1 ∩ the nF1−nF2 mirror plane — a shortcut valid only when the two cylinders coincide there;
// unequal cylinders do not, so the seam is now the TRUE cyl0 ∩ cyl1 intersection (every point on both
// cylinders, welding the two arm faces watertight), matching OCCT's 8-face solid at −0.21%. The equal-radius
// corner path is untouched (routed only when radii differ), so every fingerprint pin stays byte-identical.
// This restored 99→101 simple / 102→104 all-grid, SkipQuarantine unchanged at 1. The CLOSED elliptic-rim
// canal band (fillet_elliptic_rim_canal.go) then GREENED J6/J8 (simple): a CLOSED rim where a plane meets an
// oblique-extrusion EllipticalCylinder wall was FLAT-REFUSED, because — unlike F4's straight ruling, whose
// spine is a line and whose fillet collapses to a right circular cylinder — a closed elliptic rim's
// rolling-ball spine is a closed NON-ANALYTIC curve, so the envelope is a genuine variable-section canal.
// It is built from the closed-form spine (exact centre + exact wall/plane feet per station), lofted with
// geom.LoftCanalStations and assembled through the SAME host-agnostic rim rebuild the analytic torus bands
// use. BOTH carry a per-case exact-deviation, so they score PassDeviation: OCCT's checkprops number comes
// from `sprops <shape> 1e-4`, which mis-integrates a SurfaceOfLinearExtrusion face by +2.4..4.7% (receipt:
// the UN-BLENDED J8 pipe wall is exactly 38201.978, OCCT plain sprops 38202.0, OCCT sprops 1e-4 39104.4) —
// OCCT's OWN blend measured with its accurate integrator is 77932.8 (J6) / 51078.3 (J8) against our exact
// 77931.429 / 51076.805. The SAME slice also greened bfuseblend A7/B1 outright (no deviation): they are the
// CONCAVE members of the vein — T5/U2's two shapes on a plate wide enough that the foot ring FITS — and they
// land on OCCT's own numbers at −0.041% / −0.020% with DRAWEXE-exact topology (9 faces / 11 vertices / 17
// edges). They green only because the convexity gate reads the SOLID (the imported elliptic face's Reversed
// flag mis-classifies them). This restored 101→103 simple / 104→108 all-grid, SkipQuarantine unchanged at 1.
// The GENERAL corner-weld layer (cornerweld_*.go) then GREENED N4 (simple): a full r20 cylinder standing on a
// 100³ box's vertical corner, filleted r5 on three edges meeting at one trihedral vertex. Its geometry (the
// four corner points, the two on-host rails, the NoFold-certified corner patch) had been committed and
// DRAWEXE-validated for four prior pieces; what was missing was the WELD, and it needed two stage variants
// that now live in the shared layer rather than in N4: (a) the convex cap-rim arm's far vertex is a G1 SEAM,
// not a cap — only the 90° piece of the 270° rim was picked and the rim continues tangentially across the boss
// wall's second face, so the arm runs THROUGH the seam to the end of the tangent chain (farRimContinuation);
// (b) crossing that seam splits the band into one face per host-face span, which is the C4 split. OCCT's own
// blend does exactly this (its oracle carries the band over all 270° as two faces, 76.3° + exactly 180°, and
// recedes BOTH wall faces to z=45), so the "cap-less rim far termination" the recon predicted is really a
// continuation. Ours: watertight 14-face solid, area 64287.18 vs OCCT 64287.2 (−0.000035%), volume 1046938.9
// vs DRAWEXE vprops 1.04694e6 (−0.000109%) — and reconciled PER FACE, since the corner fill became the true
// rolling-ball canal (the earlier +0.008% was two 27%/+29.7 face errors cancelling). This restored 103→104
// simple / 108→109 all-grid, SkipQuarantine unchanged at 1. Slice 2 of the same layer then GREENED O1 — the
// layer's own falsification test, and it passed as a pure CONFIGURATION (cornerweld_class_o1.go): a boss
// cylinder fused to a box, filleted r=5 on three edges meeting at one trihedral vertex, whose two CONCAVE
// arms (a cylinder arm and a cove torus arm, roll-sense regime R2 → R+r) terminate at the corner while the
// CONVEX planar band runs past it, so the corner patch is the rolling-ball canal of a ball riding the boss
// CYLINDER and rolling on the band's tube (the cylinder-on-cylinder sibling of N4's plane-on-torus frame).
// Ours: watertight 12-face solid, area 65101.43 vs OCCT 65104.9 (−0.0053%), volume 1111282.0 vs DRAWEXE
// vprops 1.11166e6 (−0.034%), reconciled per face — eight of twelve faces to ≤2.5e-5 relative, the three
// around the corner carrying OCCT's OWN patch approximation (its interior sits 1.16% of r off the exact
// envelope, and its implied rolling ball misses tangency to the boss wall by 0.174). This restored 104→105
// simple / 109→110 all-grid, SkipQuarantine unchanged at 1. (The mixed-radius TRIHEDRAL
// corner A4, r10/r5/r5, needs a torus corner patch — declined for now, a tracked follow-up.) A mismatch means
// either a false green slipped through or a case was over/under-quarantined — see harden-green-gate-brief.md's
// "STOP and report" instruction.
func assertHardenedRollup(t *testing.T, byGrid map[string]map[Outcome]int, allGridGreen, skipQuarantine int) {
	t.Helper()
	simpleGreen := byGrid["simple"][Pass] + byGrid["simple"][PassDeviation]
	if simpleGreen != 105 {
		t.Errorf("simple grid green (Pass+PassDeviation) = %d, want 105 (general corner-weld layer + N4 + O1)", simpleGreen)
	}
	if allGridGreen != 110 {
		t.Errorf("all-grid green (Pass+PassDeviation) = %d, want 110 (general corner-weld layer + N4 + O1)", allGridGreen)
	}
	if skipQuarantine != 1 {
		t.Errorf("SkipQuarantine = %d, want 1 (H6 only, #2007 U4 freed by U4-5)", skipQuarantine)
	}
}

// itoa avoids importing strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
