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
// self-crosses: 17 loops on 11 cases, of 1144 faces across the scored corpus. Each ceiling is the
// measured pinched-off area, in the surface's own metric chart, rounded UP a little so float noise
// cannot fail it while a real growth does. Derived by an instrumented corpus-wide sweep at
// ops.PropertyQuality(), not from any report.
//
// The roots, largest first (selfcross-trim-report.md):
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
//     single-vertex substitution cannot express — see the report's §"what the fix needs".
//   - SPHERE-PATCH SEAM (simple/E4, simple/W1): a fillet corner sphere whose loop runs along the seam;
//     W1's pinches off 0 area, i.e. the crossing is degenerate.
//   - PLANAR HOST RETRIM (simple/M4 N3 N9 W2 Y2 Y4, and Q5's cap): a plane's own retrimmed loop
//     crossing itself; W2's is 2.5e-13, i.e. float noise on an otherwise simple loop.
func knownSelfCrossingLoops() []selfCrossDebtEntry {
	return []selfCrossDebtEntry{
		{"Q5", "simple", 2, 84913},   // f12758 (the r=2500 band's host wall) 84912.4, f12754 0.284334
		{"F2", "complex", 4, 1105},   // 1104.69 / 28.1712 / 7.78603 / 7.16978
		{"E4", "simple", 1, 128.49},  // 128.489
		{"M4", "simple", 1, 59.585},  // 59.5848
		{"Y2", "simple", 2, 50.001},  // 50 / 1.41397
		{"Y4", "simple", 1, 23.911},  // 23.9109
		{"N3", "simple", 1, 4.5540},  // 4.55391
		{"N9", "simple", 1, 4.1496},  // 4.14956
		{"D8", "complex", 2, 1.2119}, // BOTH mirror corner rounds, identically 1.21187
		{"W2", "simple", 1, 1e-12},   // 2.49967e-13 — a degenerate crossing, float noise
		{"W1", "simple", 1, 1e-12},   // 0 — a degenerate seam crossing that pinches off nothing
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
