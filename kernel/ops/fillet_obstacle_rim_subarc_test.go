// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// rimSubArcOnRimTol is the ABSOLUTE ceiling on how far a node-adjacent rim segment's curve may sit off
// the exact rim: the scale of the rim's OWN 1/64 per-segment discretization. It exists only to separate a
// sub-arc from a CHORD — the straight chords this replaced measure 7.0e-03 … 3.0e-02, at least 70x above
// it. The real, non-tunable bar is the comparative one below (no worse than the rim's own interior
// segments), which is what a self-calibrating gate needs; this constant cannot be loosened into a pass.
const rimSubArcOnRimTol = 1e-4

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
			d := singleHostDetection(t, c)
			rim := d.holeEdge.Geometry()
			wall := insertNodesIntoRim(d)
			nodeSegs := nodeAdjacentSegments(t, d, wall)
			bar := worstInteriorRimArcDeviation(wall, rim, nodeSegs) + rimSubArcSlack
			for _, seg := range nodeSegs {
				assertSegmentIsRimSubArc(t, wall, seg, rim, bar)
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
//   - CURVE AGREEMENT: for each matched pair the two consumers' curves must trace the same point set,
//     forward or reversed, to rimSubArcAgreeTol.
func TestObstacleRimConsumersAgreeByValue(t *testing.T) {
	for _, c := range singleHostObstacleCorpus {
		t.Run(c.name, func(t *testing.T) {
			d := singleHostDetection(t, c)
			_, of, og, _ := obstacleFeatureFor(t, importCorpusBody(t, c.step), c.name, c.mid, c.radius)
			wall := insertNodesIntoRim(d)
			notch := hostSideSubArc(d.holeSampled, d.nodes, d.back, d.rimTrims)
			patch := patchBoundaryLoop(d, og, of)
			covered := map[int]int{}
			matchRimSegments(t, "notch", notch, len(notch.pts)-1, wall, covered)
			matchRimSegments(t, "patch", patch, len(patch.pts), wall, covered)
			assertEveryWallRimSegmentCovered(t, d, wall, covered)
		})
	}
}

// singleHostDetection runs the real pipeline up to detectObstacle for one corpus obstacle case.
func singleHostDetection(t *testing.T, c singleHostObstacleCase) obstacleDetection {
	t.Helper()
	body := importCorpusBody(t, c.step)
	ef, res := singleEdgeFillet(t, body, c.name, c.mid, c.radius)
	d, ok := detectObstacle(ef, res)
	if !ok {
		t.Fatalf("%s: detectObstacle found no single-host obstacle", c.name)
	}
	return d
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

// assertSegmentIsRimSubArc checks one segment's curve: present, pinned to its own two vertices, and on
// the rim curve.
func assertSegmentIsRimSubArc(t *testing.T, loop filletLoop, seg int, rim geom.Curve3, bar float64) {
	t.Helper()
	c := curveAt(loop.curves, seg)
	if c == nil {
		t.Errorf("rim segment %d carries NO curve — the straight truncated node chord is back", seg)
		return
	}
	a, b := loop.pts[seg], loop.pts[(seg+1)%len(loop.pts)]
	lo, hi := c.Domain()
	if da, db := c.PointAt(lo).DistanceTo(a), c.PointAt(hi).DistanceTo(b); da > 1e-12 || db > 1e-12 {
		t.Errorf("rim segment %d sub-arc endpoints drift %.3e / %.3e from its own vertices", seg, da, db)
	}
	dev := maxCurveDeviationFrom(c, rim)
	if dev > rimSubArcOnRimTol {
		t.Errorf("rim segment %d curve leaves the exact rim by %.3e — that is chord-scale, not sub-arc scale (ceiling %.3e)", seg, dev, rimSubArcOnRimTol)
	}
	if dev > bar {
		t.Errorf("rim segment %d trimmed sub-arc is %.3e off the rim, WORSE than the rim's own interior per-segment arcs (bar %.3e)", seg, dev, bar)
	}
}

// worstInteriorRimArcDeviation is the largest off-rim deviation among the rim's UNTOUCHED interior
// per-segment arcs (rimSegmentArc over a full 1/64 span) — the discretization the node sub-arcs belong to,
// and therefore the bar they must not be worse than. It is measured from the same loop on the same rim, so
// the bar calibrates itself per case: ~1.3e-14 on a circular rim, 1.2e-04 on T6's ellipse.
func worstInteriorRimArcDeviation(wall filletLoop, rim geom.Curve3, exclude []int) float64 {
	skip := map[int]bool{}
	for _, s := range exclude {
		skip[s] = true
	}
	worst := 0.0
	for seg := range wall.curves {
		if skip[seg] || curveAt(wall.curves, seg) == nil {
			continue
		}
		if d := maxCurveDeviationFrom(wall.curves[seg], rim); d > worst {
			worst = d
		}
	}
	return worst
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
	for k := 0; k < 60; k++ {
		step /= 2
		for _, t := range []float64{bt - step, bt + step} {
			if d := p.DistanceTo(ref.PointAt(t)); d < best {
				best, bt = d, t
			}
		}
	}
	return best
}

// matchRimSegments requires every one of loop's first segCount consecutive point pairs that lies on the
// rim (matched by exact point value against wall's ring) to be a segment of that ring, and records which
// wall segment each covered. A pair whose two points are not adjacent on the wall ring is a T-junction.
func matchRimSegments(t *testing.T, who string, loop filletLoop, segCount int, wall filletLoop, covered map[int]int) {
	t.Helper()
	index := map[math.Point3]int{}
	for i, p := range wall.pts {
		index[p] = i
	}
	for i := 0; i < segCount; i++ {
		a, b := loop.pts[i], loop.pts[(i+1)%len(loop.pts)]
		ia, oka := index[a]
		ib, okb := index[b]
		if !oka || !okb {
			continue // not a rim segment (a wall-front or wing-arc side of the patch loop)
		}
		seg, ok := wallSegmentBetween(ia, ib, len(wall.pts))
		if !ok {
			t.Errorf("%s: rim pair %v->%v is not a segment of the wall's rim ring (indices %d,%d) — T-junction", who, a, b, ia, ib)
			continue
		}
		covered[seg]++
		assertCurvesCoincide(t, who, seg, curveAt(loop.curves, i), curveAt(wall.curves, seg), a, b)
	}
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
	dev := maxCurveDeviationFrom(mine, theirs)
	if dev > rimSubArcAgreeTol {
		t.Errorf("%s: rim segment %d curves disagree by %.3e (tol %.3e)", who, seg, dev, rimSubArcAgreeTol)
	}
}

// assertEveryWallRimSegmentCovered requires the notch and the patch between them to present EVERY rim
// segment of the wall's ring exactly once — the station-count half of the invariant. The wall's ring also
// carries its three preserved seam/top segments, which no rim consumer covers, so those are excluded by
// counting only segments whose two endpoints are both rim stations.
func assertEveryWallRimSegmentCovered(t *testing.T, d obstacleDetection, wall filletLoop, covered map[int]int) {
	t.Helper()
	rimSegs := len(d.holeSampled.pts) + 2 // 64 samples + the 2 inserted nodes, as a closed ring
	for seg := 0; seg < rimSegs; seg++ {
		if covered[seg] != 1 {
			t.Errorf("wall rim segment %d is presented by %d rim consumers, want exactly 1 (notch or patch) — station counts have drifted apart",
				seg, covered[seg])
		}
	}
}
