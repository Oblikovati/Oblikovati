// SPDX-License-Identifier: GPL-2.0-only

package occtparity

import (
	"fmt"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
)

// loopSegOnFaceTol is the default budget for "every loop segment lies on the face it bounds",
// expressed RELATIVE to the result body's bounding diagonal so it is scale-invariant (a tscale
// SCALE=1000 fixture gets the same budget as its unit twin, ADR-0042). 1e-6 of the diagonal is
// ~4 decades above the construction noise a correctly-built rail carries (analytic arcs land at
// 1e-16..1e-13 relative; a canal loft's own approximation at ~1e-10), so it separates a
// CONSTRUCTION defect from arithmetic without being a tolerance to tune.
const loopSegOnFaceTol = 1e-6

// TestEveryLoopSegmentLiesOnItsFace is the corpus-wide invariant guard: on the SHIPPED body of every
// scored case, every boundary edge of every face must have its CURVE on that face's own surface.
//
// This is the invariant a whole class of defects violates silently — a face whose boundary is not on
// its own surface still validates as a watertight solid (the vertices weld, the loops close), so the
// scoreboard cannot see it, but the tessellator then tiles a boundary that is not on the surface and
// every downstream consumer inherits the error. B5/C4/D7 shipped an arc 12.36 (94% of the fillet
// radius) off its own cylinder while two of them were merely FAIL(area) and L3 was fully GREEN.
//
// Cases still carrying a residual are listed EXPLICITLY in knownOffSurfaceDebt with the measured
// ceiling, so the guard is a RATCHET, not a tolerance: a listed case may improve freely, but growing
// past its ceiling — or ANY unlisted case exceeding loopSegOnFaceTol — fails loud. Each debt entry is
// a real, separately-rooted defect (see offsurface-loopseg-report.md §2 for the taxonomy); the table
// shrinks as those roots are fixed and must never be widened to accommodate a regression.
func TestEveryLoopSegmentLiesOnItsFace(t *testing.T) {
	dir := CorpusFixtureDir()
	debt := offSurfaceDebtIndex()
	for _, r := range Corpus() {
		body, ok := shippedCaseBody(r, dir)
		if !ok {
			continue // skipped / faulty: no single healthy body to measure
		}
		assertLoopSegmentsOnFaces(t, r, body, debt[r.Grid+"/"+r.Case])
	}
}

// assertLoopSegmentsOnFaces fails when the body's worst boundary-edge-off-its-own-face residual, taken
// relative to the bounding diagonal, exceeds the case's budget (its debt ceiling, else the default).
func assertLoopSegmentsOnFaces(t *testing.T, r Record, body *topo.Body, budget float64) {
	t.Helper()
	if budget == 0 {
		budget = loopSegOnFaceTol
	}
	worst, where := worstLoopSegmentOffFace(body)
	if rel := worst / boundingDiag(body); rel > budget {
		t.Errorf("%s/%s: loop segment leaves the face it bounds by %.6g (rel %.4g of the %.4g diagonal, budget %.4g) — %s",
			r.Grid, r.Case, worst, rel, boundingDiag(body), budget, where)
	}
}

// shippedCaseBody runs one case's real fillet and returns its single result body, or ok=false when the
// case is skipped (TODO / variable-radius / quarantine / import divergence) or the fillet is unhealthy —
// a faulty result has no invariant to hold.
func shippedCaseBody(r Record, dir string) (*topo.Body, bool) {
	if _, decided := preRunOutcome(r); decided {
		return nil, false
	}
	body, err := importInput(filepath.Join(dir, r.InputStep))
	if err != nil {
		return nil, false
	}
	sets, ok := scoreLocate(r, body)
	if !ok {
		return nil, false
	}
	res, filletOK, _ := runFillet(body, sets)
	if !filletOK || len(res) != 1 || res[0] == nil {
		return nil, false
	}
	return res[0], true
}

// worstLoopSegmentOffFace returns the largest distance from any face's boundary-edge curve to that
// face's OWN surface, plus a description of the offender (face surface type, edge id, curve type).
func worstLoopSegmentOffFace(b *topo.Body) (float64, string) {
	worst, where := 0.0, "(none)"
	for _, f := range b.Faces() {
		s := f.Geometry()
		if s == nil {
			continue
		}
		for _, e := range boundaryEdgesOf(f) {
			d, ok := curveOffSurface(e.Geometry(), s)
			if !ok || d <= worst {
				continue
			}
			worst = d
			where = fmt.Sprintf("face %d (%T) bounded by edge %d (%T)", f.ID(), s, e.ID(), e.Geometry())
		}
	}
	return worst, where
}

// boundaryEdgesOf returns each DISTINCT edge in f's loops (a seam edge used twice is measured once).
func boundaryEdgesOf(f *topo.Face) []*topo.Edge {
	seen := map[uint64]bool{}
	var out []*topo.Edge
	for _, l := range f.Loops() {
		for _, u := range l.EdgeUses() {
			if e := u.Edge(); e != nil && !seen[e.ID()] {
				seen[e.ID()] = true
				out = append(out, e)
			}
		}
	}
	return out
}

// curveOffSurface returns the max distance from 17 evenly-spaced samples of c to its closest point on
// s. Sampling (rather than a closed-form residual) is what makes this uniform across every curve and
// surface pair the kernel builds; 17 stations resolve a quarter-arc's mid-span bulge to <0.2% of the
// sagitta, far finer than the defects it must catch.
func curveOffSurface(c geom.Curve3, s geom.Surface) (float64, bool) {
	if c == nil {
		return 0, false
	}
	lo, hi := c.Domain()
	const stations = 17
	worst := 0.0
	for i := 0; i <= stations; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/float64(stations))
		_, _, foot := geom.ClosestPointOnSurface(s, p)
		if d := p.DistanceTo(foot); d > worst {
			worst = d
		}
	}
	return worst, true
}

// offSurfaceDebtEntry is one case's measured off-surface ceiling, relative to its bounding diagonal.
type offSurfaceDebtEntry struct {
	name string
	grid string
	rel  float64
}

// offSurfaceDebtIndex keys knownOffSurfaceDebt by "grid/case" for O(1) lookup.
func offSurfaceDebtIndex() map[string]float64 {
	out := make(map[string]float64, len(knownOffSurfaceDebt()))
	for _, d := range knownOffSurfaceDebt() {
		out[d.grid+"/"+d.name] = d.rel
	}
	return out
}

// knownOffSurfaceDebt is the FULL population of cases whose shipped body still carries a boundary edge
// off its own face, each capped at 1.1× its measured residual (relative to the bounding diagonal) so a
// regression fires while an improvement is free. Derived by an instrumented corpus-wide sweep, not from
// any report. It was 52 entries; the far-end wall trim (fillet_farend_trim.go) retired 17 of them —
// A5 A6 A8 B1 B5 B9 C4 C7 D3 D7 E1 E2 F2 F6 I3 M2 Q5 are now CLEAN — and shrank 5 more (N5 0.0216→0.000129,
// V1 0.00451→0.000295, V3 0.00712→0.00199, V5 0.00145→0.000313, complex/D8 0.0461→0.037); the elliptic
// survivor-rim carry (fillet_survivor_rim_ellipse.go) then retired F7 (35 → 34), the rim-fillet HOST-SEAM
// carry (fillet_rim_build.go's wallSeamCurve → retainedHostSeamCurve) retired J2 and J4 (34 → 32), which
// also cut the table's CEILING 5.4× — 0.334 → complex/F2's 0.0616 — and the far-end trim's stop-face BRANCH
// pick (fillet_farend_trim.go's slideOntoWall → nearestHitOnSide) retired complex/D8 (32 → 31). The table
// must SHRINK, never widen.
//
// ★ NINE ceilings re-capped at 1.1× a NEW measurement by the edge catalog's nil-vs-curve adoption
// (kernel/ops/assemble_curve_agreement.go): a consumer that HAS the boundary curve now supplies it to
// the shared edge instead of losing it to the neighbour's straight chord, so the residual these entries
// measure is the residual of the chord that no longer ships. Measured with the gate's own function
// (worstLoopSegmentOffFace / boundingDiag), base → HEAD: T7 0.00196→0.000429, S9 0.00181→0.000793,
// T1 0.00142→0.000497, S1 0.000958→0.000320, S4 0.000926→0.000334, T4 0.000904→0.000491,
// T3 0.000722→0.000397, S6 0.000583→0.000545, S7 0.000454→0.000420. Entry count unchanged at 31 —
// every one is smaller, none retired. ★ A limitation to read before trusting these as adoption
// gates (adversarial-review finding m-2): S6's 0.0006 and S7's 0.000463 ceilings sit ABOVE those
// cases' own BASE residuals (5.826e-4 / 4.537e-4 — their adoption gain was only 1.07×/1.08×, so
// 1.1× the new measurement clears the old one), so this gate does NOT fire on S6/S7 if the
// adoption is reverted — it fires on the other SEVEN of the nine. They are honest re-measurements,
// not regression gates; S6/S7's adoption regression cover is their fingerprint pins and
// t7_adoption_perface_test.go's family of per-face evidence.
//
// ★ Re-capped AGAIN by the footprint-rim curve carry (t3-plane-sliver-report.md): the setback family's
// residual was the rim CHORD itself (a LineSegment between two on-rim points sits a sagitta off the
// doubly-curved wall it bounds), and those segments now carry exact sub-spans of the footprint conic.
// Measured (same function), adoption HEAD → this HEAD: S7 4.202e-4→3.8e-10, T4 4.910e-4→4.6e-9,
// T7 4.289e-4→4.1e-8 (all three now BELOW the default budget — entries DELETED, count 31 → 28),
// T1 4.974e-4→5.963e-6, S1 3.197e-4→4.157e-5, S4 3.340e-4→8.097e-5, S6 5.451e-4→1.748e-4,
// T3 3.973e-4→2.393e-4; S9 7.927e-4 unchanged (its worst residual is not a rim chord) and holds.
//
// ★ Re-capped a THIRD time by the railB interpolated contact-locus carry (railb-locus-report.md):
// what remained on the six setback survivors was the railB contact-locus POLYLINE — degree-1 chords
// through the 7 node contacts, lying IN the pInner host plane (partner residual ~1e-15) but off the
// lofted patch by the chord sagitta. contactLocusRail now interpolates EVERY solved station contact
// into the loft's own boundary row (geom.CanalFootLocusRail), and both consumers (patch loop + host
// notch detour) carry its per-segment sub-spans by value. Measured (same function,
// worstLoopSegmentOffFace/boundingDiag), previous HEAD → this HEAD: S9 7.927231e-4→3.198130e-9,
// T3 2.393473e-4→4.900577e-10, S6 1.748055e-4→1.834108e-10, S4 8.097143e-5→6.279003e-10,
// S1 4.157203e-5→1.072957e-9, T1 5.963266e-6→2.172440e-7 — ALL SIX below the default budget, so
// their entries are DELETED per TestOffSurfaceDebtIsWellFormed (count 28 → 22). Each case's worst
// residual is now a DIFFERENT family (S1/T1 the reversedCurve3 arm-vs-cylinder fit, S9 the torus
// rim TrimmedCurve3 at 3.2e-9, all measured in railb-locus-report.md §4); the CANAL-rail taxonomy
// bullet below predates this carry for the S/T setback cases.
// The roots that remain, largest first (stopface-reversed-report.md):
//
//   - CURVED-HOST RETRIM ARC off its own cylinder (complex/F2, 0.0616) — a `fillet:x` retrim edge, not a
//     rim carry. (J2/J4's 0.334/0.165 CHORDED SEAM MERIDIAN entries are retired: the rim rebuild used to
//     re-aim the host seam as a straight ruling, which is only the host's meridian on a cylinder / cone /
//     elliptical-cylinder — on J2's SPHERE and J4's TORUS the meridian is an ARC, so they shipped a 90.38 /
//     61.24-long chord 28.44 / 10.43 off their own host. The seam is now the retained sub-span of the
//     host's own meridian: 2.7e-14 / 1.6e-13, and J4 GREENED from +70.2% body area. Note for the record
//     that the entry these two carried before that also blamed the WRONG thing — "an imported base-face
//     loop the fillet never touched" — which the input measurement falsified, both bodies importing clean
//     to 1e-13.)
//     (complex/D8's 0.037 entry is retired too, and the reason recorded for it — "endFaceAt picks the first
//     face at the terminal vertex that is neither A nor B, so the far-end trim lands the section on a
//     plausible-but-wrong wall" — was MEASURED FALSE: both of D8's terminal vertices are valence-3 with
//     exactly ONE non-A/B face, so endFaceAt had no choice to get wrong and picked the correct stop wall
//     both times. The real root was the far-end slide's ±branch tie, fillet_farend_trim.go's
//     slideOntoWall: on a stop wall symmetric about the section plane the two crossings are EXACTLY
//     equidistant, so "nearest" was decided by the intersector's output order INDEPENDENTLY at each of the
//     33 stations, and the station list alternated between the wall's two branches. The trim now picks the
//     branch on the stop FACE's own side: 18.8877 → 2.3e-06, rel 0.0336 → 4.1e-09.)
//     (F7's 0.29 entry, the ELLIPSE-rim chord, is retired: the elliptic survivor-rim carry
//     — fillet_survivor_rim_ellipse.go — now re-derives that rim from the parent's own eccentric angles,
//     taking F7's worst residual 89.4426 → 3.4e-14 and its per-face gross error vs DRAWEXE 40793 → 2.8.)
//   - SPREAD-FAN / oblique-plane chords (P8 P9 V8 V9 V1 V3 V5 X9 K7 L1 L7 N5): a run-out spread cap is
//     tiled as a chord fan, so each chord sits a sagitta off the curved wall it spans.
//   - CANAL / BSpline patch rails (C2 C6 C8 S1 S4 S6 S7 S9 T1 T3 T4 T7 U4): the rail is
//     built on the exact rolling-ball envelope while the patch is a fitted BSpline, so rail and patch
//     agree only to the fit's own residual (reverse-segment-fix-report.md §6 concern 2).
//   - ★ The five MID-SPAN OBSTACLE cases (R9 S3 T6 U3 X3) are now two decades below that, and their root
//     has changed twice. It was read as the fitted-BSpline patch; obstacle-canal-report.md made the patch
//     the EXACT envelope and the residual survived, which re-rooted it on the two straight RIM CHORDS the
//     node split left with NO curve at all (insertNodesIntoRim: "a straight truncated chord"). Confirmed
//     on the shipped bodies by closed form — the residual WAS that chord's own sagitta to a few tenths of
//     a percent (R9 6.978437e-03 vs 8(1−cos(Δθ/2)) = 6.997e-03 off its r=8 boss cylinder; S3 9.394e-03
//     with the cone's own 0.9701 normal projection; U3/X3 1.561e-02 / 2.959e-02 off the patch AND
//     1.197e-02 / 1.601e-02 off the UNTOUCHED obstacle wall, the same chord against two different faces).
//     Those four segments now carry the rim conic TRIMMED at the node's own rim parameter, and every one
//     of the five improved 70×–159×:
//     R9 1.550764e-04 → 1.174647e-06   S3 1.708003e-04 → 1.375036e-06   T6 1.693403e-04 → 1.780964e-06
//     U3 3.170438e-04 → 1.987829e-06   X3 1.595493e-04 → 2.268098e-06
//     (Note for the record that the adversarial review's reading of R9's 6.978437e-03 as a "wing-tangent
//     chord" was itself wrong: both its endpoints lie on x²+y²=64 at z=10, one of them the node on the
//     receded tangent line y=−7 and the other rim sample 42 — it was the node RIM chord all along.)
//     What DOMINATES each case now is a different, narrower root, newly VISIBLE rather than newly caused:
//     R9/S3/U3/X3 the canal WALL-FRONT chord against the patch (5.29e-05 / 7.56e-05 / 9.79e-05 /
//     4.21e-04 — the wall-foot station polyline vs the surface it is a station list of), and T6 the
//     ELLIPTIC rim's own per-segment circular fit (1.224210e-04 off its EllipticalCylinder, rimSegmentArc
//     over a 1/64 span of an a=15,b=10 ellipse).
//   - ★ U4 (the DUAL-host case) carried the same node-chord root one level down and now shrinks with it:
//     spliceRimPoint dropped a split segment's curve on both halves, discarding two of the four trimmed
//     node sub-arcs the rim had just gained. splitRimSegmentCurve trims the segment's own conic at the
//     station instead, taking U4 from 2.233973e-04 to 1.493883e-04 and its ceiling 0.000246 → 0.000165.
//     What dominates U4 now is host A's COUPLED node, which analyticNode does not refine onto the rim
//     (coupledNodeStation), so its r=8 boss rim keeps the honest straight chord — 9.623665e-03, both
//     endpoints on x²+z²=64 at y=−20. That solve is the queued dual-host item, not this root.
//     ★ U4's ceiling therefore HOLDS at 0.000165 through the dual dip-rim slice (headroom 1.105×), and
//     that is the honest outcome, not an oversight: the DUAL path's own rim defect was live but SECOND.
//     splitRimSegmentCurve recovered the split parameter off the SEGMENT's per-segment circular fit rather
//     than off the rim, and on U4's imported b-spline rim that fit is ~1e-7 out while the station is exact
//     to ~1e-12 — a per-segment FIT RESIDUAL tested against a WHOLE-MODEL weld. On the shipped path that
//     weld is res.Weld() = 6.442049363e-08 (U4 boundingDiag 64.420493634), and the two inversions read
//     8.723146033e-08 (1.354x) and 7.634559529e-08 (1.185x), so both rejected and four of boss B's rim
//     halves shipped as chords — 5.002371e-03 / 4.967703e-03 / 2.675363e-03 / 2.674992e-03 off the panels
//     and 4.301259e-03 / 4.275629e-03 / 2.080290e-03 / 2.079314e-03 off the obstacle wall. (The 3.394e-08
//     an earlier revision quoted was the deleted regression test's ResolutionForPoints(holeSampled).Weld()
//     = 3.39411225647e-08, not the shipped weld; the margins were 1.19-1.35x, not 2.25-2.57x.) They now
//     trace the rim (rimSubArcBetween) at 5.09e-05 and below, but they were never U4's WORST offender, so
//     the ceiling cannot move until the coupled node is solved.
//     ★ No other case in the 475 changes RESULT — but three cases run the construction, not one. An
//     instrumented edgeCatalog.use census over a full FilletEdges shows simple/S4 (124→48) and simple/T7
//     (116→48) nil-mismatch counts moving base→HEAD, i.e. both execute buildDualPanelFaces and both hand
//     the catalog different curves than they did at base. Their shipped bodies are byte-identical (their
//     fingerprint pins never moved) only because every dip-rim curve they hand is discarded, and each
//     still carries 48 residual nil-mismatches from OTHER shared-curve families. "Byte-identical results"
//     is not "untouched construction", so kernel/ops' dualRimCases now drives the by-value and trace-the-rim
//     gates on S4 and T7 as well as U4.
func knownOffSurfaceDebt() []offSurfaceDebtEntry {
	return []offSurfaceDebtEntry{
		{"F2", "complex", 0.0616}, {"C2", "simple", 0.0192},
		{"V9", "simple", 0.017}, {"P9", "simple", 0.017}, {"X9", "simple", 0.00284},
		{"V3", "simple", 0.00199}, {"C8", "simple", 0.00129},
		{"P8", "simple", 0.00061}, {"V8", "simple", 0.00061},
		// The railB interpolated contact-locus carry (railb-locus-report.md) took ALL NINE setback
		// entries' family below the default budget — the six survivors measured S9 3.198130e-9,
		// T3 4.900577e-10, S6 1.834108e-10, S4 6.279003e-10, S1 1.072957e-9, T1 2.172440e-7
		// (worstLoopSegmentOffFace/boundingDiag) — so S9/T3/S6/S4/S1/T1 are deleted per
		// TestOffSurfaceDebtIsWellFormed (28 → 22), joining S7/T4/T7 which the footprint-rim carry
		// (t3-plane-sliver-report.md) had already retired at 3.8e-10 / 4.6e-9 / 4.1e-8.
		{"V5", "simple", 0.000313}, {"V1", "simple", 0.000295},
		{"U4", "simple", 0.000165}, {"K7", "simple", 0.000152},
		{"L1", "simple", 0.000133}, {"N5", "simple", 0.000129}, {"L7", "simple", 0.000128},
		{"C6", "simple", 0.000121},
		// The five mid-span obstacle cases, each re-capped at 1.1x its post-sub-arc measurement (above).
		{"X3", "simple", 0.0000025}, {"U3", "simple", 0.00000219}, {"T6", "simple", 0.00000196},
		{"S3", "simple", 0.00000151}, {"R9", "simple", 0.00000129},
	}
}

// TestOffSurfaceDebtIsWellFormed keeps the debt table honest: no duplicate key, and every ceiling
// LOOSER than the default budget (an entry at or below it would be dead weight hiding nothing).
func TestOffSurfaceDebtIsWellFormed(t *testing.T) {
	seen := map[string]bool{}
	for _, d := range knownOffSurfaceDebt() {
		key := d.grid + "/" + d.name
		if seen[key] {
			t.Errorf("duplicate off-surface debt entry %s", key)
		}
		seen[key] = true
		if d.rel <= loopSegOnFaceTol {
			t.Errorf("%s debt ceiling %g is within the default budget %g — delete the entry instead", key, d.rel, loopSegOnFaceTol)
		}
	}
}
