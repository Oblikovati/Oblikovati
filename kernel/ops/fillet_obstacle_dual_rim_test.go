// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// dualRimRun is U4's dual-host assembly driven ONCE through its shipping functions: both split obstacle
// walls (buildDualWallsAndWings) and all four corner-blend panels (buildDualPanelFaces), from the SAME
// body and edgeFillet so every shared rim point compares by exact VALUE rather than by tolerance.
type dualRimRun struct {
	ef           edgeFillet
	res          Resolution
	detA, detB   obstacleDetection
	wallA, wallB filletFace
	panelFaces   []filletFace
	panels       []panelSpan
}

// runU4Dual builds that fixture. It goes through buildDualPanelFaces / buildDualWallsAndWings — the two
// functions assembleDualObstacleSet itself calls — so the gates below see the loops the body SHIPS,
// including panelBSideSeg's reversal (reverseRingSeg), not a side traced by hand.
func runU4Dual(t *testing.T) dualRimRun {
	t.Helper()
	_, fils, res := u4Fillet(t)
	ef := fils[0]
	dets, ok := detectObstacles(ef, res)
	if !ok || len(dets) != 2 {
		t.Fatalf("U4 fixture: detectObstacles = (%d dets, ok=%v), want (2, true)", len(dets), ok)
	}
	spans := partitionUnionStations(dets, ef)
	panels := dualPanelSpans(spans)
	detA, detB, hok := hostDetections(dets)
	panelFaces, pok := buildDualPanelFaces(ef, dets, panels, res)
	wallA, wallB, _, wok := buildDualWallsAndWings(ef, dets, spans, panels, res)
	if !hok || !pok || !wok || len(panels) != 4 {
		t.Fatalf("U4 fixture: hosts=%v panels=(%d,%v) walls=%v", hok, len(panels), pok, wok)
	}
	return dualRimRun{ef: ef, res: res, detA: detA, detB: detB, wallA: wallA, wallB: wallB, panelFaces: panelFaces, panels: panels}
}

// dualBoss pairs one boss's detection with its split wall face and the number of that wall's rim segments
// that legitimately carry NO curve because one of their endpoints is not ON the rim.
type dualBoss struct {
	name   string
	det    obstacleDetection
	face   filletFace
	offRim int
}

// bosses returns the two bosses under test. Host A's COUPLED node is not refined onto its rim
// (coupledNodeStation) — it sits 4.04e-03 off the r=8 circle, x²+z² = 63.935 not 64 — so the two segments
// touching each of its two nodes have no rim sub-arc and must keep the honest straight chord; boss B's
// nodes are refined, so every one of its rim segments must trace the rim.
func (r dualRimRun) bosses() []dualBoss {
	return []dualBoss{
		{name: "bossA", det: r.detA, face: r.wallA, offRim: 4},
		{name: "bossB", det: r.detB, face: r.wallB, offRim: 0},
	}
}

// dipRimRingOf returns a split obstacle wall's RIM ring — its loop without the three preserved seam/top
// entries buildDualSplitWall appends last (seam up, closed top rim, seam down). The ring closes on itself:
// its last segment runs the final rim point back to rim sample 0, which is the seam foot.
func dipRimRingOf(loop filletLoop) filletLoop {
	n := len(loop.pts) - 3
	return filletLoop{pts: loop.pts[:n], curves: loop.curves[:n], srcV: loop.srcV[:n], srcE: loop.srcE[:n]}
}

// TestDualSplitWallRimSegmentsTraceTheRim is the geometry gate for the DUAL path's rim, on U4's two real
// split walls: every rim segment whose straight chord would leave the exact rim by more than the model
// weld must carry a curve that lies ON the rim, pinned to its own two vertices and no longer than the
// rim's own per-segment span.
//
// It replaces TestSpliceRimPointKeepsTheSplitSegmentsConic, which split each segment at a point taken from
// that segment's OWN arc. That is the one input for which the old inversion-based trim could succeed, and
// the one input the shipped path never supplies: an interior seam station is solved on the exact rim
// (dipRimPointAtStation bisects to ~1e-12) while the segment's own per-segment CIRCULAR fit of U4's
// imported b-spline rim sits ~1e-7 off it, so the inversion measured 8.723146e-08 against a 3.394e-08
// weld and BOTH halves fell back to straight chords. The old gate was GREEN throughout — four of boss B's
// rim halves shipped as chords 5.002371e-03 / 4.967703e-03 / 2.675363e-03 / 2.674992e-03 off the panels
// they bound (and 4.30e-03 / 4.28e-03 / 2.08e-03 / 2.08e-03 off the obstacle wall, the same chords against
// two different faces). This gate drives the SHIPPED wall.
//
// Measured on boss B's four split halves (deviation from the exact rim CURVE, distanceToCurve's
// 4096-station sweep + 60 bisections): the chords they replaced sit 2.256595e-03 / 4.659682e-03 /
// 4.688347e-03 / 2.256281e-03 off the rim and the sub-arcs 9.613018e-09 / 3.024564e-08 / 4.735324e-08 /
// 5.918872e-09 — inside that rim's OWN interior per-segment bar of 1.771921e-07, which is what the
// comparative bar asserts. Boss A's ring is a circle and its arcs read ~3.2e-15 against a 1.004e-12 bar.
func TestDualSplitWallRimSegmentsTraceTheRim(t *testing.T) {
	r := runU4Dual(t)
	for _, b := range r.bosses() {
		t.Run(b.name, func(t *testing.T) {
			assertRingTracesTheRim(t, b, dipRimRingOf(b.face.loops[0]), r.res.Weld())
		})
	}
}

// assertRingTracesTheRim applies the rule to every segment of one wall's rim ring and re-counts the
// off-rim exceptions, so the exempt population can never silently grow.
func assertRingTracesTheRim(t *testing.T, b dualBoss, ring filletLoop, weld float64) {
	t.Helper()
	rim := b.det.holeEdge.Geometry()
	bars := interiorRimBars(ring, rim, nonSampleSegments(ring, b.det))
	offRim := 0
	for seg := range ring.curves {
		a, c := ring.pts[seg], ring.pts[(seg+1)%len(ring.pts)]
		if !endpointsOnRim(rim, a, c, weld) {
			offRim++
			assertSegmentHasNoCurve(t, ring, seg)
			continue
		}
		if maxCurveDeviationFrom(geom.NewLineSegment(a, c), rim) <= weld {
			continue // a micro-span whose own chord already traces the rim: nothing to carry
		}
		assertSegmentIsRimSubArc(t, ring, seg, rim, bars)
	}
	if offRim != b.offRim {
		t.Errorf("%s: %d rim segments have an endpoint OFF the rim, want %d — the unrefined-node exemption changed", b.name, offRim, b.offRim)
	}
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

// TestDualDipRimConsumersAgreeByValue is the WELD gate for the dual dip rim. The split obstacle wall and
// the four corner-blend panels tile ONE rim between them, and assembleBody's edge catalog is
// first-writer-wins (edgeCatalog.use keeps the first face's curve and silently discards every later one),
// so "the panels happen to agree" is not agreement — it is agreement by face build order.
//
// The panels' rim sides used to hand `make([]geom.Curve3, len(pts)-1)`: all nil, for every dip-rim segment
// of every panel. Measured on the shipped U4 body, all 28 of those nil curves were discarded because the
// walls REPLACE body faces and are assembled before the panels are appended as extras — reorder the two
// and the whole dual dip rim reverts to chords with no gate to catch it. They now read the same
// rimSubArcBetween of the same two endpoint VALUES the wall reads.
//
// Two invariants: every consecutive rim pair a panel presents must be a segment of that boss's own wall
// ring (matched by exact point value) carrying the same curve, and the segments so covered must form ONE
// contiguous run of the ring spanning the boss's own two boundary nodes — a gap or a doubled segment is
// the discretizeEdge station-count drift this project has turned into a T-junction mesh leak four times.
func TestDualDipRimConsumersAgreeByValue(t *testing.T) {
	r := runU4Dual(t)
	for _, b := range r.bosses() {
		t.Run(b.name, func(t *testing.T) {
			ring := dipRimRingOf(b.face.loops[0])
			covered := map[int]int{}
			for i, pf := range r.panelFaces {
				matchRimSegments(t, fmt.Sprintf("panel%d", i), pf.loops[0], len(pf.loops[0].pts), ring, covered)
			}
			assertDipRunSpansNodeToNode(t, b.det, ring, covered)
		})
	}
}

// assertDipRunSpansNodeToNode requires the panels' coverage of one wall's rim ring to be a single
// contiguous run, each segment presented exactly once, running node to node.
func assertDipRunSpansNodeToNode(t *testing.T, d obstacleDetection, ring filletLoop, covered map[int]int) {
	t.Helper()
	n := len(ring.pts)
	in := make([]bool, n)
	for seg, c := range covered {
		if c != 1 {
			t.Errorf("dip rim segment %d is presented by %d panels, want exactly 1 — station counts have drifted apart", seg, c)
		}
		in[seg] = true
	}
	start, runs := runStartOf(in)
	if runs != 1 {
		t.Fatalf("the panels cover %d disjoint runs of the wall's rim ring, want 1 contiguous run — a gap is a T-junction", runs)
	}
	assertRunEndsAtNodes(t, d, ring, in, start)
}

// runStartOf returns the index that begins the covered run and how many disjoint runs there are (0 covered
// segments counts as 0 runs, all-covered as 0 too — both fail the caller's want of exactly 1).
func runStartOf(in []bool) (int, int) {
	n := len(in)
	start, runs := 0, 0
	for i := 0; i < n; i++ {
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
