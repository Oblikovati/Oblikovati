// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"testing"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// nilCurveOfferDebt is one case whose SHIPPED body was assembled with n welded edges where one
// consumer offered a curve and the other offered nil.
type nilCurveOfferDebt struct {
	grid, name string
	n          int
}

// TestNoShippedBuildResolvesACurveDisagreementByBuildOrder is the corpus-wide standing gate on the
// fillet edge catalog's arbitration.
//
// The catalog welds ONE shared topo edge per (vertex pair, identity class) and both faces meeting on
// it offer a curve. Historically the first offer won and every later one was discarded SILENTLY, so
// which geometry shipped was decided by the order the assembler happened to visit faces in. That is
// not a tolerance question — on `simple/M8` the two offers for one welded edge were a 255.52° boss
// rim arc and its own 104.48° COMPLEMENT, 22.01 apart, and M8 was GREEN and pinned throughout.
// `edgeCatalog.recordCurveOffer` now records every disagreement on the body it produced
// (topo.Body.BuildDiagnostics), and this gate reads the SHIPPED body's own report:
//
//   - CodeAssembleCurveConflict — two consumers offering geometrically DIFFERENT curves — is a HARD
//     ZERO. There is no rule by which the assembler could pick, so any instance means geometry is
//     being chosen by build order. M8's was the corpus's only one and is fixed at source
//     (rebuildArcSeg's major-span arm).
//   - CodeAssembleCurveNilOffer — one consumer offering a curve, the other nil — is a RATCHET at the
//     exact measured population below. A nil is an absence of information, not an assertion of
//     straightness, so each of these is a consumer that should carry a curve and does not. Adopting
//     the curve assembler-side was measured and refuted (it moves the region the nil-offering face
//     bounds: simple/T3's blend torus goes 2827.227365 → 13816.882599 mesh area, and S9/T1/T3/T4
//     fall out of OCCT's 1 % gate) — so the debt is retired at the CONSUMER, and this table shrinks
//     as that happens. It may never grow.
//
// The population is measured, not quoted from any report: an instrumented sweep of every scored
// case's shipped body at HEAD. Note it is the SHIPPED count — the assembler builds up to four
// do-no-harm candidate bodies per case (fillet.go rebuildCandidates) and the discarded ones carry
// their own, larger reports (698 over all passes and 14 cases, vs 362 over 10 shipped bodies).
func TestNoShippedBuildResolvesACurveDisagreementByBuildOrder(t *testing.T) {
	dir := CorpusFixtureDir()
	want := nilCurveOfferDebtIndex()
	got := map[string]int{}
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no shipped body to read a build report from
		}
		key := r.Grid + "/" + r.Case
		assertNoCurveConflict(t, key, body)
		got[key] = countBuildDiagnostics(body, ops.CodeAssembleCurveNilOffer)
	}
	assertNilOfferPopulation(t, want, got)
}

// assertNoCurveConflict fails on any welded edge whose two consumers offered geometrically different
// curves — the class whose winner is decided by build order alone. The diagnostic Detail carries the
// offending pair, its endpoints, both curves and the deviation, so a failure is directly actionable.
func assertNoCurveConflict(t *testing.T, key string, body *topo.Body) {
	t.Helper()
	for _, d := range body.BuildDiagnostics() {
		if d.Code == ops.CodeAssembleCurveConflict {
			t.Errorf("%s: the shipped body resolved a curve disagreement by build order — %s", key, d.Detail)
		}
	}
}

// assertNilOfferPopulation compares the measured nil-vs-curve debt against the recorded population,
// failing in BOTH directions: growth is a regression (a consumer newly drops a curve on a shared
// edge), and a shrink must be recorded so the ratchet keeps its teeth.
func assertNilOfferPopulation(t *testing.T, want, got map[string]int) {
	t.Helper()
	for key, n := range got {
		if n > want[key] {
			t.Errorf("%s: nil-vs-curve offers on the shipped body grew %d → %d — a consumer newly drops a "+
				"curve on a welded edge and the straight chord it leaves is decided by build order", key, want[key], n)
		}
	}
	for key, n := range want {
		if got[key] < n {
			t.Errorf("%s: nil-vs-curve offers fell %d → %d — record the improvement in knownNilCurveOfferDebt "+
				"(the ratchet only shrinks when the table shrinks with it)", key, n, got[key])
		}
	}
}

// countBuildDiagnostics counts the body's build-report entries carrying code.
func countBuildDiagnostics(body *topo.Body, code diag.Code) int {
	n := 0
	for _, d := range body.BuildDiagnostics() {
		if d.Code == code {
			n++
		}
	}
	return n
}

// nilCurveOfferDebtIndex keys the recorded population by "grid/case".
func nilCurveOfferDebtIndex() map[string]int {
	out := make(map[string]int, len(knownNilCurveOfferDebt()))
	for _, d := range knownNilCurveOfferDebt() {
		out[d.grid+"/"+d.name] = d.n
	}
	return out
}

// knownNilCurveOfferDebt is the FULL population of shipped bodies assembled with at least one welded
// edge where one consumer offered a curve and the other offered nil — 362 edges over 10 cases,
// measured by an instrumented sweep of every scored case at HEAD, not quoted from any report.
//
// Every entry is a real consumer-side gap, not noise: the offered curves are the blend patch's own
// boundary rails (r=8 / r=25 rim arcs on T3, ~0.068 / 0.085 off the chord they lose to) and the
// straight chord that ships in their place is what the off-surface residual of these cases measures.
// Adopting the curve in the catalog takes that residual down 2.8–4.6× on every case here and is
// still wrong (see the test docstring): retire each entry at the consumer that drops the curve.
func knownNilCurveOfferDebt() []nilCurveOfferDebt {
	return []nilCurveOfferDebt{
		{"complex", "F2", 2},
		{"simple", "S1", 48},
		{"simple", "S4", 48},
		{"simple", "S6", 24},
		{"simple", "S7", 48},
		{"simple", "S9", 24},
		{"simple", "T1", 48},
		{"simple", "T3", 24},
		{"simple", "T4", 48},
		{"simple", "T7", 48},
	}
}

// TestNilCurveOfferDebtIsWellFormed keeps the population table honest: no duplicate key and no
// zero-count entry (which would record no debt at all while still exempting a case from growth).
func TestNilCurveOfferDebtIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range knownNilCurveOfferDebt() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate nil-curve-offer debt entry %s", key)
		}
		seen[key] = true
		if d.n <= 0 {
			t.Errorf("%s debt count %d is not positive — delete the entry instead", key, d.n)
		}
	}
}
