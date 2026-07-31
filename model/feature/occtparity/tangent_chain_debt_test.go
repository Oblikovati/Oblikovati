// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestEveryPickBlendsItsWholeTangentChain is the corpus-wide RATCHET on the defect class the area
// gate is structurally blind to: OCCT's `blend` propagates a pick along its whole TANGENT-CONTINUOUS
// SPINE; wherever the fillet blends less than that spine, the harness compares our solid against a
// solid we never build — complex/D8 was the exemplar (OCCT 18 faces / 336159 against a single-edge
// 11 / 348183, an unreachable +3.577 % at deps 0.01).
//
// WHAT IT MEASURES. For every record whose input STEP imports, every pick that locates is expanded
// by occtTangentChain (this harness's INDEPENDENT port of OCCT ChFi3d_Builder::PerformElement,
// occt_tangent_chain_test.go) and the chain members the FILLET would not blend — neither picked by
// the record nor reached by the product's own spine propagation (ops.TangentEdgeChain, the walk
// every pick expands through in kernel/ops/fillet_chain_propagate.go) — are counted, UNIONED over
// the case's picks. The measurement is on the IMPORTED INPUT body, so it stays visible on a case
// whose fillet declines and ships no body at all. The oracle walker is deliberately NOT the product
// walker: the ratchet is a member-for-member cross-examination of the product's propagation against
// an independent port of OCCT's rule, on every record.
//
// The detection slice's 14-case / 51-edge debt population was RETIRED by that propagation (the
// walker's 1° antiparallel + same-convexity gates were aligned to PerformElement's FaceTangency +
// π/2 no-turn-back rule): the re-measured debt is ZERO corpus-wide, so the table below is empty and
// EVERY case is gated at zero unblended spine edges. Any walker regression or any new unpropagated
// spine fails loud; the counts are exact integers, so there is no float noise for a margin to absorb.
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
// members that the fillet would NOT blend: not picked by the record AND not reached by the
// product's own spine propagation (ops.TangentEdgeChain — the PerformElement walk every pick now
// expands through before the fillet dispatch, kernel/ops/fillet_chain_propagate.go). The oracle
// walker stays this harness's INDEPENDENT port; what retired the debt is the product walker
// reaching the same edges, measured member-for-member here on every record.
func unblendedChainEdges(body *topo.Body, r Record) int {
	picks := locatedPickEdges(body, r)
	covered := make(map[uint64]bool, len(picks))
	for _, e := range picks {
		covered[e.ID()] = true
		addProductSpineMembers(body, e, covered)
	}
	extra := map[uint64]bool{}
	for _, e := range picks {
		for _, c := range occtTangentChain(e) {
			if !covered[c.ID()] {
				extra[c.ID()] = true
			}
		}
	}
	return len(extra)
}

// addProductSpineMembers marks every edge the PRODUCT's spine walk reaches from seed — the edges
// the fillet actually widens the pick to.
func addProductSpineMembers(body *topo.Body, seed *topo.Edge, covered map[uint64]bool) {
	keys, _, err := ops.TangentEdgeChain(body, seed.ReferenceKey(), ops.DefaultTangentChainAngle)
	if err != nil {
		return
	}
	for _, k := range keys {
		if e, found := body.FindEdgeByKey(k); found {
			covered[e.ID()] = true
		}
	}
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

// TestProductSpineWalkerMatchesOraclePerPick is the walker-alignment gate the union-coverage
// ratchet above cannot carry alone: on a multi-pick case, several SHORTENED product chains can
// jointly cover one oracle spine (complex/C5's four picks cover its two 5-edge spines even under
// the old 1° gate), so the union measurement stays 0 while the walker itself is wrong. This gate
// cross-examines the walkers PER PICK, as edge SETS: every located pick of every importable record
// must expand to exactly the oracle's spine membership. Measured 0 mismatches corpus-wide;
// mutating ops.TangentEdgeChain's turn-back gate back to the 1° antiparallel band fails this loud
// on complex/C5's shortened chains.
func TestProductSpineWalkerMatchesOraclePerPick(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, r := range Corpus() {
		body, err := importInput(filepath.Join(dir, r.InputStep))
		if err != nil {
			continue
		}
		for pi, e := range locatedPickEdges(body, r) {
			assertPickSpineParity(t, r, pi, body, e)
		}
	}
}

// assertPickSpineParity fails unless the product walker's spine for pick e equals the oracle
// walker's spine as an edge set.
func assertPickSpineParity(t *testing.T, r Record, pi int, body *topo.Body, e *topo.Edge) {
	t.Helper()
	oracle := map[uint64]bool{}
	for _, c := range occtTangentChain(e) {
		oracle[c.ID()] = true
	}
	prod := map[uint64]bool{}
	addProductSpineMembers(body, e, prod)
	for id := range prod {
		if !oracle[id] {
			t.Errorf("%s/%s pick[%d]: product spine includes edge %d the oracle's does not (ops %d vs oracle %d edges)",
				r.Grid, r.Case, pi, id, len(prod), len(oracle))
			return
		}
	}
	if len(prod) != len(oracle) {
		t.Errorf("%s/%s pick[%d]: product spine has %d edges, oracle %d — the walkers disagree",
			r.Grid, r.Case, pi, len(prod), len(oracle))
	}
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

// knownTangentChainDebt is the measured population of picks whose OCCT blend spine runs past what
// the fillet blends. It is EMPTY: the 14-case / 51-edge population the detection slice recorded was
// fully RETIRED by pick propagation (kernel/ops/fillet_chain_propagate.go — every pick expands
// through ops.TangentEdgeChain, whose gates were aligned to PerformElement's FaceTangency + π/2
// no-turn-back rule, replacing the KNOWN-divergent 1° antiparallel band + same-convexity gates).
// Re-measured across every importable record after the alignment: 0 uncovered edges corpus-wide —
// including the two FAIL(area) deciders (complex/D8/F2, the closed 8-edge rim spine), the seven
// no-body faulties (complex/B9 C1 C5 F1, simple/P3 P7 U5), the pre-fillet skips (buildevol/K6,
// complex/E1 E2, encoderegularity/A5's 14-edge spine), and the acquitted simple/N4.
//
// The function stays so the ratchet keeps its shape: an EMPTY table gates every case at zero, and
// any walker regression (mutating the π/2 gate back to the 1° band re-lights complex/C5 and the
// D8/F2 family loud) or any new unpropagated spine fails immediately.
func knownTangentChainDebt() []tangentChainDebt {
	return nil
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
