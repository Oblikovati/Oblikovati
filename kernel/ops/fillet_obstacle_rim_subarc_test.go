// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// rimSubArcChordFraction is the sub-arc/CHORD separator, expressed as a fraction of the very chord
// deviation it replaced rather than as an absolute distance: a node sub-arc must sit at least 10x closer
// to the rim than this rim's OWN interior per-segment chords do. It replaced an absolute 1e-4, which is
// a fixed length in a project whose tolerances are model-relative (ADR-0042, Resolution/scale.Weld) —
// the residual it separates scales with the model (the same 1/64 chord measures 9.6e-03 on R9's r=8 rim
// and 3.6e-02 on X3's r=30 one), so the bar must scale with it too. Measured headroom at 0.1: R9 9.60e-04
// vs 3.2e-15, S3 1.20e-03 vs 4.8e-15, T6 1.80e-03 vs 7.86e-05 (23x), U3 1.44e-03 vs 5.7e-15,
// X3 3.60e-03 vs 1.5e-14. The real, non-tunable bar is the comparative one below (no worse than the rim's
// own interior segments), which is what a self-calibrating gate needs.
const rimSubArcChordFraction = 0.1

// rimSubArcExtentFactor is the ceiling on a node sub-arc's LENGTH, as a multiple of the rim's own longest
// interior per-segment arc — the EXTENT half of the gate. Without it, on a circular rim the complementary
// MAJOR arc (same conic, same two endpoints, exactly on the rim, the long way round) is caught only by
// conditioning noise: the adversarial review's M5 mutation measured 1.477e-12 against a 1.004e-12 bar,
// i.e. it passed on 2 of 4 segments. A node sub-arc is a sub-span of ONE 1/64 segment, so it is always
// shorter than the longest of them (worst measured ratio 0.966, U3); the complementary arc is ~63/64 of
// the whole rim, 42x above this ceiling. 1.5 keeps 55 % headroom over the worst legitimate ratio.
const rimSubArcExtentFactor = 1.5

// rimSubArcPinTol is how far a shared segment's curve may start/end from that segment's own two
// vertices. Measured worst on the SHIPPED loops (buildNotchedHost's reversed splice included): 9.72e-15
// on X3, two decades below this.
const rimSubArcPinTol = 1e-12

// rimSubArcSlack is the rounding floor added to the comparative bar (a node sub-arc must be no worse than
// the worst INTERIOR per-segment arc of the same rim). On the four circular rims both sit at ~1.3e-14, so
// a bare <= would be a coin flip on the last bits; 1e-12 is two decades above that and ten decades below
// the chord it guards against.
const rimSubArcSlack = 1e-12

// rimSubArcAgreeTol is how far two consumers' curves for the SAME rim segment may disagree, sampled
// point-for-point. Measured across all five bodies and all 148 shared rim segments the worst disagreement
// is 1.42e-14 — pure rounding: a reversed traversal re-derives the arc through its own recovered midpoint
// (reversedArcThroughMid), which is the same conic sub-arc, so the only difference is the order the three
// defining points are fed to Arc3dByThreePoints. 1e-12 is 70x that and twelve decades below any real
// disagreement (a nil against an arc is caught by name, and a WRONG arc is chord-scale, >=1e-5).
const rimSubArcAgreeTol = 1e-12

// TestObstacleRimNodeSegmentsAreTrimmedSubArcs is the geometry half of the node-chord gate, on all five
// single-host obstacle bodies: the four rim segments the two boundary nodes split must each carry an ARC
// pinned to its own two vertices and lying ON the obstacle's exact rim curve.
//
// They used to carry NO curve at all — a straight truncated chord — and that was the largest single
// off-surface residual in the obstacle class (R9 7.00e-03 off its own r=8 boss cylinder; U3 1.56e-02 and
// X3 2.96e-02 off the corner-blend patch AND 1.20e-02 / 1.60e-02 off the untouched obstacle wall, the
// same chord against two different faces). Restoring the nil makes this test RED on all five.
func TestObstacleRimNodeSegmentsAreTrimmedSubArcs(t *testing.T) {
	for _, c := range singleHostObstacleCorpus {
		t.Run(c.name, func(t *testing.T) {
			d := runSingleHostCase(t, c).d
			rim := d.holeEdge.Geometry()
			wall := insertNodesIntoRim(d)
			nodeSegs := nodeAdjacentSegments(t, d, wall)
			bars := interiorRimBars(wall, rim, nodeSegs)
			for _, seg := range nodeSegs {
				assertSegmentIsRimSubArc(t, wall, seg, rim, bars)
			}
		})
	}
}

// TestObstacleRimConsumersAgreeByValue is the WELD half: the notched host, the split obstacle wall and
// the corner-blend patch tile ONE rim between them, and assembleBody's edge catalog is first-writer-wins
// (edgeCatalog.use keeps the first face's curve for a shared segment and silently discards the rest). So
// agreement here has to be by VALUE, not by build order and not "within tolerance".
//
// Two invariants, both asserted directly rather than inferred from a leak census:
//   - STATION AGREEMENT: every consecutive point pair the notch or the patch presents on the rim must be
//     a consecutive pair of the WALL's own rim ring, matched by exact point value, and together they must
//     cover every one of the wall's rim segments exactly once. A station count mismatch in either
//     direction (the discretizeEdge invariant this project has broken four times, each time a T-junction
//     mesh leak) fails here rather than downstream in a free-edge count.
//   - CURVE AGREEMENT: for each matched pair the two consumers' curves must span the segment's own two
//     vertices and trace the same point set, forward or reversed, to rimSubArcAgreeTol — measured in
//     BOTH directions (see assertCurvesCoincide).
//
// ★ It drives the notch through buildNotchedHost — the REAL shipping path — not through hostSideSubArc:
// see shippedNotchLoop.
func TestObstacleRimConsumersAgreeByValue(t *testing.T) {
	for _, c := range singleHostObstacleCorpus {
		t.Run(c.name, func(t *testing.T) {
			r := runSingleHostCase(t, c)
			wall := insertNodesIntoRim(r.d)
			notch := shippedNotchLoop(t, c.name, r)
			patch := patchBoundaryLoop(r.d, r.og, r.of)
			cov := newRimCoverage()
			matchRimSegments(t, "notch", notch, len(notch.pts), wall, cov)
			matchRimSegments(t, "patch", patch, len(patch.pts), wall, cov)
			assertRimTJunctionCount(t, c.name, cov, 0)
			assertEveryWallRimSegmentCovered(t, r.d, wall, cov.covered)
		})
	}
}

// singleHostRun is one corpus obstacle case driven ONCE through the real pipeline: the detection, the
// obstacle feature the patch reads, and the filletRebuildMaps buildNotchedHost consumes. All four come
// from the SAME body and the SAME edgeFillet, which is what lets the maps lookup (keyed on face
// identity) resolve and the three consumers' points compare by exact value.
type singleHostRun struct {
	d    obstacleDetection
	og   obstacleGeom
	of   *ObstacleFeature
	maps filletRebuildMaps
}

// runSingleHostCase runs resolveFilletPicks → computeCorners → computeFillets → detectObstacle →
// buildObstacleFeature on one corpus obstacle case and bundles what the rim gates read.
func runSingleHostCase(t *testing.T, c singleHostObstacleCase) singleHostRun {
	t.Helper()
	body := importCorpusBody(t, c.step)
	ef, res := singleEdgeFillet(t, body, c.name, c.mid, c.radius)
	d, ok := detectObstacle(ef, res)
	if !ok {
		t.Fatalf("%s: detectObstacle found no single-host obstacle", c.name)
	}
	of, og, ok := buildObstacleFeature(ef, d, res)
	if !ok {
		t.Fatalf("%s: buildObstacleFeature declined", c.name)
	}
	maps, _ := filletBuildMaps(body, []edgeFillet{ef})
	return singleHostRun{d: d, og: og, of: of, maps: maps}
}

// shippedNotchLoop returns the notched host face's boundary loop AS SHIPPED: buildNotchedHost →
// mergeHoleIntoNotch → orientedHostArc → (reverseOpenArc) → spliceSubArc.
//
// Calling hostSideSubArc directly instead bypasses the REVERSAL, and that is the branch four of the five
// bodies actually take (orientedHostArc reverses whenever nodes[0] is the nearer one). Proven by the
// adversarial review's M10 mutation: gutting reverseOpenArc's curve propagation — every reversed segment
// handed nil — left the by-value gate GREEN while only the leak census noticed. The gate exists so a
// T-junction or a dropped curve is caught by a GATE, so the reversal has to be inside it.
func shippedNotchLoop(t *testing.T, name string, r singleHostRun) filletLoop {
	t.Helper()
	face, ok := buildNotchedHost(r.d, r.maps)
	if !ok {
		t.Fatalf("%s: buildNotchedHost declined — the gate cannot see the shipped notch", name)
	}
	return face.loops[0]
}

// nodeAdjacentSegments returns the four indices of wall's rim ring whose segment touches an inserted
// node: the two legs of each node's split segment. It locates them by the node POINT, so it cannot drift
// out of step with insertNodesIntoRim's own indexing.
func nodeAdjacentSegments(t *testing.T, d obstacleDetection, wall filletLoop) []int {
	t.Helper()
	var out []int
	for i, p := range wall.pts {
		if p != d.pMinus && p != d.pPlus {
			continue
		}
		out = append(out, (i-1+len(wall.pts))%len(wall.pts), i) // the leg into the node, and out of it
	}
	if len(out) != 4 {
		t.Fatalf("expected 4 node-adjacent rim segments (2 nodes x 2 legs), found %d", len(out))
	}
	return out
}

// assertSegmentIsRimSubArc checks one segment's curve: present, pinned to its own two vertices, ON the
// rim curve, and no LONGER than the span it is a sub-arc of.
func assertSegmentIsRimSubArc(t *testing.T, loop filletLoop, seg int, rim geom.Curve3, bars rimSubArcBars) {
	t.Helper()
	c := curveAt(loop.curves, seg)
	if c == nil {
		t.Errorf("rim segment %d carries NO curve — the straight truncated node chord is back", seg)
		return
	}
	a, b := loop.pts[seg], loop.pts[(seg+1)%len(loop.pts)]
	assertCurveSpansSegment(t, "wall", seg, c, a, b)
	assertSubArcOnRim(t, seg, maxCurveDeviationFrom(c, rim), bars)
	if l := curveSampledLength(c); l > bars.extent {
		t.Errorf("rim segment %d sub-arc is %.6g long, past the rim's own per-segment span (ceiling %.6g) — endpoints and rim alone also admit the COMPLEMENTARY arc, the long way round",
			seg, l, bars.extent)
	}
}

// assertSubArcOnRim applies the two off-rim ceilings: chord-scale separation and the comparative bar.
func assertSubArcOnRim(t *testing.T, seg int, dev float64, bars rimSubArcBars) {
	t.Helper()
	if dev > bars.onRim {
		t.Errorf("rim segment %d curve leaves the exact rim by %.3e — that is chord-scale, not sub-arc scale (ceiling %.3e, a tenth of this rim's own chord deviation)", seg, dev, bars.onRim)
	}
	if dev > bars.arc {
		t.Errorf("rim segment %d trimmed sub-arc is %.3e off the rim, WORSE than the rim's own interior per-segment arcs (bar %.3e)", seg, dev, bars.arc)
	}
}

// rimSubArcBars are the three ceilings one rim sets for its OWN node sub-arcs. All three are measured
// from the same loop on the same rim, so every one of them calibrates itself per case and none is a
// tuned absolute: onRim is a tenth of the CHORD deviation the sub-arcs replaced, arc is "no worse than
// the rim's untouched interior per-segment arcs", extent is "no longer than its longest one".
type rimSubArcBars struct {
	onRim, arc, extent float64
}

// interiorRimBars measures the three bars over the rim's UNTOUCHED interior per-segment arcs (the
// discretization the node sub-arcs belong to) and applies each bar's factor. Measured per case (chord /
// arc / length): R9 9.60e-03 / 3.77e-15 / 0.785, S3 1.20e-02 / 5.02e-15 / 0.982, T6 1.80e-02 / 1.23e-04 /
// 1.471, U3 1.44e-02 / 6.22e-15 / 1.178, X3 3.60e-02 / 1.78e-14 / 2.945.
func interiorRimBars(wall filletLoop, rim geom.Curve3, exclude []int) rimSubArcBars {
	skip := map[int]bool{}
	for _, s := range exclude {
		skip[s] = true
	}
	var raw rimSubArcBars
	for seg := range wall.curves {
		if skip[seg] || curveAt(wall.curves, seg) == nil {
			continue
		}
		raw = widenRimBars(raw, wall, seg, rim)
	}
	return rimSubArcBars{onRim: raw.onRim * rimSubArcChordFraction, arc: raw.arc + rimSubArcSlack,
		extent: raw.extent * rimSubArcExtentFactor}
}

// widenRimBars folds one interior segment's three measurements into the running maxima: its straight
// CHORD's deviation from the rim (what a node segment used to carry), its per-segment ARC's deviation
// (what it carries now), and that arc's length (the span a node sub-arc must fit inside).
func widenRimBars(raw rimSubArcBars, wall filletLoop, seg int, rim geom.Curve3) rimSubArcBars {
	chord := geom.NewLineSegment(wall.pts[seg], wall.pts[(seg+1)%len(wall.pts)])
	return rimSubArcBars{
		onRim:  stdmath.Max(raw.onRim, maxCurveDeviationFrom(chord, rim)),
		arc:    stdmath.Max(raw.arc, maxCurveDeviationFrom(wall.curves[seg], rim)),
		extent: stdmath.Max(raw.extent, curveSampledLength(wall.curves[seg])),
	}
}

// curveSampledLength is c's polyline length over 64 stations — a LOWER bound on the true arc length,
// which is the safe side for a ceiling that rejects a curve for being too LONG.
func curveSampledLength(c geom.Curve3) float64 {
	lo, hi := c.Domain()
	total, prev := 0.0, c.PointAt(lo)
	for i := 1; i <= 64; i++ {
		p := c.PointAt(lo + (hi-lo)*float64(i)/64)
		total, prev = total+prev.DistanceTo(p), p
	}
	return total
}

// maxCurveDeviationFrom is the largest distance from 17 samples of c to the closest point on ref, found
// by a coarse-then-golden scan of ref's own parameter (ref is a closed conic, so no inversion is needed).
func maxCurveDeviationFrom(c, ref geom.Curve3) float64 {
	lo, hi := c.Domain()
	worst := 0.0
	for i := 0; i <= 17; i++ {
		if d := distanceToCurve(c.PointAt(lo+(hi-lo)*float64(i)/17), ref); d > worst {
			worst = d
		}
	}
	return worst
}

// distanceToCurve is the distance from p to ref, by a 4096-station sweep refined by 60 bisections of the
// best bracket — enough to resolve 1e-15 on a conic of model size, so the measure has no floor of its own
// that could hide a sub-arc's error.
func distanceToCurve(p math.Point3, ref geom.Curve3) float64 {
	lo, hi := ref.Domain()
	const n = 4096
	best, bt := stdmath.Inf(1), lo
	for i := 0; i <= n; i++ {
		t := lo + (hi-lo)*float64(i)/n
		if d := p.DistanceTo(ref.PointAt(t)); d < best {
			best, bt = d, t
		}
	}
	step := (hi - lo) / n
	for range 60 {
		step /= 2
		for _, t := range []float64{bt - step, bt + step} {
			if d := p.DistanceTo(ref.PointAt(t)); d < best {
				best, bt = d, t
			}
		}
	}
	return best
}

// rimCoverage accumulates what a wall's rim ring received from the consumers that share it: how many
// times each ring segment was presented, and one description per presented pair whose two points are NOT
// adjacent on the ring — a T-junction. The T-junctions are COLLECTED rather than failed on the spot so a
// caller with recorded pre-existing debt (dualRimCases' boss A on S4/T7) can assert the exact count
// instead of the gate being unable to run at all.
type rimCoverage struct {
	covered    map[int]int
	tJunctions []string
}

// newRimCoverage starts an empty coverage tally.
func newRimCoverage() *rimCoverage {
	return &rimCoverage{covered: map[int]int{}}
}

// matchRimSegments matches every one of loop's first segCount consecutive point pairs that lies on the
// rim (by exact point value against wall's ring) to a segment of that ring, records which one each
// covered, and checks the two consumers' curves for that segment coincide.
func matchRimSegments(t *testing.T, who string, loop filletLoop, segCount int, wall filletLoop, cov *rimCoverage) {
	t.Helper()
	index := map[math.Point3]int{}
	for i, p := range wall.pts {
		index[p] = i
	}
	for i := range segCount {
		matchOneRimPair(t, who, loop, i, wall, index, cov)
	}
}

// matchOneRimPair folds a single consumer pair into the coverage: skipped when it is not a rim pair at
// all, recorded as a T-junction when its two rim points are not adjacent on the ring, otherwise counted
// and curve-compared against the wall's own curve for that segment.
func matchOneRimPair(t *testing.T, who string, loop filletLoop, i int, wall filletLoop,
	index map[math.Point3]int, cov *rimCoverage) {
	t.Helper()
	a, b := loop.pts[i], loop.pts[(i+1)%len(loop.pts)]
	ia, oka := index[a]
	ib, okb := index[b]
	if !oka || !okb {
		return // not a rim segment (a wall-front or wing-arc side of the patch loop)
	}
	seg, ok := wallSegmentBetween(ia, ib, len(wall.pts))
	if !ok {
		cov.tJunctions = append(cov.tJunctions,
			fmt.Sprintf("%s: rim pair %v->%v is not a segment of the wall's rim ring (indices %d,%d) — T-junction", who, a, b, ia, ib))
		return
	}
	cov.covered[seg]++
	assertCurvesCoincide(t, who, seg, curveAt(loop.curves, i), curveAt(wall.curves, seg), a, b)
}

// assertRimTJunctionCount requires the collected T-junctions to be exactly the recorded population. It is
// a ratchet in BOTH directions: a new one fails, and so does a retired one, because retiring a recorded
// defect must land with the measurement that retires it.
func assertRimTJunctionCount(t *testing.T, who string, cov *rimCoverage, want int) {
	t.Helper()
	if len(cov.tJunctions) == want {
		return
	}
	for _, msg := range cov.tJunctions {
		t.Log(msg)
	}
	t.Errorf("%s: the consumers present %d rim pairs that are not segments of the wall's ring, want %d (see the log above)",
		who, len(cov.tJunctions), want)
}

// wallSegmentBetween returns the wall ring segment index joining adjacent indices ia,ib (either
// direction), or ok=false when they are not adjacent.
func wallSegmentBetween(ia, ib, n int) (int, bool) {
	switch {
	case (ia+1)%n == ib:
		return ia, true
	case (ib+1)%n == ia:
		return ib, true
	}
	return 0, false
}

// assertCurvesCoincide requires two consumers' curves for one shared segment to trace the same point set
// (forward or reversed), or both to be nil. A nil against a non-nil is the build-order dependence this
// gate exists to forbid: the edge catalog would silently keep whichever face was assembled first.
func assertCurvesCoincide(t *testing.T, who string, seg int, mine, theirs geom.Curve3, a, b math.Point3) {
	t.Helper()
	if (mine == nil) != (theirs == nil) {
		t.Errorf("%s: rim segment %d (%v->%v) — one consumer carries a curve and the other nil (%T vs %T); the edge catalog is first-writer-wins, so this is agreement by build order",
			who, seg, a, b, mine, theirs)
		return
	}
	if mine == nil {
		return
	}
	assertCurveSpansSegment(t, who, seg, mine, a, b)
	assertCurveSpansSegment(t, who+"/wall", seg, theirs, a, b)
	dev := stdmath.Max(maxCurveDeviationFrom(mine, theirs), maxCurveDeviationFrom(theirs, mine))
	if dev > rimSubArcAgreeTol {
		t.Errorf("%s: rim segment %d curves disagree by %.3e (tol %.3e)", who, seg, dev, rimSubArcAgreeTol)
	}
}

// assertCurveSpansSegment requires c to START and END on the segment's own two vertices (either
// traversal direction) — the EXTENT half of curve agreement.
//
// maxCurveDeviationFrom is ONE-SIDED: it samples the first curve and measures each sample to the nearest
// point ANYWHERE on the second, so a proper SUB-ARC of the shared segment — right conic, right start,
// stopping halfway — reads 0. Proven by the adversarial review's M9 mutation: a consumer handing a
// half-length sub-arc of a shared segment passed AgreeByValue, the loop-segment gate AND the watertight
// gate, all three silent. Pinning both endpoints makes the extent explicit, and the deviation above is
// now measured in both directions as well.
func assertCurveSpansSegment(t *testing.T, who string, seg int, c geom.Curve3, a, b math.Point3) {
	t.Helper()
	lo, hi := c.Domain()
	p, q := c.PointAt(lo), c.PointAt(hi)
	drift := stdmath.Min(p.DistanceTo(a)+q.DistanceTo(b), p.DistanceTo(b)+q.DistanceTo(a))
	if drift > rimSubArcPinTol {
		t.Errorf("%s: rim segment %d curve runs %v->%v, not between its own vertices %v->%v (endpoint drift %.3e, tol %.3e) — a sub-arc of the shared segment reads ZERO deviation",
			who, seg, p, q, a, b, drift, rimSubArcPinTol)
	}
}

// assertEveryWallRimSegmentCovered requires the notch and the patch between them to present EVERY rim
// segment of the wall's ring exactly once — the station-count half of the invariant. The wall's ring also
// carries its three preserved seam/top segments, which no rim consumer covers, so those are excluded by
// counting only segments whose two endpoints are both rim stations.
func assertEveryWallRimSegmentCovered(t *testing.T, d obstacleDetection, wall filletLoop, covered map[int]int) {
	t.Helper()
	rimSegs := len(d.holeSampled.pts) + 2 // 64 samples + the 2 inserted nodes, as a closed ring
	for seg := range rimSegs {
		if covered[seg] != 1 {
			t.Errorf("wall rim segment %d is presented by %d rim consumers, want exactly 1 (notch or patch) — station counts have drifted apart",
				seg, covered[seg])
		}
	}
}
