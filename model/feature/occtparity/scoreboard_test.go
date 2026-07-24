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

// assertHardenedRollup pins the honest rollup at the isWatertightSolid bar: 95 green in the
// simple grid, 97 green across all grids, and SkipQuarantine=1 (H6 only). The single-boss setback tiling
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
// all-grid, SkipQuarantine unchanged at 1. A mismatch means either a false green slipped through or a case
// was over/under-quarantined — see harden-green-gate-brief.md's "STOP and report" instruction.
func assertHardenedRollup(t *testing.T, byGrid map[string]map[Outcome]int, allGridGreen, skipQuarantine int) {
	t.Helper()
	simpleGreen := byGrid["simple"][Pass] + byGrid["simple"][PassDeviation]
	if simpleGreen != 96 {
		t.Errorf("simple grid green (Pass+PassDeviation) = %d, want 96 (corner-blend-weld M8: mixed-sense 2r-torus curved corner)", simpleGreen)
	}
	if allGridGreen != 98 {
		t.Errorf("all-grid green (Pass+PassDeviation) = %d, want 98 (corner-blend-weld M8: mixed-sense 2r-torus curved corner)", allGridGreen)
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
