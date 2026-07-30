// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"sort"
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
// `edgeCatalog.resolveCurveOffer` now settles every disagreement by a rule and records what it did
// on the body it produced (topo.Body.BuildDiagnostics); this gate reads the SHIPPED body's report:
//
//   - CodeAssembleCurveConflict — two consumers offering geometrically DIFFERENT curves — is a HARD
//     ZERO. There is no rule by which the assembler could pick, so any instance means geometry is
//     being chosen by build order. M8's was the corpus's only one and is fixed at source
//     (rebuildArcSeg's major-span arm).
//   - CodeAssembleCurveNilOffer — one consumer offering a curve, the other nil — is a RATCHET at the
//     exact measured population below. The curve is ADOPTED (reversed onto the edge's own stored
//     sense when the offering consumer walks it backwards, which is all 272 of them); the entry
//     records that a consumer still declines to carry a boundary it shares with a curved neighbour,
//     and the table shrinks as those consumers are fixed. It may never grow.
//
// ★ COVERAGE, and the blind spot, asserted rather than assumed. An EMPTY build report has two
// meanings — "the catalog reported nothing" and "this body never went through the catalog" — and
// nothing in the report itself separates them, so a gate reading only the absence of a code reads as
// coverage it does not have. Measured: 103 of the 125 shipped corpus bodies go through the catalog;
// the other 22 do NOT, because fillet.go's three early returns (FilletCylinderRim,
// FilletCylinderArc, filletTangentStripe) assemble on their own topo.Builder and never touch it.
// Each covered body is therefore required to carry CodeAssembleEdgeCatalog, and the blind set is
// pinned exactly (catalogBlindCases) so it can neither grow silently nor shrink unrecorded. The same
// assertion is what would catch a body-rebuilding step appearing between the fillet and this gate:
// BuildDiagnostics does not survive a rebuild (topo.BodyFromShells / MergeBodies drop it), so one
// ops.Boolean in between would move every case into the blind set and fail here rather than quietly
// zeroing the census.
//
// The population is measured, not quoted from any report: an instrumented sweep of every scored
// case's shipped body at HEAD. Note it is the SHIPPED count — the assembler builds up to four
// do-no-harm candidate bodies per case (fillet.go rebuildCandidates) and the discarded ones carry
// their own, larger reports (698 over all passes and 13 cases, vs 272 over 10 shipped bodies).
func TestNoShippedBuildResolvesACurveDisagreementByBuildOrder(t *testing.T) {
	dir := CorpusFixtureDir()
	got, blind := map[string]int{}, []string{}
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no shipped body to read a build report from
		}
		key := r.Grid + "/" + r.Case
		if countBuildDiagnostics(body, ops.CodeAssembleEdgeCatalog) == 0 {
			blind = append(blind, key)
			continue
		}
		assertNoCurveConflict(t, key, body)
		got[key] = countBuildDiagnostics(body, ops.CodeAssembleCurveNilOffer)
	}
	assertNilOfferPopulation(t, nilCurveOfferDebtIndex(), got)
	assertCatalogBlindSpot(t, blind)
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
			t.Errorf("%s: nil-vs-curve offers on the shipped body grew %d → %d — a consumer newly declines to "+
				"carry a boundary it shares with a curved neighbour", key, want[key], n)
		}
	}
	for key, n := range want {
		if got[key] < n {
			t.Errorf("%s: nil-vs-curve offers fell %d → %d — record the improvement in knownNilCurveOfferDebt "+
				"(the ratchet only shrinks when the table shrinks with it)", key, n, got[key])
		}
	}
}

// assertCatalogBlindSpot requires the set of shipped bodies that did NOT go through the edge catalog
// to be exactly the recorded one, so the gate's real coverage is stated rather than assumed.
func assertCatalogBlindSpot(t *testing.T, got []string) {
	t.Helper()
	sort.Strings(got)
	want := catalogBlindCases()
	if len(got) == len(want) && equalStrings(got, want) {
		return
	}
	t.Errorf("shipped bodies not assembled through the edge catalog: %d %v, want %d %v — a case that LEFT the "+
		"catalog is outside every assertion in this file (and a case that joined it is an improvement to record)",
		len(got), got, len(want), want)
}

// equalStrings reports whether two sorted string slices of equal length hold the same values.
func equalStrings(a, b []string) bool {
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// catalogBlindCases are the shipped corpus bodies the edge catalog never sees, measured by sweeping
// every scored case for CodeAssembleEdgeCatalog: fillet.go's three early returns build them on their
// own topo.NewBuilder (fillet_rim_build.go, fillet_arc_build.go, fillet_stripe_build.go) — a lone
// circular rim pick, a lone arc pick, and a tangent stripe. 22 of 125 shipped bodies, 17.6 %. Every
// assertion in this file is silent about them BY CONSTRUCTION, which is why the set is pinned here
// instead of being left to an empty build report that reads like coverage.
func catalogBlindCases() []string {
	return []string{
		"bfuseblend/A1", "bfuseblend/A2", "bfuseblend/A3", "bfuseblend/A7", "bfuseblend/B1",
		"simple/B2", "simple/H6", "simple/I9", "simple/J1", "simple/J2", "simple/J4", "simple/J6",
		"simple/J8", "simple/K1", "simple/N6", "simple/R8", "simple/U6", "simple/W2", "simple/W6",
		"simple/W8", "simple/W9", "simple/Z1",
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
// edge where one consumer offered a curve and the other offered nil — 272 edges over 10 cases,
// measured by an instrumented sweep of every scored case at HEAD, not quoted from any report.
//
// 362 → 272 in one step, and the 90 that left were never debt: they were offers departing their own
// chord by 4.4e-16…2.0e-15 (exactly 25 % of every case: 12 of 48, 6 of 24), three orders below every
// model weld here (5.1e-08…1.3e-07). The nil branch applied no threshold at all while the conflict
// branch used the model weld; it now uses the same one, which is a correctness fix to the detector's
// own contract, not a laundered shrink — TestOfferWithinTheWeldIsNeitherAdoptedNorRecorded
// (kernel/ops) drives an offer a hair each side of the weld and requires the low one silent and the
// high one still recorded AND adopted.
//
// What remains is real: the offered curves are the blend patch's own boundary rails (r=8 / r=25 rim
// arcs on T3, 0.068 / 0.085 off the chord). The catalog now ADOPTS every one of them — measured
// improvement, not an override: T3's blend torus 2827.227365 → 2826.791716 and its two mirror-twin
// blend cylinders 227.809448 / 227.851401 → 227.764913 / 227.764925, off-surface residual
// 7.220e-04 → 3.973e-04, rollup unchanged at 114 simple / 119 all-grid. The entry that remains is
// the CONSUMER still declining to carry the boundary; retire each at that consumer.
func knownNilCurveOfferDebt() []nilCurveOfferDebt {
	return []nilCurveOfferDebt{
		{"complex", "F2", 2},
		{"simple", "S1", 36},
		{"simple", "S4", 36},
		{"simple", "S6", 18},
		{"simple", "S7", 36},
		{"simple", "S9", 18},
		{"simple", "T1", 36},
		{"simple", "T3", 18},
		{"simple", "T4", 36},
		{"simple", "T7", 36},
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
