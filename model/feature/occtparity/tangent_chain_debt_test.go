// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/topo"
)

// TestEveryPickBlendsItsWholeTangentChain is the corpus-wide RATCHET on the defect class the area
// gate is structurally blind to: OCCT's `blend` propagates a pick along its whole TANGENT-CONTINUOUS
// SPINE, and we blend the one picked edge. Wherever that spine is longer than the case's own pick
// list, the harness compares our solid against a solid we never built — complex/D8 is the exemplar
// (OCCT 18 faces / 336159 against our 11 / 348183, an unreachable +3.577 % at deps 0.01).
//
// WHAT IT MEASURES. For every record whose input STEP imports, every pick that locates is expanded
// by occtTangentChain (OCCT's own ChFi3d_Builder::PerformElement rule, ported in
// occt_tangent_chain_test.go) and the chain members the record does NOT pick are counted, UNIONED
// over the case's picks so an edge shared by two picks' chains counts once. The measurement is on
// the IMPORTED INPUT body — the solid the pick is resolved against and the one OCCT propagates over
// — so it stays visible on a case whose fillet is FAIL(faulty) and ships no body at all. Seven of
// the fourteen entries below are exactly that.
//
// ★ THE SWEEP OVERTURNED ITS OWN BRIEF. The class was expected to be full of accidental greens —
// cases passing on cancelling errors while scored against OCCT's longer chain. It is not. Of the 117
// green cases, exactly ONE (simple/N4) is in this population, and it is ACQUITTED face-for-face: a
// fresh DRAWEXE run emits 14 faces, we emit 14, and all fourteen pair to DRAWEXE's own
// 6-significant-figure print (80.7328/80.7749 … 10946.3/10946.3058). Its one unpicked chain edge is
// the other half of a seam-split rim our closed-rim path already blends whole. An independent
// corpus-wide check agrees and is stronger: DRAWEXE's result face count was measured for all 475
// cases, and only TWO greens differ from ours at all (simple/N6 9 vs 10, its known second-end
// defect; simple/U3 12 vs 11, +0.115 % over three faces vs our two) — neither is in this population.
//
// ONLY TWO ENTRIES ARE CURRENTLY DECIDING A VERDICT: complex/D8 and complex/F2, both FAIL(area), both
// with DRAWEXE emitting 18 faces to our 11. The other twelve either never ship a body (FAIL(faulty))
// or are skipped before the fillet runs. That bounds the blast radius honestly: the class costs the
// scoreboard two reds it could not otherwise reach, and zero false greens.
//
// It is a RATCHET, not a tolerance: a listed case may improve freely, but growing past its recorded
// count — or ANY unlisted case acquiring an unblended chain edge — fails loud. The counts are exact
// integers (a chain length is a count), so there is no float noise for a margin to absorb.
func TestEveryPickBlendsItsWholeTangentChain(t *testing.T) {
	dir := CorpusFixtureDir()
	debt := tangentChainDebtIndex()
	for _, r := range Corpus() {
		assertPicksCoverTheirChains(t, r, dir, debt[r.Grid+"/"+r.Case])
	}
}

// assertPicksCoverTheirChains fails when a case leaves more of its picks' OCCT spines unblended than
// its recorded debt. A record whose input does not import has no body to measure the pick on.
func assertPicksCoverTheirChains(t *testing.T, r Record, dir string, ceiling int) {
	t.Helper()
	body, err := importInput(filepath.Join(dir, r.InputStep))
	if err != nil {
		return
	}
	if n := unblendedChainEdges(body, r); n > ceiling {
		t.Errorf("%s/%s: OCCT propagates its blend along %d tangent-continuous edge(s) this case does "+
			"not pick, recorded debt %d — our result is being scored against a solid we do not build",
			r.Grid, r.Case, n, ceiling)
	}
}

// unblendedChainEdges is the union, over every located pick, of that pick's OCCT tangent-chain
// members that the record does not itself pick.
func unblendedChainEdges(body *topo.Body, r Record) int {
	picks := locatedPickEdges(body, r)
	picked := make(map[uint64]bool, len(picks))
	for _, e := range picks {
		picked[e.ID()] = true
	}
	extra := map[uint64]bool{}
	for _, e := range picks {
		for _, c := range occtTangentChain(e) {
			if !picked[c.ID()] {
				extra[c.ID()] = true
			}
		}
	}
	return len(extra)
}

// locatedPickEdges resolves every pick that can be located on the body. A pick that cannot is left
// out rather than failing the sweep: an unlocatable pick is a fixture gap (SKIP(import)), and the
// remaining picks still carry real, measurable chain debt.
func locatedPickEdges(body *topo.Body, r Record) []*topo.Edge {
	tol := importTol(body)
	out := make([]*topo.Edge, 0, len(r.Picks))
	for _, p := range r.Picks {
		if e, err := locateEdge(body, p.Locator, tol); err == nil {
			out = append(out, e)
		}
	}
	return out
}

// tangentChainDebt is one case's measured count of tangent-chain edges OCCT blends and we do not.
type tangentChainDebt struct {
	grid, name string
	unblended  int
}

// tangentChainDebtIndex keys the debt table by "grid/case".
func tangentChainDebtIndex() map[string]int {
	out := make(map[string]int, len(knownTangentChainDebt()))
	for _, d := range knownTangentChainDebt() {
		out[d.grid+"/"+d.name] = d.unblended
	}
	return out
}

// knownTangentChainDebt is the FULL measured population of picks whose OCCT blend spine runs past
// the case's own pick list: 14 cases of the 367 whose input imports, 51 unblended edges in all,
// from an instrumented sweep over every corpus record — not from any report.
//
// ★ EVERY ENTRY IS DETECTED, NOT CAUSED. The slice that landed this ratchet changed no production
// file: the scoreboard is byte-identical (112 simple / 117 all-grid) and the six existing ratchets
// are untouched. Every count is what base e88b2548 already ships.
//
// The entries, by what the debt currently COSTS:
//
//   - DECIDING A VERDICT (2). complex/D8 (7) and complex/F2 (7) are both FAIL(area) with DRAWEXE
//     emitting 18 faces to our 11 — the same closed 8-edge rim spine in both. D8's gap is +3.577 %
//     (348183.010 vs 336159, re-measured against a fresh DRAWEXE run here, confirming the exemplar's
//     figure); F2's is +4.682 % (51727.006 vs 49413.3). Neither can be closed by anything except
//     propagation — D8's own FAIL(area) is user-confirmed intentional for exactly this reason.
//   - SHIPPING NO BODY (7). complex/B9 C1 C5 F1, simple/P3 P7 U5 are FAIL(faulty): the fillet
//     declines or produces no solid, so the chain is not what decides them today. They are listed
//     because the debt is real on the input and will decide them the moment the fillet succeeds.
//   - SKIPPED BEFORE THE FILLET (4). buildevol/K6, complex/E1, complex/E2 are variable-radius;
//     encoderegularity/A5 is an OCCT TODO whose 17 picks include two we cannot locate (its 13 is the
//     located picks' debt, and it is the corpus's largest single spine at 14 edges).
//   - GREEN AND ACQUITTED (1). simple/N4 — see the header. It is listed, not exempted, because the
//     measurement is real; what the DRAWEXE face-for-face proves is that our closed-rim path already
//     builds the same solid, not that the chain is absent.
//
// ★ NOTHING HERE IS ROUTABLE TODAY, and that was measured, not assumed: re-running all ten
// measurable cases with every pick EXPANDED to its full chain makes all ten decline — none regresses
// to a wrong answer, all ten refuse. See tangent-chain-report.md §5 for which prerequisite blocks
// each (start-section bound on D8/F1/F2/C1, curved-miter arms on C5/N4/P7/U5, degenerate miter on
// B9, non-planar corner face on P3).
func knownTangentChainDebt() []tangentChainDebt {
	return []tangentChainDebt{
		{"complex", "D8", 7},           // FAIL(area) +3.577 % — the exemplar; DRAWEXE 18 faces to our 11
		{"complex", "F2", 7},           // FAIL(area) +4.682 % — the same closed 8-edge rim spine
		{"complex", "F1", 7},           // FAIL(faulty) — same spine, no body shipped
		{"encoderegularity", "A5", 13}, // SKIP(todo) — the corpus's longest spine (14 edges), 2 picks unlocatable
		{"complex", "C5", 4},           // FAIL(faulty) — 4 picks, two spines of 5
		{"complex", "C1", 2},           // FAIL(faulty)
		{"complex", "E1", 2},           // SKIP(varradius)
		{"complex", "E2", 2},           // SKIP(varradius)
		{"simple", "P7", 2},            // FAIL(faulty)
		{"complex", "B9", 1},           // FAIL(faulty)
		{"buildevol", "K6", 1},         // SKIP(varradius)
		{"simple", "N4", 1},            // PASS — acquitted face-for-face against DRAWEXE (14/14 faces)
		{"simple", "P3", 1},            // FAIL(faulty)
		{"simple", "U5", 1},            // FAIL(faulty)
	}
}

// TestTangentChainDebtTableIsWellFormed keeps the table honest: no duplicate key and no zero entry
// (one would be dead weight hiding nothing, since an unlisted case is already gated at zero).
func TestTangentChainDebtTableIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range knownTangentChainDebt() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate tangent-chain debt entry %s", key)
		}
		seen[key] = true
		if d.unblended <= 0 {
			t.Errorf("%s debt entry claims %d unblended edges — delete it instead", key, d.unblended)
		}
	}
}

// TestOCCTTangentChainDetectsAndDoesNotOverDetect is the detector's own falsifiable guard, in BOTH
// directions, on real corpus geometry rather than a synthetic fixture.
//
// POSITIVE: complex/D8's single pick must walk the whole closed 8-edge top rim. That 8 is not our
// own number — DRAWEXE builds 18 faces on that case where a single-edge blend builds 11, and the
// d8-multiface report attributes the surplus to exactly this spine.
//
// NEGATIVE: simple/N4's first and third picks must chain to themselves alone. Without a negative
// control a walker that returned "everything" would pass the positive one, and the whole population
// would be an artefact.
func TestOCCTTangentChainDetectsAndDoesNotOverDetect(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, r := range Corpus() {
		switch r.Grid + "/" + r.Case {
		case "complex/D8":
			assertChainLengths(t, r, dir, []int{8})
		case "simple/N4":
			assertChainLengths(t, r, dir, []int{1, 2, 1})
		}
	}
}

// assertChainLengths fails unless each of the case's located picks walks a chain of the wanted length.
func assertChainLengths(t *testing.T, r Record, dir string, want []int) {
	t.Helper()
	body, err := importInput(filepath.Join(dir, r.InputStep))
	if err != nil {
		t.Fatalf("%s/%s: input did not import: %v", r.Grid, r.Case, err)
	}
	picks := locatedPickEdges(body, r)
	if len(picks) != len(want) {
		t.Fatalf("%s/%s: located %d picks, want %d", r.Grid, r.Case, len(picks), len(want))
	}
	for i, e := range picks {
		if n := len(occtTangentChain(e)); n != want[i] {
			t.Errorf("%s/%s pick[%d]: OCCT spine walks %d edge(s), want %d", r.Grid, r.Case, i, n, want[i])
		}
	}
}
