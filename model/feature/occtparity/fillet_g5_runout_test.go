// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

// TestG5RunoutCasesPass is the Task-8 regression gate for the n-valent fillet-runout feature
// (G5 first slice, #ADR-0050): the valence-5 corpus case (simple/V3) that motivated the whole
// feature must score PASS — a valid closed solid whose area is within OCCT's own tolerance
// (assertArea/areaWithin, r.Deps). simple/V5 (the valence-6 case) is deliberately NOT gated
// here: Task 8's investigation found V5 now builds a valid solid (it moved off FailFaulty since
// the pre-feature baseline) but its area is ~3.24% off OCCT's reference — outside the ~1% gate —
// and every far face at its runout vertex is geom.Plane, so the anticipated "non-planar far
// face" deferral reason does not hold; the residual is a genuine, still-open valence-6 spread
// defect. Per the Task-8 brief's own contingency, V3 alone is a complete, honest increment.
// TestG5V5StillFailsArea below is the tripwire: it fails loudly the day V5's outcome changes,
// so nobody has to remember to revisit this comment.
func TestG5RunoutCasesPass(t *testing.T) {
	dir := CorpusFixtureDir()
	rec := findCorpusRecord(t, "simple", "V3")
	if got := ScoreCase(rec, dir); got != Pass {
		t.Errorf("simple/V3: ScoreCase = %v, want Pass", got)
	}
}

// TestG5V5StillFailsArea pins simple/V5's known-open outcome (FailArea, not gated into
// TestG5RunoutCasesPass — see its doc comment) so a change either direction is loud: if this
// starts failing because V5 now scores Pass, promote it into the hard gate and update both
// comments; if it regresses to FailFaulty or an import/skip, that is a genuine defect to
// investigate before touching this test.
func TestG5V5StillFailsArea(t *testing.T) {
	dir := CorpusFixtureDir()
	rec := findCorpusRecord(t, "simple", "V5")
	if got := ScoreCase(rec, dir); got != FailArea {
		t.Errorf("simple/V5: ScoreCase = %v, want FailArea (known-open valence-6 gap; update this test's comment if this is an intentional change)", got)
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
