// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import stdmath "math"

// testingT is the slice of *testing.T the assertion needs, kept as an interface so it is
// unit-testable with a fake recorder. The `...any` signature is the one justified use of
// `any` in this package: it must structurally match stdlib testing.TB's Errorf/Logf so a
// real *testing.T satisfies it without adapting.
type testingT interface {
	Errorf(format string, args ...any)
	Logf(format string, args ...any)
}

// driftWarnRel is a sharper, non-failing internal signal (0.1%) inside OCCT's own 1%
// (deps) failing gate — lets us watch parity tightening over time without moving the gate.
const driftWarnRel = 1e-3

// assertArea mirrors OCCT checkprops: fail when relative area error exceeds deps (OCCT's own
// tolerance, usually 1%), warn (non-failing) when it merely exceeds driftWarnRel (0.1%).
// expected==0 is handled by the caller (OCCT-TODO -s 0 cases are skipped upstream, never
// reach here).
func assertArea(t testingT, name string, got, expected, deps float64) {
	rel := stdmath.Abs(got-expected) / stdmath.Abs(expected)
	if !areaWithin(got, expected, deps) {
		t.Errorf("%s: area %.6g != OCCT %.6g (rel %.4f%% > %.2f%%)", name, got, expected, rel*100, deps*100)
		return
	}
	if rel > driftWarnRel {
		t.Logf("%s: area drift %.4f%% from OCCT %.6g (within %.1f%% gate)", name, rel*100, expected, deps*100)
	}
}

// areaWithin reports whether got is within OCCT's relative tolerance deps of expected — the
// gate condition, shared by assertArea (gating) and the scoreboard (non-gating tally).
func areaWithin(got, expected, deps float64) bool {
	return stdmath.Abs(got-expected)/stdmath.Abs(expected) <= deps
}

// assertCaseArea asserts the built area against OCCT's reference for a genuine-parity case, or —
// when the case carries a documented per-case deviation (OCCT's own result is geometrically
// flawed; occt-oracle-not-religion) — against OUR known-correct exact area, first logging the
// forensic receipt (the OCCT area deviated from, the signed deviation, and the Reason). BOTH
// branches gate at r.Deps, so a future regression of OUR geometry >Deps off the exact area still
// FAILS — unlike a todo-skip, which would stop gating the case entirely.
//
// THE STANDARD for adding one (this comment is the deviation mechanism's human-readable audit trail —
// keep it in step with corpus.json): a Deviation is always PER-CASE and RECEIPTED — the global deps is
// never widened and no tolerance is ever loosened to absorb one. It is admissible only when OCCT's own
// recorded number is demonstrably wrong (a geometry OCCT builds off-tangency, or a number its own
// integrator mis-computes), the correct value is derived independently (closed-form/direct integration),
// and the Reason string carries the forensic receipt. FOUR cases carry one today, all in `simple`:
//   - C8 — OCCT's corner is a non-tangent 3-stripe BSpline sag;
//   - D1 — OCCT's snout ruling arm is 3.5% under the true rolling-ball envelope;
//   - J6, J8 — OCCT's `sprops <shape> 1e-4` MIS-INTEGRATES the oblique-prism/pipe wall (a
//     SurfaceOfLinearExtrusion), so its recorded area is inflated while its GEOMETRY agrees with ours.
//
// Every other case asserts OCCT's ExpectedArea unchanged.
func assertCaseArea(t testingT, name string, area float64, r Record) {
	if d := r.Deviation; d != nil {
		t.Logf("%s: per-case exact-deviation — asserting OUR exact area %.6g (%+.2f%% from OCCT %.6g): %s",
			name, d.ExactArea, (d.ExactArea-d.OCCTArea)/d.OCCTArea*100, d.OCCTArea, d.Reason)
	}
	assertArea(t, name, area, r.areaTarget(), r.Deps)
}
