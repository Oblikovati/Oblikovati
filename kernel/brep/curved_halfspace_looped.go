// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Looped split (M2 Phase 1, Oblikovati/Oblikovati#1334). Splits a face the cutting plane CROSSES — the
// step that, composed over a tool's planes, cuts a curved solid by a box. It finds where the boundary
// crosses g=0 (the imprint on the surface), keeps the runs on the negative side, and threads them into
// closed loops by bridging each run's exit along whichever imprint curve reaches the next run's entry.
// The imprint may be one conic (a sphere cap, a disk lid) or several lines (a cylinder arc band a box
// re-cuts), and a single plane may leave the face in SEVERAL kept pieces (a slab leaves a cylinder band
// in two strips) — so it yields one sub-face per traced loop. The bridges, reversed, are the section
// edges that chain into the lid. Scope: an even number of simple crossings and a single outer loop; a
// hole, an odd crossing (a tangency/island), or an unbridgeable run defers to ErrUnsupportedHalfSpace.

// keptSeg is one sub-edge of a boundary loop after splitting at the plane crossings, tagged by whether
// its interior lies on the kept (negative) side.
type keptSeg struct {
	edge loopEdge
	keep bool
}

// loopedSplit splits a looped face f by the plane along its imprint curves, returning the kept sub-faces
// and the section arcs (oriented for the lid). It handles f's outer loop crossed at an even number of
// points with no holes; the imprint may be one conic (a sphere cap, a disk lid) or several lines (a
// cylinder arc band a box re-cuts), each kept run bridged along whichever imprint curve joins it. A face
// with holes, an odd crossing count (a tangency/island), or an unbridgeable run defers.
func loopedSplit(f curvedFace, curves []geom.Curve3, plane geom.Plane, n math.Vector3) ([]curvedFace, []loopEdge, error) {
	if len(f.loops) != 1 {
		return nil, nil, ErrUnsupportedHalfSpace // holes/multi-loop: a later increment
	}
	segs, crossings := splitLoopByPlane(f.loops[0], plane, n)
	if crossings == 0 {
		// An imprint curve exists but does not cross this face's boundary — the face lies wholly on one
		// side (e.g. a planar lid whose plane meets the cut plane in a line far outside it).
		if signedDistance(faceSample(f), plane, n) <= 0 {
			return []curvedFace{f}, nil, nil
		}
		return nil, nil, nil
	}
	if crossings%2 != 0 {
		return nil, nil, ErrUnsupportedHalfSpace // a tangency / island we don't resolve yet
	}
	runs := keptRuns(segs)
	if len(runs) == 0 {
		return nil, nil, nil // every crossing-free piece was on the positive side: dropped
	}
	return traceKeptFaces(f, curves, runs)
}

// traceKeptFaces threads the kept boundary runs into closed loops, each bridged along the imprint, and
// returns one kept sub-face per loop plus all section edges (the reversed bridges, for the lid). A plane
// can split one face into SEVERAL kept pieces — a slab leaves a cylinder band in two strips — so the runs
// are grouped into as many loops as the bridging forms, not assumed to chain into one.
func traceKeptFaces(f curvedFace, curves []geom.Curve3, runs [][]loopEdge) ([]curvedFace, []loopEdge, error) {
	used := make([]bool, len(runs))
	var faces []curvedFace
	var section []loopEdge
	for s := range runs {
		if used[s] {
			continue
		}
		loop, sec, err := traceKeptLoop(f, curves, runs, used, s)
		if err != nil {
			return nil, nil, err
		}
		faces = append(faces, curvedFace{surface: f.surface, reversed: f.reversed, lineage: f.lineage, loops: []curvedLoop{{edges: loop}}})
		section = append(section, sec...)
	}
	return faces, section, nil
}

// traceKeptLoop walks from the start run, alternating boundary runs with imprint bridges, until it
// returns to the start — one closed kept loop. Each run's exit bridges along the imprint curve to the
// crossing where the next (or the same) run begins.
func traceKeptLoop(f curvedFace, curves []geom.Curve3, runs [][]loopEdge, used []bool, start int) ([]loopEdge, []loopEdge, error) {
	var loop, section []loopEdge
	for j := start; ; {
		used[j] = true
		loop = append(loop, runs[j]...)
		exit := runs[j][len(runs[j])-1].end()
		bridge, k, ok := bridgeToEntry(f, curves, runs, exit)
		if !ok {
			return nil, nil, ErrUnsupportedHalfSpace
		}
		loop = append(loop, bridge)
		section = append(section, reverseEdge(bridge)) // the lid traverses the cut the opposite way
		if k == start {
			return loop, section, nil
		}
		if used[k] {
			return nil, nil, ErrUnsupportedHalfSpace // a bridge re-entered a consumed run: an arrangement we defer
		}
		j = k
	}
}

// bridgeToEntry finds the run whose start the exit point bridges to along an imprint curve, returning the
// bridge edge and that run's index. The imprint curve must pass through BOTH the exit and the candidate
// entry, so the matching line of an axis-parallel pair selects the run that continues the same strip.
func bridgeToEntry(f curvedFace, curves []geom.Curve3, runs [][]loopEdge, exit math.Point3) (loopEdge, int, bool) {
	for k, run := range runs {
		entry := run[0].start()
		if bridge, ok := bridgeArc(f, curves, exit, entry); ok {
			return bridge, k, true
		}
	}
	return loopEdge{}, -1, false
}

// splitLoopByPlane cuts every loop edge at its g=0 crossings, returning the resulting sub-edges tagged
// kept/dropped (by their midpoint side) and the total crossing count. A closed edge (a full seam circle)
// is cut at its crossings ONLY, the arcs running between consecutive crossings across the seam — never
// split at the arbitrary seam vertex, which would fragment a kept arc that wraps it into two edges that
// then fail to weld with the matching single arc on the adjoining face.
func splitLoopByPlane(loop curvedLoop, plane geom.Plane, n math.Vector3) ([]keptSeg, int) {
	var segs []keptSeg
	crossings := 0
	for _, le := range loop.edges {
		cs := edgeCrossings(le, plane, n)
		crossings += len(cs)
		if samePoint(le.start(), le.end()) && len(cs) > 0 {
			segs = append(segs, closedEdgeSegs(le, cs, plane, n)...)
		} else {
			segs = append(segs, openEdgeSegs(le, cs, plane, n)...)
		}
	}
	return segs, crossings
}

// openEdgeSegs splits an open edge at its crossings into [t0, c…, t1] sub-edges, tagged by midpoint side.
func openEdgeSegs(le loopEdge, cs []float64, plane geom.Plane, n math.Vector3) []keptSeg {
	bounds := append([]float64{le.t0}, cs...)
	bounds = append(bounds, le.t1)
	out := make([]keptSeg, 0, len(bounds)-1)
	for i := 0; i+1 < len(bounds); i++ {
		out = append(out, taggedSeg(le.curve, bounds[i], bounds[i+1], plane, n))
	}
	return out
}

// closedEdgeSegs splits a closed (seam) edge into the arcs between consecutive crossings, the last
// wrapping across the seam (its end parameter carries +period so the arc spans the seam rather than
// stopping at it). period is the signed full-domain span le walks (±1 by traversal direction).
func closedEdgeSegs(le loopEdge, cs []float64, plane geom.Plane, n math.Vector3) []keptSeg {
	period := le.t1 - le.t0
	out := make([]keptSeg, 0, len(cs))
	for i := range cs {
		a, b := cs[i], cs[(i+1)%len(cs)]
		if i == len(cs)-1 {
			b += period // the final arc runs from the last crossing across the seam to the first
		}
		out = append(out, taggedSeg(le.curve, a, b, plane, n))
	}
	return out
}

// taggedSeg builds a sub-edge over [a, b] of curve c and tags it kept when its midpoint is on the plane's
// negative side.
func taggedSeg(c geom.Curve3, a, b float64, plane geom.Plane, n math.Vector3) keptSeg {
	mid := c.PointAt((a + b) / 2)
	return keptSeg{edge: loopEdge{curve: c, t0: a, t1: b}, keep: signedDistance(mid, plane, n) < 0}
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

// bridgeArc returns the portion of one imprint curve from exit to entry that lies inside face f, trying
// each curve until one passes through both points (the right line of an axis-parallel pair, or the
// imprint conic). A line gives the segment between them; a closed conic gives whichever of its two arcs
// has its midpoint inside f.
func bridgeArc(f curvedFace, curves []geom.Curve3, exit, entry math.Point3) (loopEdge, bool) {
	for _, c := range curves {
		if e, ok := bridgeAlong(f, c, exit, entry); ok {
			return e, true
		}
	}
	return loopEdge{}, false
}

// bridgeAlong returns the exit→entry portion of a single imprint curve c, or ok=false if c does not pass
// through both endpoints (so the wrong line of a pair is rejected) or no arc of it lies inside f.
func bridgeAlong(f curvedFace, c geom.Curve3, exit, entry math.Point3) (loopEdge, bool) {
	tExit, _ := geom.CurveParamAtPoint3(c, exit)
	tEntry, _ := geom.CurveParamAtPoint3(c, entry)
	if !onCurve(c, tExit, exit) || !onCurve(c, tEntry, entry) {
		return loopEdge{}, false
	}
	if _, isLine := c.(geom.Line); isLine {
		return loopEdge{curve: c, t0: tExit, t1: tEntry}, true
	}
	if _, isHyp := c.(geom.Hyperbola); isHyp {
		return loopEdge{curve: c, t0: tExit, t1: tEntry}, true // a hyperbola branch is simple: bridge by θ range
	}
	for _, t1 := range arcCandidates(tExit, tEntry) {
		cand := loopEdge{curve: c, t0: tExit, t1: t1}
		if pointInCurvedFace(f, c.PointAt((tExit+t1)/2)) {
			return cand, true
		}
	}
	return loopEdge{}, false
}

// onCurve reports whether c.PointAt(t) coincides with p — used to confirm a candidate imprint curve
// actually passes through a crossing point before bridging along it.
func onCurve(c geom.Curve3, t float64, p math.Point3) bool {
	return samePoint(c.PointAt(t), p)
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
