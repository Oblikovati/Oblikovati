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

// reverseEdge flips a loop edge's traversal (swaps its parameter endpoints).
func reverseEdge(e loopEdge) loopEdge {
	return loopEdge{curve: e.curve, t0: e.t1, t1: e.t0}
}
