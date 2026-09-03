// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// splitLoopsAtPoints cuts every loop edge at each of the points lying ON it, strictly inside its span
// (ADR-0060). A trim chart re-emits a boundary run as one edge even where an imprint met it and then
// dissolved (both cells kept); the neighbouring face built from the same interaction keeps that point
// as a vertex, and a shared edge must subdivide identically on both faces — the same discipline the
// planar stitch's T-junction pass applies to polygonal rings.
func splitLoopsAtPoints(loops []curvedLoop, pts []math.Point3, res geom.Resolution) []curvedLoop {
	if len(pts) == 0 {
		return loops
	}
	out := make([]curvedLoop, len(loops))
	for li, l := range loops {
		var edges []loopEdge
		for _, e := range l.edges {
			edges = append(edges, splitEdgeAtPoints(e, pts, res)...)
		}
		out[li] = curvedLoop{edges: edges}
	}
	return out
}

// splitEdgeAtPoints returns the edge cut at the given points that lie on it inside its span, in
// traversal order; the edge itself when none do.
func splitEdgeAtPoints(e loopEdge, pts []math.Point3, res geom.Resolution) []loopEdge {
	var cuts []float64
	for _, p := range pts {
		if t, on := edgeParamOn(e, p, res); on {
			cuts = append(cuts, t)
		}
	}
	if len(cuts) == 0 {
		return []loopEdge{e}
	}
	sort.Float64s(cuts)
	if e.t0 > e.t1 {
		for i, j := 0, len(cuts)-1; i < j; i, j = i+1, j-1 {
			cuts[i], cuts[j] = cuts[j], cuts[i]
		}
	}
	params := append(append([]float64{e.t0}, cuts...), e.t1)
	pieces := make([]loopEdge, 0, len(params)-1)
	for i := 1; i < len(params); i++ {
		if params[i] != params[i-1] {
			pieces = append(pieces, loopEdge{curve: e.curve, t0: params[i-1], t1: params[i]})
		}
	}
	return pieces
}

// edgeParamOn inverts the edge's curve at p and accepts the parameter when p is on the curve within the
// weld and strictly inside the span (in whichever period branch of a closed curve lands there).
func edgeParamOn(e loopEdge, p math.Point3, res geom.Resolution) (float64, bool) {
	t, ok := geom.CurveParamAt(e.curve, p)
	if !ok {
		return 0, false
	}
	lo, hi := stdmath.Min(e.t0, e.t1), stdmath.Max(e.t0, e.t1)
	branches := []float64{t}
	if geom.CurveIsClosed(e.curve) {
		dlo, dhi := e.curve.Domain()
		branches = append(branches, t-(dhi-dlo), t+(dhi-dlo))
	}
	for _, b := range branches {
		if b <= lo+paramSplitSlack || b >= hi-paramSplitSlack {
			continue
		}
		if float64(e.curve.PointAt(b).DistanceTo(p)) <= res.Weld() {
			return b, true
		}
	}
	return 0, false
}

// paramSplitSlack keeps a cut off an edge's own ends: an incidence AT an end is already a vertex.
const paramSplitSlack = 1e-9 // tol:parametric — curve parameter slack against the edge ends
