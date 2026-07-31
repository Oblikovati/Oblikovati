// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"slices"
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
//     sense when the offering consumer walks it backwards); the entry records that a consumer still
//     declines to carry a boundary it shares with a curved neighbour, and the table shrinks as those
//     consumers are fixed. It may never grow. The remaining population is complex/F2's curve-FIRST
//     pair alone (2 kept, 0 adopted, 0 declined): the wing arm-arc carry (sampledArcSegs,
//     kernel/ops/fillet_runout_faces.go) retired the last adoption family — the nine setback cases'
//     wing faces now offer the same trimmed arm-arc sub-spans their patches do.
//
// ★ COVERAGE, and the blind spot, asserted rather than assumed. An EMPTY build report has two
// meanings — "the catalog reported nothing" and "this body never went through the catalog" — and
// nothing in the report itself separates them, so a gate reading only the absence of a code reads as
// coverage it does not have. Measured: 104 of the 127 shipped corpus bodies go through the catalog;
// the other 23 do NOT, because fillet.go's three early returns (FilletCylinderRim,
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
// their own, larger reports (698 over all passes and 13 cases at the adoption HEAD; the footprint-rim
// curve carry shrank the shipped count 272 → 110 over the same 10 bodies, and the wing arm-arc carry
// 110 → 2 — complex/F2 alone).
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
	if slices.Equal(got, want) {
		return
	}
	t.Errorf("shipped bodies not assembled through the edge catalog: %d %v, want %d %v — a case that LEFT the "+
		"catalog is outside every assertion in this file (and a case that joined it is an improvement to record)",
		len(got), got, len(want), want)
}

// catalogBlindCases are the shipped corpus bodies the edge catalog never sees, measured by sweeping
// every scored case for CodeAssembleEdgeCatalog: fillet.go's three early returns build them on their
// own topo.NewBuilder (fillet_rim_build.go, fillet_arc_build.go, fillet_stripe_build.go) — a lone
// circular rim pick, a lone arc pick, and a tangent stripe. 23 of 127 shipped bodies, 18.1 %. Every
// assertion in this file is silent about them BY CONSTRUCTION, which is why the set is pinned here
// instead of being left to an empty build report that reads like coverage.
//
// bfuseblend/B3 JOINED this set when it greened (W-DH, independent-closed-rims weld): its two
// pairwise-disjoint Cylinder∧Plane rims are each rebuilt sequentially through FilletCylinderRim, the
// same A1/W6/W8 cove-class early return — never through the edge catalog, so B3's welded edges carry
// no CodeAssembleEdgeCatalog diagnostic to read a curve-offer report from. simple/M5 (W-DH's other
// new green) is NOT blind: its corner weld goes through the general assembler and was checked clean
// by this file's other assertions (no CodeAssembleCurveConflict, no CodeAssembleCurveNilOffer growth).
//
// bfuseblend/B4 B5 and simple/K2 K3 K4 JOINED for the same architectural reason (W-B, Cylinder∧Cylinder
// closed-loop SSI-seam canal, fillet_cylcyl_seam*.go): the canal welds through the SAME rebuildRim
// (fillet_rim_build.go) the analytic torus/elliptic rim families already use — a new CALLER of the
// existing pinned bypass, not a new bypass (measured: the K2/K3/K4 seam ends route through
// cylinderArmEdge → rebuildWithRimFillet / rebuildWithConcaveRimFillet, verified against rebuildRim's
// own topo.NewBuilder call).
func catalogBlindCases() []string {
	return []string{
		"bfuseblend/A1", "bfuseblend/A2", "bfuseblend/A3", "bfuseblend/A7", "bfuseblend/B1",
		"bfuseblend/B3", "bfuseblend/B4", "bfuseblend/B5",
		"simple/B2", "simple/H6", "simple/I9", "simple/J1", "simple/J2", "simple/J4", "simple/J6",
		"simple/J8", "simple/K1", "simple/K2", "simple/K3", "simple/K4", "simple/N6", "simple/R8",
		"simple/U6", "simple/W2", "simple/W6", "simple/W8", "simple/W9", "simple/Z1",
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
// edge where one consumer offered a curve and the other offered nil — 2 edges over 1 case,
// measured by an instrumented sweep of every scored case at HEAD, not quoted from any report.
//
// 110 → 2 by the wing arm-arc carry (wing-arm-arcs-report.md): the nine setback cases each carried
// 12 records, one family — the fillet's own r=R arm cross-section arcs (geom.Arc3d sweep π/12 per
// sub-edge, the quarter arc at ringSegSamples), where the WING face's cut section (sampledArcSegs,
// kernel/ops/fillet_runout_faces.go) tiled plain chords while the setback patch offered its exact
// trimmed sub-spans (runoutPatchLoops → sampleCurve3OpenTrimmed, re-derived as Arc3d by
// reverseSegmentCurve when the shell orientation flips the patch loop). The wing now carries the
// SAME sampleCurveNTrimmed sub-spans, so every one of those edges is a two-sided VALUE AGREEMENT —
// silent by the weld rule — and the shipped meshes are identical at the pins' 1e-6·boundingDiag
// quantization (all nine fingerprint pins unchanged). The RAW mesh does drift ≤7.6e-12 (T3: 521 of
// 81,844 triangles) because the mesher resamples the stored edge curve (edge_discretize.go's
// sampleEdgeCurve → e.Geometry().PointAt) and the replacement curves agree only to ≲1e-13 — four
// decades below the pins' quantum, which is WHY the pins hold. The adoption had already repaired
// the edge; the debt was the consumer's loop model.
// What remains is complex/F2's curve-FIRST pair (EllipticalArc, departure 2.92426 against a
// 1.76388e-07 weld), retained by prior adjudication (finding m-1).
//
// 272 → 110 by the footprint-rim curve carry (t3-plane-sliver-report.md): the intact boss wall's
// subdivided rim segments now OFFER their own exact sub-spans of the footprint conic
// (subdivideBossWall → maps.insertCurves), so every band-side rim edge the setback patches used to
// repair by adoption is now a two-sided VALUE AGREEMENT and never presents as a nil offer at all.
//
// 362 → 272 in one step, and the 90 that left were never debt: they were offers departing their own
// chord by 4.4e-16…2.0e-15 (exactly 25 % of every case: 12 of 48, 6 of 24), three orders below every
// model weld here (5.1e-08…1.3e-07). The nil branch applied no threshold at all while the conflict
// branch used the model weld; it now uses the same one, which is a correctness fix to the detector's
// own contract, not a laundered shrink — TestOfferWithinTheWeldIsNeitherAdoptedNorRecorded
// (kernel/ops) drives an offer a hair each side of the weld and requires the low one silent and the
// high one still recorded AND adopted. The CURVE-FIRST arm was completed in the adversarial-review
// fix wave (finding m-1: resolveCurveOffer's offered==nil branch recorded with no threshold while
// the file docstring claimed the rule); the population is unchanged by it — the only curve-first
// records are complex/F2's two, departing 2.92426 against a 1.76388e-07 weld, seven orders above —
// and TestCurveFirstKeepUsesTheSameWeldThreshold (kernel/ops) gates that arm both ways too.
//
// (Historical, the adoption slice: the offered curves were the blend patch's own boundary rails —
// r=8 / r=25 rim arcs on T3, 0.068 / 0.085 off the chord — and the catalog ADOPTED every one:
// T3's blend torus 2827.227365 → 2826.791716 and its two mirror-twin blend cylinders
// 227.809448 / 227.851401 → 227.764913 / 227.764925, off-surface residual 7.220e-04 → 3.973e-04,
// rollup unchanged at 114 simple / 119 all-grid. Each family since was then retired at its
// declining CONSUMER; F2's pair is retained, not declining — the curve won at first writing.)
func knownNilCurveOfferDebt() []nilCurveOfferDebt {
	return []nilCurveOfferDebt{
		{"complex", "F2", 2},
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
