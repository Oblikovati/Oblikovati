// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

func TestClassifyMirrorsOCCT(t *testing.T) {
	todo := Record{TODO: "TODO OCC22817 All:TEST INCOMPLETE"}
	if classify(todo, true, false, false) != SkipTODO {
		t.Fatal("TODO case must skip")
	}
	ok := Record{}
	if classify(ok, true, true, true) != Pass {
		t.Fatal("clean run must pass")
	}
	if classify(ok, true, true, false) != FailFaulty {
		t.Fatal("invalid solid must fail Faulty")
	}
	if classify(ok, false, false, false) != SkipImportDivergence {
		t.Fatal("import failure separates from fillet")
	}
}

// TestPassDeviationRollsUp checks the documented per-case deviation outcome names itself distinctly
// yet rolls into the pass tally (Pass and PassDeviation both count as a pass; failures/skips do not).
func TestPassDeviationRollsUp(t *testing.T) {
	if PassDeviation.String() != "PASS(deviation)" {
		t.Fatalf("PassDeviation String = %q", PassDeviation.String())
	}
	if !Pass.IsPass() || !PassDeviation.IsPass() {
		t.Fatal("Pass and PassDeviation must both count as a pass")
	}
	for _, o := range []Outcome{FailArea, FailFaulty, SkipTODO, SkipImportDivergence, Incomplete} {
		if o.IsPass() {
			t.Fatalf("%s must not count as a pass", o)
		}
	}
}
