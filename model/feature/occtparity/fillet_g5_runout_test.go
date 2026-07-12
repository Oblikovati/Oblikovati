// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestG5RunoutCasesPass is the regression gate for the n-valent fillet-runout feature (G5,
// #ADR-0050): every wired runout corpus case must score PASS — a valid closed solid whose area is
// within OCCT's own tolerance (assertArea/areaWithin, r.Deps).
//
//   - simple/V3 (valence-5): the case that motivated the feature (Task 8).
//   - simple/V5 (valence-6) and simple/V1 (valence-4, symmetric wedge): promoted here once the
//     rail-termination setback (fillet_runout_setback.go) closed their area gap. Both had scored
//     FailArea — V5 ~+3.2%, V1 ~+1% — because each flank tangent rail ran out to the pick-edge end
//     vertex's axial projection (an apex at d>r, OUTSIDE the fillet tube), overshooting the runout.
//     OCCT terminates each rail where its generator pierces the ADJACENT far plane; sliding both
//     rails' ends to that pierce (V5's fan end + start end, V1's fan end + trihedral start) drops
//     each into OCCT's area tolerance while leaving the runout interior (the near-root splits)
//     untouched. See .superpowers/sdd/v5-setback-characterization.md and task-3-report.md.
func TestG5RunoutCasesPass(t *testing.T) {
	dir := CorpusFixtureDir()
	for _, c := range []string{"V1", "V3", "V5"} {
		if got := ScoreCase(findCorpusRecord(t, "simple", c), dir); got != Pass {
			t.Errorf("simple/%s: ScoreCase = %v, want Pass", c, got)
		}
	}
}

// findCorpusRecord locates one corpus record by grid+case, failing loudly rather than silently
// scoring a zero Record if the fixture/corpus set ever changes underneath this gate.
func findCorpusRecord(t *testing.T, grid, caseName string) Record {
	t.Helper()
	for _, r := range Corpus() {
		if r.Grid == grid && r.Case == caseName {
			return r
		}
	}
	t.Fatalf("corpus record %s/%s not found", grid, caseName)
	return Record{}
}
