// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Looped split (M2 Phase 1, Oblikovati/Oblikovati#1334). Splits a face the cutting plane CROSSES — the
// step that, composed over a tool's planes, cuts a curved solid by a box. For each boundary loop it
// finds where the loop crosses g=0 (the imprint curve on the surface), keeps the runs on the negative
// side, and bridges each run's exit back to its entry along the part of the imprint that lies inside the
// face (pointInCurvedFace picks which arc). The bridge arcs, reversed, are the section edges that chain
// into the lid. Scope: a single imprint conic and loops with an even number of simple crossings (the box
// case); an imprint that closes inside the face (an island cut) defers to ErrUnsupportedHalfSpace.

// keptSeg is one sub-edge of a boundary loop after splitting at the plane crossings, tagged by whether
// its interior lies on the kept (negative) side.
type keptSeg struct {
	edge loopEdge
	keep bool
}

// loopedSplit splits a looped face f by the plane along imprint conic c, returning the kept sub-faces
// and the section arcs (oriented for the lid). It handles f's outer loop crossed at an even number of
// points with no holes; anything else defers.
func loopedSplit(f curvedFace, c geom.Curve3, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	if len(f.loops) != 1 {
		return nil, nil, ErrUnsupportedHalfSpace // holes/multi-loop: a later increment
	}
	segs, crossings := splitLoopByPlane(f.loops[0], plane, n)
	if crossings == 0 || crossings%2 != 0 {
		return nil, nil, ErrUnsupportedHalfSpace // island cut or a tangency we don't resolve yet
	}
	runs := keptRuns(segs)
	if len(runs) == 0 {
		return nil, nil, nil // every crossing-free piece was on the positive side: dropped
	}
	loop, section, err := closeRuns(f, c, runs)
	if err != nil {
		return nil, nil, err
	}
	kept := curvedFace{surface: f.surface, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}}
	return []curvedFace{kept}, section, nil
}

// closeRuns bridges each kept run's exit back to the next run's entry along the imprint arc inside f,
// producing one closed kept loop and the matching (reversed) section arcs. It assumes the runs, taken in
// order, alternate with the imprint bridges (the even-crossing convex case).
func closeRuns(f curvedFace, c geom.Curve3, runs [][]loopEdge) ([]loopEdge, []loopEdge, error) {
	var loop, section []loopEdge
	for i, run := range runs {
		loop = append(loop, run...)
		exit := run[len(run)-1].end()
		entry := runs[(i+1)%len(runs)][0].start()
		bridge, ok := bridgeArc(f, c, exit, entry)
		if !ok {
			return nil, nil, ErrUnsupportedHalfSpace
		}
		loop = append(loop, bridge)
		section = append(section, reverseEdge(bridge)) // the lid traverses the cut the opposite way
	}
	return loop, section, nil
}

// splitLoopByPlane cuts every loop edge at its g=0 crossings, returning the resulting sub-edges tagged
// kept/dropped (by their midpoint side) and the total crossing count.
func splitLoopByPlane(loop curvedLoop, plane geom.Plane, n math.Vector3) ([]keptSeg, int) {
	var segs []keptSeg
	crossings := 0
	for _, le := range loop.edges {
		cs := edgeCrossings(le, plane, n)
		crossings += len(cs)
		bounds := append([]float64{le.t0}, cs...)
		bounds = append(bounds, le.t1)
		for i := 0; i+1 < len(bounds); i++ {
			sub := loopEdge{curve: le.curve, t0: bounds[i], t1: bounds[i+1]}
			mid := le.curve.PointAt((bounds[i] + bounds[i+1]) / 2)
			segs = append(segs, keptSeg{edge: sub, keep: signedDistance(mid, plane, n) < 0})
		}
	}
	return segs, crossings
}

// edgeCrossings returns the parameters within an edge's [t0, t1] where g crosses zero (the plane cuts
// the edge), found by sampling for sign changes then bisecting each.
func edgeCrossings(le loopEdge, plane geom.Plane, n math.Vector3) []float64 {
	const samples = 32
	var out []float64
	prevT := le.t0
	prevG := signedDistance(le.curve.PointAt(le.t0), plane, n)
	for i := 1; i <= samples; i++ {
		t := le.t0 + (le.t1-le.t0)*float64(i)/samples
		g := signedDistance(le.curve.PointAt(t), plane, n)
		if (prevG < 0) != (g < 0) {
			out = append(out, bisectCrossing(le, plane, n, prevT, t))
		}
		prevT, prevG = t, g
	}
	return out
}

// bisectCrossing refines a sign-change bracket [ta, tb] to the parameter where g = 0.
func bisectCrossing(le loopEdge, plane geom.Plane, n math.Vector3, ta, tb float64) float64 {
	ga := signedDistance(le.curve.PointAt(ta), plane, n)
	for i := 0; i < 50; i++ {
		tm := (ta + tb) / 2
		gm := signedDistance(le.curve.PointAt(tm), plane, n)
		if (ga < 0) == (gm < 0) {
			ta, ga = tm, gm
		} else {
			tb = tm
		}
	}
	return (ta + tb) / 2
}

// keptRuns extracts the maximal cyclic runs of kept sub-edges. An all-kept loop is one run; an all-
// dropped loop none.
func keptRuns(segs []keptSeg) [][]loopEdge {
	n := len(segs)
	start := -1
	for i := 0; i < n; i++ {
		if segs[i].keep && !segs[(i+n-1)%n].keep {
			start = i
			break
		}
	}
	if start == -1 {
		if n > 0 && segs[0].keep {
			return [][]loopEdge{allEdges(segs)}
		}
		return nil
	}
	return collectRuns(segs, start)
}

// collectRuns walks the segments from start, gathering each maximal kept run.
func collectRuns(segs []keptSeg, start int) [][]loopEdge {
	n := len(segs)
	var runs [][]loopEdge
	var cur []loopEdge
	for k := 0; k < n; k++ {
		s := segs[(start+k)%n]
		if s.keep {
			cur = append(cur, s.edge)
		} else if len(cur) > 0 {
			runs = append(runs, cur)
			cur = nil
		}
	}
	if len(cur) > 0 {
		runs = append(runs, cur)
	}
	return runs
}

func allEdges(segs []keptSeg) []loopEdge {
	out := make([]loopEdge, len(segs))
	for i, s := range segs {
		out[i] = s.edge
	}
	return out
}

// bridgeArc returns the portion of the imprint curve c from exit to entry that lies inside face f. For a
// line it is the segment between them; for a closed conic it is whichever of the two arcs has its
// midpoint inside f.
func bridgeArc(f curvedFace, c geom.Curve3, exit, entry math.Point3) (loopEdge, bool) {
	tExit, _ := geom.CurveParamAtPoint3(c, exit)
	tEntry, _ := geom.CurveParamAtPoint3(c, entry)
	if _, isLine := c.(geom.Line); isLine {
		return loopEdge{curve: c, t0: tExit, t1: tEntry}, true
	}
	for _, t1 := range arcCandidates(tExit, tEntry) {
		cand := loopEdge{curve: c, t0: tExit, t1: t1}
		if pointInCurvedFace(f, c.PointAt((tExit+t1)/2)) {
			return cand, true
		}
	}
	return loopEdge{}, false
}

// arcCandidates returns the two end parameters that, paired with tExit, sweep the two arcs (positive and
// negative wrap) of a periodic curve from tExit to tEntry.
func arcCandidates(tExit, tEntry float64) []float64 {
	d := stdmath.Mod(tEntry-tExit, 1)
	if d < 0 {
		d += 1
	}
	return []float64{tExit + d, tExit + d - 1}
}

// reverseEdge flips a loop edge's traversal (swaps its parameter endpoints).
func reverseEdge(e loopEdge) loopEdge {
	return loopEdge{curve: e.curve, t0: e.t1, t1: e.t0}
}
