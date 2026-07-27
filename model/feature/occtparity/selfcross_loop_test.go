// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/ops"
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
// WHAT IT HAS ALREADY BOUGHT. The population it opened at — 17 loops on 11 cases — is down to 12 on 8: the
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

// describeSelfCrossing names each offending face, its surface and the area it pinches off.
func describeSelfCrossing(bad []ops.SelfCrossingLoop) string {
	out := ""
	for _, b := range bad {
		out += fmt.Sprintf(" [%T face %d loop %d pinches %.6g]", b.Face.Geometry(), b.Face.ID(), b.Loop, b.Area)
	}
	return out
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
// self-crosses: 7 loops on 4 cases, of 1155 faces across the scored corpus (it was 17 on 11 of 1144,
// then 12 on 8, then 9 on 6).
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
//   - FAR-END TRIM RUNS OFF ITS STOP FACE (complex/D8's two corner rounds, and the same shape on
//     complex/F2 and simple/Q5). trimTerminalSection slides every station of the band's terminal
//     section onto the stop face's EXTENDED implicit surface (extendableWall says so outright) with no
//     check that the landing is on the FACE. complex/D8's r=30 band stops on a radius-24 quarter round
//     that spans u ∈ [−π/2, 0] while its own section reaches u = +0.2527 — 6.064 of developed length
//     onto the flat wall next door — so the round's loop carries a boundary edge that leaves it. The
//     lobe is closed form: R·∫₀^asin(0.25)(v_up(u)+100)du = 1.2111 against the wall's own 3307.1168,
//     and BOTH mirror walls measure 1.21187. Fixing it needs the stop face's boundary RE-TRIMMED (the
//     run-out consumes a whole rim edge and shortens the next ruling), which transformLoop's
//     single-vertex substitution cannot express — see selfcross-trim-report.md §5.
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
func knownSelfCrossingLoops() []selfCrossDebtEntry {
	return []selfCrossDebtEntry{
		{"Q5", "simple", 2, 84913},   // f12723 (the r=2500 band's host wall) 84912.4, f12719 0.284334
		{"F2", "complex", 2, 1105},   // 1098.03 / 7.16978 (was 4 loops: 28.1712 and 7.78603 retired)
		{"D8", "complex", 2, 1.2119}, // BOTH mirror corner rounds, identically 1.21187
		{"W2", "simple", 1, 1e-12},   // 2.49967e-13 — a degenerate crossing, float noise
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
