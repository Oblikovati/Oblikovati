// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/ops"
)

// TestNoFaceLoopRetracesItself is the corpus-wide RATCHET on the half of "is this boundary a simple
// polygon" that TestNoFaceLoopSelfCrossesOnItsSurface structurally cannot see.
//
// WHY IT IS A SEPARATE RATCHET AND NOT MORE ROWS IN THE OTHER ONE. The self-crossing guard asks
// simpleLoop2D's predicate — do two edges TRANSVERSALLY intersect — and records the AREA the crossing
// pinches off. A boundary that back-tracks along a collinear sibling scores exactly zero there (two
// overlapping collinear segments never straddle each other's line), and it pinches off no area at all:
// the two traversals contribute equal and opposite shoelace terms. So the defect has a different
// detector AND a different unit — a LENGTH, not an area — and quoting one as the other would be the
// category error selfcross-trim-report.md §8.5 already had to warn about once.
//
// WHY IT MATTERS EVEN THOUGH IT COSTS NO AREA. Precisely BECAUSE it costs no area: an area gate cannot
// see it. Every one of the loops below is on a body whose area the scoreboard accepts, and three of the
// four cases are fully GREEN — simple/Y1 measures 61327.876 against OCCT's 61327.9 while its host plane
// runs 10 out along z = 90 and 10 straight back. The polygon is still not simple, so it still has no
// well-defined interior and no correct triangulation, and conformingPlaneMesh still silently falls back
// to the ear-clip on it.
//
// It is a RATCHET, not a tolerance: every case that still carries one is listed with its MEASURED count
// and retraced length, so a listed case may improve freely while a new one — or a listed one growing —
// fails loud. The table must shrink and must never be widened to accommodate a regression.
func TestNoFaceLoopRetracesItself(t *testing.T) {
	dir := CorpusFixtureDir()
	q := ops.PropertyQuality()
	debt := retraceDebtIndex()
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no single healthy body to measure
		}
		assertRetraceWithinDebt(t, r, ops.RetracingFaceLoops(body, q), debt[r.Grid+"/"+r.Case])
	}
}

// assertRetraceWithinDebt fails when a case carries more retracing loops than its recorded debt, or one
// that retraces further than recorded. A case with no entry may carry none at all.
func assertRetraceWithinDebt(t *testing.T, r Record, bad []ops.RetracingLoop, debt retraceDebtEntry) {
	t.Helper()
	if len(bad) > debt.loops {
		t.Errorf("%s/%s: %d face loop(s) retrace their own boundary, recorded debt %d — %s",
			r.Grid, r.Case, len(bad), debt.loops, describeRetracing(bad))
	}
	for _, b := range bad {
		if b.Overlap > debt.overlap {
			t.Errorf("%s/%s: face %d loop %d covers %.6g of its own boundary twice (ceiling %.6g)",
				r.Grid, r.Case, b.Face.ID(), b.Loop, b.Overlap, debt.overlap)
		}
	}
}

// describeRetracing names each offending face, its surface and the length it covers twice.
func describeRetracing(bad []ops.RetracingLoop) string {
	out := ""
	for _, b := range bad {
		out += fmt.Sprintf(" [%T face %d loop %d retraces %.6g]", b.Face.Geometry(), b.Face.ID(), b.Loop, b.Overlap)
	}
	return out
}

// retraceDebtEntry is one case's measured retrace debt: how many of its loops back-track, and the
// longest stretch any of them covers twice, in the surface's own metric chart.
type retraceDebtEntry struct {
	name, grid string
	loops      int
	overlap    float64
}

// retraceDebtIndex keys knownRetracingLoops by "grid/case" for O(1) lookup.
func retraceDebtIndex() map[string]retraceDebtEntry {
	out := make(map[string]retraceDebtEntry, len(knownRetracingLoops()))
	for _, d := range knownRetracingLoops() {
		out[d.grid+"/"+d.name] = d
	}
	return out
}

// knownRetracingLoops is the FULL measured population of shipped faces whose developed boundary runs
// back over ground it already covered: 4 loops on 3 cases, of the healthy shipped bodies of the scored
// corpus (it opened at 7 loops on 4 cases).
//
// ★ NEWLY DETECTED, NOT NEWLY CAUSED. This table opened at the population the detector found on the
// SHIPPED body of the base tree; nothing in the slice that added it changed one line of production
// geometry. It is pre-existing debt becoming visible for the first time.
//
// RETIRED, first wave: simple/B2 (1 loop, 10) and 2 of simple/N6's 3 (5 each) — the ARC-fillet setback
// end-cap merge (kernel/ops/fillet_arc_endcap.go). rebuildWithArcFillet closed each end of its torus
// band with a flat setback triangle in the RADIAL plane through that end; on a sector solid the cut wall
// IS that plane, so the triangle was emitted as a second, coincident face while the wall kept its own
// un-receded corner and the cap face's loop was routed out to that corner and straight back. The rim
// vertex is now SUPERSEDED by the cap-tangent point: the side face absorbs the cap, the cap∩side edge is
// re-ended on the tangent point, and no separate cap face (and no rim vertex) is built. As with every
// earlier wave the faces now match their own CLOSED FORMS rather than merely stopping retracing: B2's
// two sector walls 5000 → 4978.5319 (closed form 4978.5398, DRAWEXE 4978.54) and its whole body 8 faces
// → the 6 DRAWEXE ships, retiring a +85.84 (+0.401 %) surplus that sat inside a 1 % PASS; N6's x = 50
// wall 2100 → 2105.3670 (closed form 2105.3650, DRAWEXE 2105.37). Gated on those closed forms by
// TestB2ArcFilletMergesItsSetbackCapsIntoTheSectorWalls and TestN6ArcFilletMergesOnlyTheRadialEnd.
// ★ Both cases also went WATERTIGHT — B2 3 → 0 free edges at property quality, N6 6 → 0 at both
// qualities — so their knownMeshLeaks rows went with them, which is the correspondence that table
// predicted ("closing the retracing root should retire these four").
//
// MEASURING FUNCTION: ops.RetracingFaceLoops on faceOuterBoundary+faceHoleBoundaries developed by
// toUVLoops into the surface's metric chart, QUALITY ops.PropertyQuality() (the parity path). The
// quantity is a LENGTH in that chart — a retrace encloses zero area, so there is no area to quote.
//
// ★ AND EVERY ONE OF THEM HAS A CLOSED-FORM TARGET: the retraced length is the SETBACK the blend
// applies to that face, i.e. the fillet radius. simple/N6 r=5 → 5, simple/W2 r=0.2 → 0.2. Measured
// against those radii the worst residual is 2.2e-16 relative — double-precision exact, not a captured
// number.
// ★ simple/Y1's 10 is NOT independent confirmation: that fixture's slot is also 10 wide and its spike
// runs along the slot's roof, so radius and slot width coincide. It is quoted as a measurement, not as
// a derivation.
//
// THE ROOT, in one sentence: the retrim ADDS the setback point without REMOVING what that point
// supersedes, so the rebuilt loop runs out to the superseded geometry and straight back. Two variants,
// both still here: the un-receded VERTEX is kept alongside its own tangent point (N6's (80,10,10) beside
// (80,10,5); W2's y=0 beside y=−0.2), or the setback LINE is drawn across an obstacle it should stop at
// (Y1's z=90 run over the slot at x ∈ [90,100]). It is the same SETBACK-BAND-OVERRUN root
// knownSelfCrossingLoops records for simple/Y2 and simple/Y4, and the band∩obstacle imprint walk
// (kernel/ops/fillet_band_imprint.go) closed it for those two; the walk's own gate
// (bandImprintQualifies) declines every case here — Y1 explicitly, on the measured grounds that its
// area was already right — so the root survives in its DEGENERATE form, where the overshoot lands
// COLLINEAR with what it overshoots instead of transverse to it. That is why it produces a zero-area
// spike rather than a crossing, and why no area gate can see it.
//
// ★ TWO OF THE THREE REMAINING CASES ARE GREEN — simple/Y1 and simple/N6 are PASS. simple/Y1 is the
// sharpest instance the corpus has produced: the band-imprint walk deliberately DECLINES it (r=10 puts
// the contact line exactly on the slot's roof, so no cut falls strictly inside the box) on the measured
// grounds that its body area was already right — 61327.876 against OCCT's 61327.9 — and it is: the spike
// contributes nothing to the shoelace. Only simple/W2 is FAIL(area), and for an unrelated reason.
//
// Largest first:
//
//   - simple/Y1 (r=10) — host plane y=0, the r=10 member of the Y slot family: (90,0,90) → (100,0,90) →
//     (0,0,90), i.e. out to the box's own corner and straight back along z=90.
//   - simple/N6 (r=5) and simple/W2 (r=0.2) are ★ RETIRED, both by the arc band's RUN-OUT termination
//     (kernel/ops/fillet_arc_runout.go), and their history is worth keeping because it corrects what this
//     table used to say about them.
//     N6's spike was (80,10,10)→(80,10,5)→(80,10,80) on the x=80 host plane — the arc fillet's SECOND
//     end, whose radial setback plane is 0.6435 rad off that wall, so the coplanar end-cap merge declines
//     it. This entry said "closing it needs the band's terminal section trimmed against the wall, not a
//     merge", and that is exactly what landed: the setback cap's tip lands at (77,14,10), which is inside
//     the POCKET (removed material), so the cap was drawn into the void and the wall kept a corner the
//     cyl-tangent point ran r past. Terminating the band on the wall's own spiric section instead ships
//     DRAWEXE's own nine faces — x=80 at 1406.80 against 1406.8, the pocket floor at 641.95 against
//     641.965, the band at 254.43 against 254.441 — and no retrace.
//     W2's two spikes were 0.2 at (2.9859,1,0)→(2.9859,−0.2,0)→(2.9859,0,0) and (2,0,1)→(2,−0.2,1)→
//     (2,1,1). Its ends decline the coplanar merge for a different reason: the imported cylinder's ref
//     direction is not exact, so each setback plane sits 1.0e-4 rad off the side face (atan(1e-4/1) from
//     a centre at z=0.9999 against a rim point at z=1). ★★ The larger defect this entry recorded — the
//     whole band built on the VOID side — was ATTRIBUTED HALF-RIGHT and is now settled by DRAWEXE: it
//     took BOTH the Reversed cap normal (pl.Normal() points into the material) AND the groove radial
//     sense (material lies outside that cylinder, so the ball rides at cylR+r = 1.2, not cylR−r = 0.8),
//     and even both together only move the band to the material side — the cylR+r cove then spills
//     0.19998 = r below the bottom plane and W2's mesh leak widens 3/3 → 8/29. The run-out is what closes
//     it: OCCT terminates the band on the bottom plane's spiric section, and W2 now ships DRAWEXE's seven
//     faces at 11.766423 against 11.76665 with 0 retraces, 0 leaks and 0 self-crossings.
func knownRetracingLoops() []retraceDebtEntry {
	return []retraceDebtEntry{
		{"Y1", "simple", 1, 10.001}, // 10 = r = the slot width; see the caveat above
	}
}

// TestRetraceDebtIsWellFormed keeps the debt table honest: no duplicate key, and every entry claiming
// at least one retracing loop (a zero entry would be dead weight hiding nothing).
func TestRetraceDebtIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range knownRetracingLoops() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate retrace debt entry %s", key)
		}
		seen[key] = true
		if d.loops < 1 || d.overlap <= 0 {
			t.Errorf("%s debt entry is empty (loops %d, overlap %g) — delete it instead", key, d.loops, d.overlap)
		}
	}
}
