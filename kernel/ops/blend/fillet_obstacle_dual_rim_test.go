// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"fmt"
	"path/filepath"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/math"
)

// dualRimCase names one corpus case whose fillet reaches the DUAL-host construction — two qualifying
// obstacles, four corner-blend panels, two split obstacle walls — together with the number of each
// boss's rim segments that legitimately carry NO curve because a boundary node was never refined ONTO
// the rim (analyticNode's coupled-node exemption).
//
// The pick locator is the corpus.json record's own (midpoint + radius), so each entry drives the very
// fillet the corpus scores.
type dualRimCase struct {
	name     string
	stepPath string
	pickMid  math.Point3
	radius   float64
	bossA    dualBossShape
	bossB    dualBossShape
}

// dualBossShape is the measured shape of ONE boss's dip rim on ONE case, so every population these gates
// tolerate is a number in the table rather than a silent exemption inside an assertion:
//
//   - offRim: wall-ring segments that legitimately carry NO curve because a boundary node was never
//     refined ONTO the rim (analyticNode's coupled-node exemption).
//   - tJunctions: rim pairs the PANELS present that are not adjacent on the wall's ring. 0 is healthy.
//     S4 and T7 carry 4 each on boss A, pre-existing and NOT caused by the dip-rim slice: their two core
//     panels present boss A's rim points out of order. Measured on S4's shipped panel1 loop, its A-side
//     runs x = −7.135 (the node) → −0.980 → −1.951 → … → −7.071 → 0, i.e. it jumps to the far end,
//     walks back, and jumps again, so two of its pairs skip 8 ring segments. The point list is
//     dipRimSubArcSnapped's and is byte-identical at base f633c0ce (the slice's diff touches `curves`
//     only), so this is a defect this gate now RECORDS, not one it caused. It is not convicted by
//     knownRetracingLoops / knownSelfCrossingLoops, which is itself worth the next slice's attention.
//   - runs: disjoint runs of the ring the panels' coverage forms. 1 is healthy; the same S4/T7 boss-A
//     ordering leaves 2.
type dualBossShape struct {
	offRim     int
	tJunctions int
	runs       int
}

// dualRimCases are EVERY corpus case that reaches buildDualPanelFaces / buildDualWallsAndWings and whose
// rim geometry these gates can therefore observe.
//
// U4 is the case this construction was built for and the only one whose SHIPPED body the slice moved.
// simple/S4 and simple/T7 run the identical construction (measured: both detect exactly 2 obstacles,
// partition into 4 panels, and build both split walls), and an instrumented edgeCatalog.use census over a
// full FilletEdges showed their nil-mismatch counts changing base→HEAD (S4 124→48, T7 116→48) — i.e. the
// dual dip-rim construction really does run on them, even though their shipped bodies are byte-identical
// (their fingerprint pins never moved) because every dip-rim curve they hand the catalog is discarded.
// "Byte-identical results" is NOT "untouched construction", and a gate that drives U4 alone leaves that
// construction unobserved on two thirds of the cases that execute it. Their residual 48 nil-mismatches
// each are OTHER shared-curve families (wing arcs, setback seam rails), not the dip rim — see the report's
// concern on edgeCatalog.use.
//
// T7 also buys rim-TYPE coverage U4 cannot give: its boss-B rim is a geom.EllipseFull, where the rim's
// own parameter midpoint and the chord midpoint genuinely differ, while U4's boss B is an imported
// b-spline and S4's bosses are circles.
func dualRimCases() []dualRimCase {
	return []dualRimCase{
		{"U4", u4StepPath, u4EdgeMidpoint, u4FilletRadius,
			dualBossShape{offRim: 4, tJunctions: 0, runs: 1}, dualBossShape{offRim: 0, tJunctions: 0, runs: 1}},
		{"S4", filepath.Join(corpusFixtureDir, "simple/S4.step"), math.P3(0, -15, 0), 8,
			dualBossShape{offRim: 4, tJunctions: 4, runs: 2}, dualBossShape{offRim: 0, tJunctions: 0, runs: 1}},
		{"T7", filepath.Join(corpusFixtureDir, "simple/T7.step"), math.P3(0, -13, 0), 6,
			dualBossShape{offRim: 4, tJunctions: 4, runs: 2}, dualBossShape{offRim: 0, tJunctions: 0, runs: 1}},
	}
}

// dualRimRun is one case's dual-host assembly driven ONCE through its shipping functions: both split
// obstacle walls (buildDualWallsAndWings) and all four corner-blend panels (buildDualPanelFaces), from
// the SAME body and edgeFillet so every shared rim point compares by exact VALUE rather than by tolerance.
type dualRimRun struct {
	kase         dualRimCase
	ef           edgeFillet
	res          tol.Resolution
	detA, detB   obstacleDetection
	wallA, wallB filletFace
	panelFaces   []filletFace
	panels       []panelSpan
}

// runDualRim builds that fixture for one case. It goes through buildDualPanelFaces /
// buildDualWallsAndWings — the two functions assembleDualObstacleSet itself calls — so the gates below
// see the loops the body SHIPS, including panelBSideSeg's reversal (reverseRingSeg), not a side traced
// by hand.
func runDualRim(t *testing.T, kase dualRimCase) dualRimRun {
	t.Helper()
	ef, res := singleEdgeFillet(t, importStepSolid(t, kase.stepPath), kase.name, kase.pickMid, kase.radius)
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("%s fixture: detectObstacles = (%d dets, ok=%v), want (2, true)", kase.name, len(dets), ok)
	}
	return assembleDualRimRun(t, kase, ef, res, dets, partitionUnionStations(dets, ef))
}

// assembleDualRimRun completes runDualRim: the four panels and the two split walls, all from the same
// detections, with every "ok" checked so a case that silently stops reaching the construction fails loud
// instead of quietly gating nothing.
func assembleDualRimRun(t *testing.T, kase dualRimCase, ef edgeFillet, res tol.Resolution,
	dets []obstacleDetection, spans []panelSpan) dualRimRun {
	t.Helper()
	panels := dualPanelSpans(spans)
	detA, detB, hok := hostDetections(dets)
	panelFaces, pok := buildDualPanelFaces(ef, dets, panels, res)
	wallA, wallB, _, wok := buildDualWallsAndWings(ef, dets, spans, panels, res)
	if !hok || !pok || !wok || len(panels) != 4 || len(panelFaces) != 4 {
		t.Fatalf("%s fixture: hosts=%v panels=(%d,%v→%d faces) walls=%v", kase.name, hok, len(panels), pok, len(panelFaces), wok)
	}
	return dualRimRun{kase: kase, ef: ef, res: res, detA: detA, detB: detB,
		wallA: wallA, wallB: wallB, panelFaces: panelFaces, panels: panels}
}

// dualBoss pairs one boss's detection with its split wall face and the number of that wall's rim segments
// that legitimately carry NO curve because one of their endpoints is not ON the rim.
type dualBoss struct {
	name  string
	det   obstacleDetection
	face  filletFace
	shape dualBossShape
}

// bosses returns the two bosses under test. Host A's COUPLED node is not refined onto its rim
// (coupledNodeStation) — on U4 it sits 4.04e-03 off the r=8 circle, x²+z² = 63.935 not 64 — so the two
// segments touching each of its two nodes have no rim sub-arc and must keep the honest straight chord;
// boss B's nodes are refined on all three cases, so every one of its rim segments must trace the rim.
func (r dualRimRun) bosses() []dualBoss {
	return []dualBoss{
		{name: "bossA", det: r.detA, face: r.wallA, shape: r.kase.bossA},
		{name: "bossB", det: r.detB, face: r.wallB, shape: r.kase.bossB},
	}
}

// dipRimRingOf returns a split obstacle wall's RIM ring — its loop without the three preserved seam/top
// entries buildDualSplitWall appends last (seam up, closed top rim, seam down). The ring closes on itself:
// its last segment runs the final rim point back to rim sample 0, which is the seam foot.
func dipRimRingOf(loop filletLoop) filletLoop {
	n := len(loop.pts) - 3
	return filletLoop{pts: loop.pts[:n], curves: loop.curves[:n], srcV: loop.srcV[:n], srcE: loop.srcE[:n]}
}

// TestDualSplitWallRimSegmentsTraceTheRim is the geometry gate for the DUAL path's rim, on the two real
// split walls of EVERY corpus case that reaches the dual construction (dualRimCases: U4, S4, T7): every rim
// segment whose straight chord would leave the exact rim by more than the model weld must carry a curve
// that lies ON the rim, pinned to its own two vertices and no longer than the rim's own per-segment span.
//
// It replaces TestSpliceRimPointKeepsTheSplitSegmentsConic, which split each segment at a point taken from
// that segment's OWN arc. That is the one input for which the old inversion-based trim could succeed, and
// the one input the shipped path never supplies: an interior seam station is solved on the exact rim
// (dipRimPointAtStation bisects to ~1e-12) while the segment's own per-segment CIRCULAR fit of U4's
// imported b-spline rim sits ~1e-7 off it — a per-segment FIT RESIDUAL measured against a WHOLE-MODEL
// weld. On the shipped path that weld is res.Weld() = 6.442049363e-08 (U4's boundingDiag 64.420493634)
// and the two inversions read 8.723146033e-08 (1.354x) and 7.634559529e-08 (1.185x), so BOTH halves fell
// back to straight chords. The margin is narrow; the KIND of comparison is what makes it hopeless (that
// rim's own interior per-segment bar is 1.771921e-07, ~2.75x the weld). The old gate was GREEN throughout
// — four of boss B's rim halves shipped as chords 5.002371e-03 / 4.967703e-03 / 2.675363e-03 /
// 2.674992e-03 off the panels they bound (and 4.30e-03 / 4.28e-03 / 2.08e-03 / 2.08e-03 off the obstacle
// wall, the same chords against two different faces). This gate drives the SHIPPED wall.
//
// (Earlier revisions of this note quoted "3.394e-08 weld". That figure is
// tol.ForPoints(d.holeSampled.pts).Weld() = 3.39411225647e-08, the DELETED test's own resolution
// over the rim sample points, not the res.Weld() the shipped splitRimSegmentCurve call is passed.)
//
// Measured on boss B's four split halves (deviation from the exact rim CURVE, distanceToCurve's
// 4096-station sweep + 60 bisections): the chords they replaced sit 2.256595e-03 / 4.659682e-03 /
// 4.688347e-03 / 2.256281e-03 off the rim and the sub-arcs 9.613018e-09 / 3.024564e-08 / 4.735324e-08 /
// 5.918872e-09 — inside that rim's OWN interior per-segment bar of 1.771921e-07, which is what the
// comparative bar asserts. Boss A's ring is a circle and its arcs read ~3.2e-15 against a 1.004e-12 bar.
func TestDualSplitWallRimSegmentsTraceTheRim(t *testing.T) {
	if testing.Short() {
		t.Skip("corpus tier (~5s): `make test-corpus`")
	}
	t.Parallel()
	for _, kase := range dualRimCases() {
		t.Run(kase.name, func(t *testing.T) {
			r := runDualRim(t, kase)
			for _, b := range r.bosses() {
				t.Run(b.name, func(t *testing.T) {
					assertRingTracesTheRim(t, b, dipRimRingOf(b.face.loops[0]), r.res.Weld())
				})
			}
		})
	}
}

// assertRingTracesTheRim applies the rule to every segment of one wall's rim ring, re-counts the off-rim
// exceptions so the exempt population can never silently grow, and re-counts the segments actually
// ASSERTED so the gate can never pass vacuously on a ring it failed to reach.
func assertRingTracesTheRim(t *testing.T, b dualBoss, ring filletLoop, weld float64) {
	t.Helper()
	rim := b.det.holeEdge.Geometry()
	bars := interiorRimBars(ring, rim, nonSampleSegments(ring, b.det))
	offRim, subArcs := 0, 0
	for seg := range ring.curves {
		off, asserted := assertRimSegmentTracesTheRim(t, ring, seg, rim, bars, weld)
		offRim, subArcs = offRim+boolCount(off), subArcs+boolCount(asserted)
	}
	if offRim != b.shape.offRim {
		t.Errorf("%s: %d rim segments have an endpoint OFF the rim, want %d — the unrefined-node exemption changed", b.name, offRim, b.shape.offRim)
	}
	assertRingWasActuallyChecked(t, b.name, subArcs, len(ring.curves))
}

// assertRimSegmentTracesTheRim applies the rule to ONE rim segment. It reports whether that segment is an
// off-rim exception (an endpoint not on the rim, so the honest straight chord is the only answer) and
// whether it was actually asserted to be a rim sub-arc.
func assertRimSegmentTracesTheRim(t *testing.T, ring filletLoop, seg int, rim geom.Curve3,
	bars rimSubArcBars, weld float64) (off, asserted bool) {
	t.Helper()
	a, c := ring.pts[seg], ring.pts[(seg+1)%len(ring.pts)]
	if !endpointsOnRim(rim, a, c, weld) {
		assertSegmentHasNoCurve(t, ring, seg)
		return true, false
	}
	if maxCurveDeviationFrom(geom.NewLineSegment(a, c), rim) <= weld {
		// A micro-span whose own chord already traces the rim: nothing to carry. There is exactly one on
		// U4 — boss B's ring segment 50, the 9.784e-04 stub between rim sample 63 and the z=0 seam station,
		// where Arc3dByThreePoints declines on three near-collinear points. Its chord's deviation from the
		// exact rim is 9.936895e-09, BELOW the 6.442049363e-08 model weld, so the straight chord costs
		// nothing measurable there.
		return false, false
	}
	assertSegmentIsRimSubArc(t, ring, seg, rim, bars)
	return false, true
}

// assertRingWasActuallyChecked is the anti-vacuity ratchet. A ring these gates never reach — an empty
// loop, a case that stopped building panels, a fixture wired to the wrong face — would otherwise satisfy
// every rule by asserting nothing. On a 1/64-sampled rim of radius r the per-segment chord sagitta is
// ~r·(2π/64)²/8 ≈ 0.012·r, decades above any model weld, so all but the handful of split micro-spans MUST
// be asserted; half the ring is a floor with room to spare.
func assertRingWasActuallyChecked(t *testing.T, who string, subArcs, segs int) {
	t.Helper()
	if segs > 0 && 2*subArcs >= segs {
		return
	}
	t.Errorf("%s: only %d of %d rim segments were asserted to be rim sub-arcs — the gate is reading an empty or wrong ring, not passing", who, subArcs, segs)
}

// boolCount folds a predicate result into a running tally.
func boolCount(b bool) int {
	if b {
		return 1
	}
	return 0
}

// assertSegmentHasNoCurve requires a segment with an off-rim endpoint to keep the straight chord: a curve
// there would be a rim sub-arc through a point that is not on the rim, and — because the panels read the
// same rule — a curve one consumer carries and the other does not.
func assertSegmentHasNoCurve(t *testing.T, ring filletLoop, seg int) {
	t.Helper()
	if c := curveAt(ring.curves, seg); c != nil {
		t.Errorf("rim segment %d has an endpoint OFF the rim yet carries a %T — there is no rim sub-arc to trim there", seg, c)
	}
}

// endpointsOnRim reports whether both of a segment's endpoints lie on the rim curve, measured by an
// independent 4096-station sweep + bisection (distanceToCurve), never by the production inversion the
// code under test uses.
func endpointsOnRim(rim geom.Curve3, a, b math.Point3, weld float64) bool {
	return distanceToCurve(a, rim) <= weld && distanceToCurve(b, rim) <= weld
}

// nonSampleSegments returns the ring segments that are NOT plain sample-to-sample spans of the rim's own
// 1/64 discretization — the node splits and the interior seam splits. They are excluded from the
// self-calibrating bars so the bars are measured on untouched geometry only.
func nonSampleSegments(ring filletLoop, d obstacleDetection) []int {
	sample := map[math.Point3]bool{}
	for _, p := range d.holeSampled.pts {
		sample[p] = true
	}
	var out []int
	for seg := range ring.curves {
		if !sample[ring.pts[seg]] || !sample[ring.pts[(seg+1)%len(ring.pts)]] {
			out = append(out, seg)
		}
	}
	return out
}

// TestDualDipRimConsumersAgreeByValue is the WELD gate for the dual dip rim, on every case in
// dualRimCases (U4, S4, T7). The split obstacle wall and the four corner-blend panels tile ONE rim
// between them, and assembleBody's edge catalog is first-writer-wins (edgeCatalog.use keeps the first
// face's curve and silently discards every later one), so "the panels happen to agree" is not agreement —
// it is agreement by face build order.
//
// The panels' rim sides used to hand `make([]geom.Curve3, len(pts)-1)`: all nil, for every dip-rim segment
// of every panel. Measured on the shipped U4 body, all 28 of those nil curves were discarded because the
// walls REPLACE body faces and are assembled before the panels are appended as extras — reorder the two
// and the whole dual dip rim reverts to chords with no gate to catch it. They now read the same
// rimSubArcBetween of the same two endpoint VALUES the wall reads.
//
// Two invariants: every consecutive rim pair a panel presents must be a segment of that boss's own wall
// ring (matched by exact point value) carrying the same curve, and the segments so covered must form the
// boss's RECORDED number of runs — 1 contiguous run spanning its own two boundary nodes everywhere except
// S4's and T7's boss A, whose pre-existing out-of-order panel side leaves 2 (dualBossShape). A gap or a
// doubled segment is the discretizeEdge station-count drift this project has turned into a T-junction mesh
// leak four times; the recorded exceptions are asserted exactly, in both directions, so neither a new one
// nor a silently retired one can pass.
func TestDualDipRimConsumersAgreeByValue(t *testing.T) {
	t.Parallel()
	for _, kase := range dualRimCases() {
		t.Run(kase.name, func(t *testing.T) {
			r := runDualRim(t, kase)
			for _, b := range r.bosses() {
				t.Run(b.name, func(t *testing.T) { assertPanelsAgreeWithWall(t, r, b) })
			}
		})
	}
}

// assertPanelsAgreeWithWall matches every panel's rim pairs against one wall's rim ring by exact point
// value and requires the coverage to be that boss's recorded run/T-junction shape.
func assertPanelsAgreeWithWall(t *testing.T, r dualRimRun, b dualBoss) {
	t.Helper()
	ring := dipRimRingOf(b.face.loops[0])
	cov := newRimCoverage()
	for i, pf := range r.panelFaces {
		matchRimSegments(t, fmt.Sprintf("panel%d", i), pf.loops[0], len(pf.loops[0].pts), ring, cov)
	}
	assertRimTJunctionCount(t, b.name, cov, b.shape.tJunctions)
	assertDipRunSpansNodeToNode(t, b, ring, cov.covered)
}

// assertDipRunSpansNodeToNode requires the panels' coverage of one wall's rim ring to be that boss's
// RECORDED number of runs (1 everywhere except S4's and T7's boss A, whose pre-existing panel-side
// ordering leaves 2 — see dualBossShape), each segment presented exactly once, and — when it is the one
// healthy contiguous run — running node to node.
func assertDipRunSpansNodeToNode(t *testing.T, b dualBoss, ring filletLoop, covered map[int]int) {
	t.Helper()
	in := make([]bool, len(ring.pts))
	for seg, c := range covered {
		if c != 1 {
			t.Errorf("%s: dip rim segment %d is presented by %d panels, want exactly 1 — station counts have drifted apart", b.name, seg, c)
		}
		in[seg] = true
	}
	start, runs := runStartOf(in)
	if runs != b.shape.runs {
		t.Fatalf("%s: the panels cover %d disjoint runs of the wall's rim ring, want %d — a gap is a T-junction", b.name, runs, b.shape.runs)
	}
	if runs == 1 {
		assertRunEndsAtNodes(t, b.det, ring, in, start)
	}
}

// runStartOf returns the index that begins the covered run and how many disjoint runs there are (0 covered
// segments counts as 0 runs, all-covered as 0 too — both fail the caller's want of exactly 1).
func runStartOf(in []bool) (int, int) {
	n := len(in)
	start, runs := 0, 0
	for i := range n {
		if in[i] && !in[(i-1+n)%n] {
			start, runs = i, runs+1
		}
	}
	return start, runs
}

// assertRunEndsAtNodes requires the covered run's first and last POINTS to be the boss's own two boundary
// nodes: the panels must tile the dip rim exactly from node to node, no shorter and no further.
func assertRunEndsAtNodes(t *testing.T, d obstacleDetection, ring filletLoop, in []bool, start int) {
	t.Helper()
	n := len(in)
	end := start
	for k := 0; k < n && in[(end+1)%n]; k++ {
		end = (end + 1) % n
	}
	first, last := ring.pts[start], ring.pts[(end+1)%n]
	if (first == d.pMinus && last == d.pPlus) || (first == d.pPlus && last == d.pMinus) {
		return
	}
	t.Errorf("the panels' covered dip run spans %v -> %v, not the boss's own two boundary nodes %v / %v",
		first, last, d.pMinus, d.pPlus)
}
