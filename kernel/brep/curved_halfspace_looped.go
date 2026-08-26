// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

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
func loopedSplit(f curvedFace, curves []geom.Curve3, plane geom.Plane, n math.Vector3, res geom.Resolution) ([]curvedFace, []loopEdge, error) {
	if !faceBoundaryCrosses(f, plane, n, res) {
		// The imprint curve exists but does not cross ANY of the face's boundary loops — the face lies
		// wholly on one side (e.g. a far clearing plane on a multi-loop annular lid). Keep or drop it whole;
		// this precedes the single-loop requirement so a clearing plane composes over a holed planar face.
		if signedDistance(faceSample(f), plane, n) <= 0 {
			return []curvedFace{f}, nil, nil
		}
		return nil, nil, nil
	}
	if len(f.loops) != 1 {
		return nil, nil, ErrUnsupportedHalfSpace // a CROSSED holes/multi-loop face: a later increment
	}
	segs, crossings := splitLoopByPlane(f.loops[0], plane, n, res)
	if crossings == 0 {
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
	return traceKeptFaces(f, curves, runs, res)
}

// faceBoundaryCrosses reports whether any of the face's boundary loops crosses the cutting plane (an edge
// runs from one side to the other). A face whose whole boundary lies on one side does not genuinely meet
// the plane — the imprint curve passes outside it — so the caller keeps or drops it whole regardless of
// how many loops it has (the multi-loop split is only needed when the plane actually cuts the boundary).
func faceBoundaryCrosses(f curvedFace, plane geom.Plane, n math.Vector3, res geom.Resolution) bool {
	for i := range f.loops {
		if _, crossings := splitLoopByPlane(f.loops[i], plane, n, res); crossings > 0 {
			return true
		}
	}
	return false
}

// traceKeptFaces threads the kept boundary runs into closed loops, each bridged along the imprint, and
// returns one kept sub-face per loop plus all section edges (the reversed bridges, for the lid). A plane
// can split one face into SEVERAL kept pieces — a slab leaves a cylinder band in two strips — so the runs
// are grouped into as many loops as the bridging forms, not assumed to chain into one.
func traceKeptFaces(f curvedFace, curves []geom.Curve3, runs [][]loopEdge, res geom.Resolution) ([]curvedFace, []loopEdge, error) {
	used := make([]bool, len(runs))
	var faces []curvedFace
	var section []loopEdge
	for s := range runs {
		if used[s] {
			continue
		}
		loop, sec, err := traceKeptLoop(f, curves, runs, used, s, res)
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
func traceKeptLoop(f curvedFace, curves []geom.Curve3, runs [][]loopEdge, used []bool, start int, res geom.Resolution) ([]loopEdge, []loopEdge, error) {
	var loop, section []loopEdge
	for j := start; ; {
		used[j] = true
		loop = append(loop, runs[j]...)
		exit := runs[j][len(runs[j])-1].end()
		bridge, k, ok := bridgeToEntry(f, curves, runs, exit, res)
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
func bridgeToEntry(f curvedFace, curves []geom.Curve3, runs [][]loopEdge, exit math.Point3, res geom.Resolution) (loopEdge, int, bool) {
	for k, run := range runs {
		entry := run[0].start()
		if bridge, ok := bridgeArc(f, curves, exit, entry, res); ok {
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
func splitLoopByPlane(loop curvedLoop, plane geom.Plane, n math.Vector3, res geom.Resolution) ([]keptSeg, int) {
	var segs []keptSeg
	crossings := 0
	for _, le := range loop.edges {
		cs := edgeCrossings(le, plane, n, res)
		crossings += len(cs)
		if samePoint(le.start(), le.end(), res) && len(cs) > 0 {
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
// the edge), found by sampling for sign changes then bisecting each. A CLOSED (seam) edge is sampled
// cyclically — see closedEdgeCrossings — so a crossing that lands on its seam vertex is not missed.
func edgeCrossings(le loopEdge, plane geom.Plane, n math.Vector3, res geom.Resolution) []float64 {
	if samePoint(le.start(), le.end(), res) {
		return closedEdgeCrossings(le, plane, n)
	}
	var out []float64
	prevT := le.t0
	prevG := signedDistance(le.curve.PointAt(le.t0), plane, n)
	const samples = 32
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

// closedEdgeCrossings returns the plane crossings of a closed (seam) edge — a full circle/ellipse whose
// start and end vertex coincide. The open sampler starts and ends AT the seam vertex, so a crossing that
// lands exactly there (the cap centre on the plane) sits at g≈0 at both ends and is missed, yielding an
// odd count that defers the split. Sampling at phase-shifted MIDPOINTS keeps every sample off the seam,
// and comparing the samples CYCLICALLY (the last wraps across the seam to the first) catches a crossing
// at the seam like any other. The wrap interval's bisection runs past the period; the result is folded
// back into [t0, t1).
func closedEdgeCrossings(le loopEdge, plane geom.Plane, n math.Vector3) []float64 {
	const samples = 64
	span := le.t1 - le.t0 // signed: negative when the closed edge is traversed reversed (t0 > t1)
	ts := make([]float64, samples)
	gs := make([]float64, samples)
	for i := range samples {
		ts[i] = le.t0 + span*(float64(i)+0.5)/samples
		gs[i] = signedDistance(le.curve.PointAt(ts[i]), plane, n)
	}
	var out []float64
	for i := range samples {
		j := (i + 1) % samples
		if (gs[i] < 0) == (gs[j] < 0) {
			continue
		}
		hi := ts[j]
		if j == 0 {
			hi += span // the last→first pair straddles the seam: extend past t1 (the curve is periodic)
		}
		out = append(out, foldParam(bisectCrossing(le, plane, n, ts[i], hi), le.t0, span))
	}
	sortByTraversal(out, le.t0, span) // closedEdgeSegs needs the crossings in traversal (t0→t1) order
	return out
}

// foldParam folds a parameter (possibly advanced past t1 by the seam-wrap bisection) back into the
// half-open traversal interval [t0, t0+span) — handling either sign of span (a reversed closed edge has
// span < 0).
func foldParam(t, t0, span float64) float64 {
	prog := (t - t0) / span // fraction of the way through the traversal
	prog -= stdmath.Floor(prog)
	return t0 + prog*span
}

// sortByTraversal orders crossings by how far along the edge's traversal (t0→t1) they sit, so they tile
// the closed curve in order regardless of whether span is positive (forward) or negative (reversed).
func sortByTraversal(out []float64, t0, span float64) {
	sort.Slice(out, func(i, j int) bool {
		return (out[i]-t0)/span < (out[j]-t0)/span
	})
}

// bisectCrossing refines a sign-change bracket [ta, tb] to the parameter where g = 0.
func bisectCrossing(le loopEdge, plane geom.Plane, n math.Vector3, ta, tb float64) float64 {
	ga := signedDistance(le.curve.PointAt(ta), plane, n)
	for range 50 {
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
	for i := range n {
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
	for k := range n {
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
func bridgeArc(f curvedFace, curves []geom.Curve3, exit, entry math.Point3, res geom.Resolution) (loopEdge, bool) {
	for _, c := range curves {
		if e, ok := bridgeAlong(f, c, exit, entry, res); ok {
			return e, true
		}
	}
	return loopEdge{}, false
}

// bridgeAlong returns the exit→entry portion of a single imprint curve c, or ok=false if c does not pass
// through both endpoints (so the wrong line of a pair is rejected) or no arc of it lies inside f.
func bridgeAlong(f curvedFace, c geom.Curve3, exit, entry math.Point3, res geom.Resolution) (loopEdge, bool) {
	tExit, _ := geom.CurveParamAtPoint3(c, exit)
	tEntry, _ := geom.CurveParamAtPoint3(c, entry)
	if !onCurve(c, tExit, exit, res) || !onCurve(c, tEntry, entry, res) {
		return loopEdge{}, false
	}
	if _, isLine := c.(geom.Line); isLine {
		return loopEdge{curve: c, t0: tExit, t1: tEntry}, true
	}
	switch c.(type) {
	case geom.Hyperbola, geom.Parabola:
		return loopEdge{curve: c, t0: tExit, t1: tEntry}, true // an open conic arm is simple: bridge by its param range
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
func onCurve(c geom.Curve3, t float64, p math.Point3, res geom.Resolution) bool {
	return samePoint(c.PointAt(t), p, res)
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
