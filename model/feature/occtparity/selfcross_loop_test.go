// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// TestNoFaceLoopSelfCrossesOnItsSurface is the corpus-wide RATCHET on a defect class that until now
// nothing in the kernel noticed: a face whose boundary, developed onto its OWN surface, crosses itself.
//
// WHY THIS IS AN INVARIANT. A trimmed face IS the region its loops bound in its surface's chart. When
// that polygon self-crosses the region is undefined — no area, no inside, no correct triangulation —
// so the mesher, mass properties, export and the next boolean are all answering a question that has no
// answer. The face still validates as a watertight solid (the vertices weld, the loops close), which
// is exactly why the scoreboard cannot see it.
//
// WHAT IT HAS ALREADY BOUGHT. The population it opened at — 17 loops on 11 cases — is down to 3 on 2: the
// backwards-carried-curve root it exposed (simple/M4 N3 N9 and two of complex/F2's walls) took those four
// cases' faces onto DRAWEXE's own per-face areas exactly, a defect no area gate could see because M4's
// body was inside 0.2% and F2's two errors partly cancelled.
//
// WHAT IT COSTS, MEASURED. The pinched-off area is often TINY and the damage is not: complex/D8's two
// MIRROR-IMAGE corner rounds pinch off the IDENTICAL 1.21187 out of a closed-form 3307.1168 (0.037%),
// and the constrained Delaunay handed those two domains answers −0.048% and −38.941% purely because
// the crossing lands at a different index in the loop (cdt_coverage.go's 7-of-10 population is exactly
// this class). A defect that small must not be allowed to spread, hence a ratchet rather than a budget.
//
// It is a RATCHET, not a tolerance: every case that still carries one is listed with its MEASURED
// count and pinched-off area, so a listed case may improve freely while a new one — or a listed one
// growing — fails loud. The table must shrink and must never be widened to accommodate a regression.
func TestNoFaceLoopSelfCrossesOnItsSurface(t *testing.T) {
	dir := CorpusFixtureDir()
	q := ops.PropertyQuality()
	debt := selfCrossDebtIndex()
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no single healthy body to measure
		}
		assertSelfCrossingWithinDebt(t, r, ops.SelfCrossingFaceLoops(body, q), debt[r.Grid+"/"+r.Case])
	}
}

// assertSelfCrossingWithinDebt fails when a case carries more self-crossing loops than its recorded
// debt, or one that pinches off more area than recorded. A case with no entry may carry none at all.
func assertSelfCrossingWithinDebt(t *testing.T, r Record, bad []ops.SelfCrossingLoop, debt selfCrossDebtEntry) {
	t.Helper()
	if len(bad) > debt.loops {
		t.Errorf("%s/%s: %d face loop(s) self-cross on their own surface, recorded debt %d — %s",
			r.Grid, r.Case, len(bad), debt.loops, describeSelfCrossing(bad))
	}
	for _, b := range bad {
		if b.Area > debt.area {
			t.Errorf("%s/%s: face %d loop %d self-crosses, pinching off %.6g of its own surface (ceiling %.6g)",
				r.Grid, r.Case, b.Face.ID(), b.Loop, b.Area, debt.area)
		}
	}
}

// describeSelfCrossing names each offending face, its surface, the area it pinches off and how
// faithfully its own crossing pair developed — because an unfaithful pair's "area" is a chart quantity,
// not an area on the surface, and reading one as the other is the category error §8.5 warned about.
func describeSelfCrossing(bad []ops.SelfCrossingLoop) string {
	out := ""
	for _, b := range bad {
		out += fmt.Sprintf(" [%T face %d loop %d pinches %.6g chart/chord %.4g]",
			b.Face.Geometry(), b.Face.ID(), b.Loop, b.Area, b.ChartChordRatio)
	}
	return out
}

// selfCrossOffSurfaceFloor is the residual, relative to the body's bounding diagonal, above which a
// boundary has demonstrably left the face it bounds. It is loopSegOnFaceTol — the SAME ruler
// TestEveryLoopSegmentLiesOnItsFace gates on, so the two guards cannot disagree about what "off its own
// surface" means.
const selfCrossOffSurfaceFloor = loopSegOnFaceTol

// TestEverySelfCrossDebtEntryIsARealDefect is the ratchet's ANTI-LAUNDERING guard, and the reason the
// entries below can be trusted to be geometry rather than bookkeeping.
//
// WHY IT EXISTS. Two of this table's former entries — simple/W1 and simple/E4 — turned out to be
// artefacts of the DEVELOPMENT, not defects of the boundary: their loops started on a sphere's (u,v)
// seam and wound a whole period about it, so the chart they were measured in did not represent the
// surface at all. They were retired only after being PROVEN artefacts (their faces mesh to an exact
// spherical-excess closed form), and by a production fix to unwrap rather than by an edit to the
// detector. Nothing, however, stopped the next such entry from being quietly dropped on suspicion —
// and "we do not launder ratchet shrinkages by quietly altering the detector" is the standing rule.
//
// WHAT IT ASSERTS. Every recorded crossing must be convictable on its own evidence, one of two ways:
// its crossing pair developed FAITHFULLY (ChartChordRatio inside the half-turn cut), so the chart
// measured the surface and the crossing is real on it; OR the pair developed unfaithfully AND the
// body's boundary is measurably OFF the face it bounds, so the unfaithfulness is caused by geometry
// that is itself wrong. An entry that is neither is a chart artefact and must be retired with its
// receipt — not left to inflate the table, and not dropped silently.
//
// ★ IT IS ALSO THE PROOF THAT THE OBVIOUS "CORRECTION" WOULD HAVE BEEN WRONG. complex/F2's two
// crossings measure 2.771 and 77.06 — the second is the 77× that stood recorded as a suspected chart
// artefact. It is not one: the crossing pair's own boundary points lie 9.125 and 9.818 off the
// radius-10 cylinder they bound, and the body's worst boundary-off-its-own-face residual is 9.87026 —
// 0.05596 of its 176.4 diagonal, under knownOffSurfaceDebt's complex/F2 ceiling of 0.0616 — on FACE 243
// itself, the very face carrying the 1098.03 crossing. The same defect, two rulers. Filtering the
// detector on the ratio, the mirror of the retrace detector's corroboratedIn3D, would have retired both
// as artefacts: measured, it drops complex/F2 from 2 reported loops to 0 while every gate in this
// harness stays green.
func TestEverySelfCrossDebtEntryIsARealDefect(t *testing.T) {
	dir := CorpusFixtureDir()
	q := ops.PropertyQuality()
	for _, d := range knownSelfCrossingLoops() {
		r := corpusRecord(t, d.grid, d.name)
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			t.Errorf("%s/%s is in knownSelfCrossingLoops but ships no healthy body to convict it on", d.grid, d.name)
			continue
		}
		assertSelfCrossingsAreConvicted(t, r, body, ops.SelfCrossingFaceLoops(body, q))
	}
}

// assertSelfCrossingsAreConvicted fails for any reported crossing that is neither faithfully developed
// nor accompanied by a boundary that has left its own face.
func assertSelfCrossingsAreConvicted(t *testing.T, r Record, body *topo.Body, bad []ops.SelfCrossingLoop) {
	t.Helper()
	worst, where := worstLoopSegmentOffFace(body)
	offSurface := worst/boundingDiag(body) > selfCrossOffSurfaceFloor
	for _, b := range bad {
		if b.ChartFaithful() || offSurface {
			continue
		}
		t.Errorf("%s/%s: face %d loop %d crosses at chart/chord %.4g — its development does not represent "+
			"the surface — yet the body's worst boundary-off-its-own-face residual is only %.6g (rel %.4g "+
			"of the %.4g diagonal, floor %.4g) at %s. That is a CHART ARTEFACT, not a defect: retire the "+
			"entry with its receipt rather than carrying it",
			r.Grid, r.Case, b.Face.ID(), b.Loop, b.ChartChordRatio, worst, worst/boundingDiag(body),
			boundingDiag(body), selfCrossOffSurfaceFloor, where)
	}
}

// selfCrossDebtEntry is one case's measured self-crossing debt: how many of its loops cross, and the
// largest area any of them pinches off.
type selfCrossDebtEntry struct {
	name, grid string
	loops      int
	area       float64
}

// selfCrossDebtIndex keys knownSelfCrossingLoops by "grid/case" for O(1) lookup.
func selfCrossDebtIndex() map[string]selfCrossDebtEntry {
	out := make(map[string]selfCrossDebtEntry, len(knownSelfCrossingLoops()))
	for _, d := range knownSelfCrossingLoops() {
		out[d.grid+"/"+d.name] = d
	}
	return out
}

// knownSelfCrossingLoops is the FULL measured population of shipped faces whose developed boundary
// self-crosses: 5 loops on 3 cases, of 1155 faces across the scored corpus (it was 17 on 11 of 1144,
// then 12 on 8, then 9 on 6, then 7 on 4).
// Each ceiling is the measured pinched-off area, in the surface's own metric chart, rounded UP a little so
// float noise cannot fail it while a real growth does. Derived by an instrumented corpus-wide sweep at
// ops.PropertyQuality(), not from any report.
//
// RETIRED: simple/M4, simple/N3, simple/N9 (whole cases) and 2 of complex/F2's 4 loops — all five were one
// root, a loop segment whose carried CURVE ran backwards between its own two points, which discretizeEdge
// then turned into a doubled-back polyline (planar-retrim-selfcross-report.md; the fix is in
// matchArcFeet + orientedOpenSurvivor, and it also emptied knownEdgeSpanDebt). Their faces now rank-pair
// EXACTLY against DRAWEXE: M4's cap 9752.18 → 9631.8 (oracle 9631.79), N3's 10955.6 → 10946.3 (10946.3),
// N9's 1307.31 → 1298.6 (1298.57), and complex/F2's largest wall 6517.86 → 10366.6 (10366.6, exact).
//
// RETIRED, second wave: simple/Y2 (2 loops, 50 / 1.41397) and simple/Y4 (1 loop, 23.9109) — the
// SETBACK-BAND-OVERRUN root below, closed by the band∩obstacle imprint walk
// (kernel/ops/fillet_band_imprint.go). Their faces now match their own CLOSED FORMS rather than merely
// stopping crossing: Y2's host plane 8475 → 8450, its band 2356.18 → 2305.2190 and the slot's three
// walls 1000 / 1000 / 964.027 → 998.5870 / 991.4214 / 953.1276; Y4's host 7600 → 7500 and its six
// neighbours likewise (model/feature/occtparity/slot_band_imprint_test.go). Note what the ratchet could
// NOT see and the closed forms could: Y4's LARGEST single error was +100 on the host plane, whose loop
// back-tracked along a COLLINEAR sibling instead of crossing it, which simpleLoop2D scores as zero
// crossings — the listed 23.9109 was a different face (the wall above the slot).
//
// RETIRED, third wave — as FALSE POSITIVES, not as fixed geometry: simple/E4 (1 loop, 128.489) and
// simple/W1 (1 loop, 0). Both were the SPHERE-PATCH SEAM entry below, and both were measuring the CHART
// rather than the boundary. Their loops start ON the sphere's (u,v) seam and wind a whole period about
// it — the patch's own vertex is the chart's pole — so they have no development in that chart at all,
// and unwrap's open-chain-only guard passed them by as little as 0.024 rad before leaping 2π on the
// closing step (retrace-detector-report.md §7.1). The guard now covers the closing step
// (kernel/ops/tessellate_trim.go), so developedFaceLoops SKIPS them, which is what it already promises
// to do for a loop that wraps the seam. ★ THE SHRINK IS CAUSED BY A PRODUCTION FIX, NOT BY A DETECTOR
// EDIT — loopSelfCrossing and developedFaceLoops are untouched — and it is NOT a geometry improvement:
// those two faces' meshes are byte-identical before and after (they route through spherePatchMesh, not
// the seam chart). What convicts the old entries is the rank-pair: E4's face meshes 171.8998 against
// its exact spherical-excess closed form 171.9270, −0.0159 %, which a face genuinely pinching off
// 128.489 of its own 171.93 could not do. Both are now gated on that closed form instead, by
// TestSeamWindingSpherePatchMeshesToClosedForm.
//
// The roots that remain, largest first:
//
//   - FAR-END TRIM RUNS OFF ITS STOP FACE (complex/F2; complex/D8 and simple/Q5 RETIRED, below).
//     trimTerminalSection slides every station of the band's terminal section onto the stop face's
//     EXTENDED implicit surface (extendableWall says so outright) with no check that the landing is on
//     the FACE. The stop face's boundary then has to be RE-TRIMMED — the run-out consumes a whole rim
//     edge and shortens the next ruling — which transformLoop's single-vertex substitution cannot
//     express (selfcross-trim-report.md §5).
//     ★ complex/F2 is NOT blocked by the multi-face split's own gates, as was believed and recorded
//     here: an instrumented corpus-wide sweep shows F2 never reaches splitTerminalSection at all —
//     trimTerminalSection declines EARLIER, at slideSectionOntoWall, at BOTH of its corners, so its
//     section has no landing inside the band's own axial span to classify. Its box-patch stop faces are
//     never evaluated. F2 waits on that decline, not on the split's narrowing.
//
// RETIRED, fifth wave: simple/Q5 (2 loops — 84912.4 off the small radius-3000 host-wall face and
// 0.284334 off the z=6000 top plane) — the FAR-END TRIM root above, closed by GENERALISING the split's
// own gate rather than by any new geometry. It used to require BOTH of a fillet's terminal sections to
// resolve into a two-piece chain; Q5 splits at ONE end only (its high-x end stops on a plane
// perpendicular to the band's axis and needs no trim at all), so the whole split was declined and nine
// of its 33 stations were slid past the small wall face's own ruling. With splitEndCount admitting a
// one-ended split, Q5's four touched faces land on their closed forms — the small wall piece
// 7645850.16 → 8121160.60 (form 8121170.18, was −5.85 %), the big wall piece 47123886.60 → 47038364.82
// (form 47038978.00, was +0.181 %), the top plane 49441243.96 → 49368723.87 (form 49368722.08, was
// +0.147 %) — its body goes −0.0918 % → −0.000194 % of DRAWEXE's 3.46388e8, and its welded mesh stops
// leaking 937 free edges. Gated by TestQ5FarEndSplitIsAtomicAndHitsItsClosedForms. ★ The shrink is
// caused by a PRODUCTION fix — SelfCrossingFaceLoops, loopSelfCrossing and developedFaceLoops are
// untouched by that slice.
//
// RETIRED, fourth wave: complex/D8 (2 loops, 1.21187 each) — the FAR-END TRIM root above, closed for this
// configuration by the trim-side two-piece split (kernel/ops/fillet_farend_split.go) plus the atomic
// chain rebuild of every host it touches (kernel/ops/fillet_farend_chain.go). D8's r=30 band stops on a
// radius-24 quarter round spanning u ∈ [−π/2, 0] while its own section reached u = +0.2527 — 6.064 of
// developed length onto the flat wall next door; it is now CUT at the analytic triple point
// (223.39418029785, 35.093784332275, −20+√864) and each piece bounds the one face that carries it. As with
// Y2/Y4 the faces now match their own CLOSED FORMS rather than merely stopping crossing: the two mirror
// rounds 3332.57 / 3392.39 → 3307.06 / 3307.06 (closed form 3307.1168, and mirror-equal to 1.6e-5 where
// they used to differ by 60), the two flat walls 16291.54 → 16290.33 (16290.3328), the top plane
// 92175.22 → 92172.19 (92172.2083) and the band 23339.47 → 23342.65 (23343.0975). Gated on those closed
// forms by TestD8FarEndSplitIsAtomicAndHitsItsClosedForms. ★ The shrink is caused by a PRODUCTION fix —
// SelfCrossingFaceLoops, loopSelfCrossing and developedFaceLoops are untouched by that slice.
//   - (RETIRED) THE SETBACK BAND OVERRUNS THE HOST FACE'S OWN BOUNDARY (simple/Y2 ×2, simple/Y4). The retrim moves
//     the filleted edge's end VERTEX to its tangent point along the ADJACENT edge's supporting line, with
//     no check that the tangent point lies within that edge's own span. Y2's r=15 setback on a face
//     interrupted by a 10-deep slot lands 5 PAST the adjacent edge's end, so the rebuilt loop keeps the
//     whole slot and the tangent line cuts across its wall. Closed form (confirmed face-for-face by
//     DRAWEXE): Y2's host plane is 8450, the slot walls 998.587 / 991.421 / 953.128 and the band 2305.22
//     — we USED TO ship 8475 / 1000 / 1000 / 964.027 / 2356.18, a +96.8 (+0.159%) body surplus hidden
//     inside a 1% PASS. The fix CLIPS the setback band against the host's own loop, which also shortens
//     the slot's faces — i.e. the fillet is LIMITED BY the obstacle, not vertex-substituted. Done.
//   - (RETIRED, as a false positive) SPHERE-PATCH SEAM (simple/E4, simple/W1): a fillet corner sphere
//     whose loop starts on the seam and winds the period, so it has no development in that chart. See
//     the third-wave note above.
//   - simple/W2's 2.5e-13 is float noise on an otherwise simple loop.
//
// complex/F2's larger loop now measures 1098.03, not the 1104.69 its ceiling was set from. The ceiling
// is deliberately left where it stood: 1098.03 is also what the BASE commit measures, so the drop
// predates this slice, and re-setting a ceiling from a fresh measurement is the ratchet-measurement
// correction that belongs in its own slice, not a side effect of a mesher fix.
// ★ MEASUREMENT CORRECTION (this slice), NOT a geometry change and NOT a shrink. Every entry below was
// re-examined against the ONE open suspicion the detector's own report left on the record — that some of
// these numbers measure the CHART rather than the GEOMETRY — and each was convicted on its own evidence.
// Nothing left the table; no ceiling moved; SelfCrossingFaceLoops reports exactly the loops and areas it
// reported before. What changed is that each report now carries the fidelity of ITS OWN crossing pair
// (SelfCrossingLoop.ChartChordRatio), so the numbers can no longer be misread, and
// TestEverySelfCrossDebtEntryIsARealDefect makes the conviction a standing guard.
//
// The verdicts, with the measured pair fidelity (chart length ÷ 3D chord over the two segments that
// actually cross, ops.PropertyQuality()):
//
//   - simple/Q5, both loops — 1.000000 and 1.000000. The development was exact on both pairs (the
//     crossing segments are a plane's and a cylinder's axial rulings), i.e. the 84912.4 was REAL.
//     ★ RETIRED, fifth wave, by the ONE-ENDED far-end split (see below): 2 loops → 0.
//   - complex/F2, both loops — 2.771 and 77.06. ★ The 77.06 is the ratio that stood recorded as a
//     SUSPECTED chart artefact, and it is NOT one. Its crossing pair's own boundary points sit 9.125 and
//     9.818 off the radius-10 cylinder they bound, and the body's worst boundary-off-its-own-face
//     residual — 9.87026, 0.05596 of its 176.4 diagonal, under knownOffSurfaceDebt's 0.0616 ceiling —
//     is on FACE 243, the very face carrying the 1098.03 crossing. The chart is unfaithful BECAUSE the
//     geometry is wrong, so the two ratchets are reading one defect with two rulers. REAL, stays — and
//     this is the receipt that filtering the detector on the ratio would have laundered them away.
//   - simple/W2, one loop — 1.000000. A plane's development is exact by construction; the 2.5e-13 is
//     float noise thrown off by the retracing spike on the same face (knownRetracingLoops). REAL, stays.
//
// Also corrected on the record: the two entries this table's prose still lists as SUSPECTS — simple/W1's
// "Area 0" and simple/E4's 128.489 — were already RETIRED, as false positives, by the unwrap fix in the
// third wave above. The suspicion that opened this correction was, for them, already acted on.
func knownSelfCrossingLoops() []selfCrossDebtEntry {
	return []selfCrossDebtEntry{
		{"F2", "complex", 2, 1105}, // 1098.03 / 7.16978 (was 4 loops); pairs 77.06 / 2.771 — see the verdicts above
		{"W2", "simple", 1, 1e-12}, // 2.49967e-13 — a degenerate crossing, float noise; pair 1.000000
	}
}

// TestSelfCrossDebtIsWellFormed keeps the debt table honest: no duplicate key, and every entry
// claiming at least one crossing loop (a zero entry would be dead weight hiding nothing).
func TestSelfCrossDebtIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range knownSelfCrossingLoops() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate self-crossing debt entry %s", key)
		}
		seen[key] = true
		if d.loops < 1 || d.area <= 0 {
			t.Errorf("%s debt entry is empty (loops %d, area %g) — delete it instead", key, d.loops, d.area)
		}
	}
}
