// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import "testing"

type fakeT struct{ errs, logs []string }

func (f *fakeT) Errorf(s string, a ...any) { f.errs = append(f.errs, s) }
func (f *fakeT) Logf(s string, a ...any)   { f.logs = append(f.logs, s) }

func TestAssertAreaTolerance(t *testing.T) {
	// within 1% => no error; >0.1% => a drift warning
	f := &fakeT{}
	assertArea(f, "c", 100.5, 100.0, 0.01) // 0.5% off: pass, warn
	if len(f.errs) != 0 {
		t.Fatalf("unexpected error at 0.5%%: %v", f.errs)
	}
	if len(f.logs) != 1 {
		t.Fatalf("expected drift warning at 0.5%%, got %d", len(f.logs))
	}
	// >1% => error
	f2 := &fakeT{}
	assertArea(f2, "c", 102.0, 100.0, 0.01) // 2% off
	if len(f2.errs) != 1 {
		t.Fatalf("expected error at 2%%, got %d", len(f2.errs))
	}
	// within 0.1% => no error, no warning
	f3 := &fakeT{}
	assertArea(f3, "c", 100.05, 100.0, 0.01) // 0.05% off
	if len(f3.errs) != 0 || len(f3.logs) != 0 {
		t.Fatalf("clean case warned/errored: %v %v", f3.errs, f3.logs)
	}
}

// c8Deviation is the C8 record fixture the deviation-gate tests exercise: OCCT area 9640.68, OUR
// exact area 9781.45 (+1.46%). It mirrors the corpus.json entry so the gate is tested on the real
// override shape, not an inline stub.
func c8Deviation() Record {
	return Record{Grid: "simple", Case: "C8", ExpectedArea: 9640.68, Deps: 0.01,
		Deviation: &CaseDeviation{ExactArea: 9781.45, OCCTArea: 9640.68, Reason: "OCCT non-tangent sag"}}
}

// TestAssertCaseAreaDeviation proves the per-case deviation override is a REAL regression gate, not
// a blanket skip (occt-oracle-not-religion). A deviation case PASSES only when the built area
// matches OUR exact area (and the receipt is logged); it FAILS at OCCT's flawed area; and a >1%
// nudge of the exact area FAILS — a todo-skip would gate none of these.
func TestAssertCaseAreaDeviation(t *testing.T) {
	dev := c8Deviation()

	f := &fakeT{} // built == OUR exact area: PASS, receipt logged
	assertCaseArea(f, "simple/C8", 9781.45, dev)
	if len(f.errs) != 0 {
		t.Fatalf("deviation case at exact area errored: %v", f.errs)
	}
	if len(f.logs) == 0 {
		t.Fatal("deviation receipt was not logged")
	}

	fo := &fakeT{} // still at OCCT's flawed area: FAIL — the gate asserts OUR exact area, not OCCT's
	assertCaseArea(fo, "simple/C8", 9640.68, dev)
	if len(fo.errs) != 1 {
		t.Fatalf("deviation gate accepted OCCT's flawed area: %v", fo.errs)
	}

	mut := dev // mutation: nudge the exact area >1% off the built area => FAIL (proves it is not a skip)
	nudged := *dev.Deviation
	nudged.ExactArea = 9781.45 * 1.02
	mut.Deviation = &nudged
	fm := &fakeT{}
	assertCaseArea(fm, "simple/C8", 9781.45, mut)
	if len(fm.errs) != 1 {
		t.Fatalf("deviation gate did not fail on a >1%% exact-area mutation: %v", fm.errs)
	}
}

// TestAssertCaseAreaPlain confirms a case WITHOUT a deviation still asserts OCCT's ExpectedArea
// (every non-C8/D1 case is unchanged) and logs no deviation receipt.
func TestAssertCaseAreaPlain(t *testing.T) {
	plain := Record{Grid: "simple", Case: "A1", ExpectedArea: 100, Deps: 0.01}

	fp := &fakeT{}
	assertCaseArea(fp, "simple/A1", 100, plain)
	if len(fp.errs) != 0 || len(fp.logs) != 0 {
		t.Fatalf("plain case at OCCT area warned/errored: %v %v", fp.errs, fp.logs)
	}

	fp2 := &fakeT{}
	assertCaseArea(fp2, "simple/A1", 102, plain) // 2% off OCCT => FAIL
	if len(fp2.errs) != 1 {
		t.Fatalf("plain case did not fail >1%% off OCCT: %v", fp2.errs)
	}
}
