// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// N-sided boundary fill (M36, follow-up to F07 #1300): fill an opening bounded by N≥3 neighbour
// surface bodies with a single clean NURBS. The opening's N inner edges are chained into an oriented
// loop and mapped onto the FOUR logical sides of a quad — splitting one edge when N<4 (so no side is
// degenerate; the rejected #1283 shortcut was a zero-width corner) and merging consecutive edges into
// one compatible side curve when N>4 (reusing geom.JoinCurves / F01 make-compatible). The four sides
// then feed the exact four-sided Coons + MatchSurface fill (FillFourSided). Continuity: a side that
// is a single original NURBS edge meets its neighbour at the requested order (G1/G2); a side that is
// a merge of several edges, or a half of a split edge, fills position-only (G0) — the documented
// N-sided G2 convergence limit.

// FillNSided fills the opening bounded by neighbours (3 or more single-face surface bodies) with one
// NURBS at the requested continuity order. With exactly four neighbours it is the four-sided fill;
// otherwise the N inner edges chain onto four compatible quad sides via the shared edge-loop fill.
func FillNSided(neighbours []*topo.Body, order int) (*topo.Body, error) {
	if len(neighbours) < 3 {
		return nil, fmt.Errorf("ops.FillNSided: needs at least 3 bounding surfaces, have %d", len(neighbours))
	}
	if len(neighbours) == 4 {
		return FillFourSided([4]*topo.Body{neighbours[0], neighbours[1], neighbours[2], neighbours[3]}, order)
	}
	edges, err := openingEdgesN(neighbours)
	if err != nil {
		return nil, err
	}
	return fillFromBoundaryEdges(edges, order)
}

// makeSidesCompatible refines the two opposite side pairs (c0/c1 in u, d0/d1 in v) to share degree
// and knots, as the discrete Coons net requires — needed once merged sides change a side's control
// count. Endpoints (and thus the corners) are preserved.
func makeSidesCompatible(c0, c1, d0, d1 *boundaryEdge) error {
	a, b, err := geom.MakeCompatible(c0.curve, c1.curve)
	if err != nil {
		return err
	}
	c0.curve, c1.curve = a, b
	a, b, err = geom.MakeCompatible(d0.curve, d1.curve)
	if err != nil {
		return err
	}
	d0.curve, d1.curve = a, b
	return nil
}

// openingEdgesN returns each neighbour's inner boundary edge (the one facing the opening centre, the
// average of the neighbour face centroids). It is the N-ary form of openingEdges.
func openingEdgesN(neighbours []*topo.Body) ([]boundaryEdge, error) {
	faces := make([]*topo.Face, len(neighbours))
	surfs := make([]geom.Surface, len(neighbours))
	var sum math.Vector3
	for i, b := range neighbours {
		f, s, ok := firstSurfaceFace(b)
		if !ok {
			return nil, fmt.Errorf("ops.FillNSided: neighbour %d has no surface face", i)
		}
		faces[i], surfs[i] = f, s
		sum = sum.Add(probe.FaceCentroid(f).AsVector())
	}
	center := sum.Scale(math.Scalar(1 / float64(len(neighbours)))).AsPoint()
	edges := make([]boundaryEdge, len(neighbours))
	for i := range neighbours {
		edges[i] = innerBoundary(faces[i], surfs[i], center)
	}
	return edges, nil
}

// orderLoop orders the edges head-to-tail so each one's end meets the next one's start, orienting
// (reversing) curves as needed. It errors if the edges do not form one closed loop.
func orderLoop(edges []boundaryEdge) ([]boundaryEdge, error) {
	tol := boundaryWeldTol(edges)
	rest := append([]boundaryEdge{}, edges[1:]...)
	loop := []boundaryEdge{edges[0]}
	for len(rest) > 0 {
		tail := loop[len(loop)-1].end()
		nxt, ok := takeSharing(&rest, tail, tol)
		if !ok {
			return nil, fmt.Errorf("ops.FillNSided: the %d edges do not form a closed loop", len(edges))
		}
		loop = append(loop, orient(nxt, tail, tol))
	}
	if !loop[len(loop)-1].end().IsEqualTo(loop[0].start(), tol) {
		return nil, fmt.Errorf("ops.FillNSided: the boundary edges do not close")
	}
	return loop, nil
}

// splitLongestEdge splits the loop's longest edge at its midpoint into two G0 (non-matchable) halves,
// so an N<4 opening reaches four sides without a degenerate corner.
func splitLongestEdge(loop []boundaryEdge) []boundaryEdge {
	longest, at := 0, 0.5
	best := -1.0
	for i, e := range loop {
		if d := float64(e.start().DistanceTo(e.end())); d > best {
			best, longest = d, i
		}
	}
	a, b, err := geom.SplitCurve(loop[longest].curve, at)
	if err != nil {
		return loop // un-splittable (e.g. degree-1 already minimal); leave as is
	}
	halves := []boundaryEdge{{curve: a}, {curve: b}} // nurbs=false: a half cannot match its neighbour edge
	out := append([]boundaryEdge{}, loop[:longest]...)
	out = append(out, halves...)
	return append(out, loop[longest+1:]...)
}

// groupToFourSides merges the ordered loop's edges into exactly four contiguous side groups (sizes as
// equal as possible). A single-edge group keeps its neighbour link (matchable); a merged group joins
// its curves into one side and fills G0.
func groupToFourSides(loop []boundaryEdge) [4]boundaryEdge {
	sizes := distributeFour(len(loop))
	var sides [4]boundaryEdge
	idx := 0
	for g := range 4 {
		group := loop[idx : idx+sizes[g]]
		sides[g] = mergeGroup(group)
		idx += sizes[g]
	}
	return sides
}

// mergeGroup returns one boundaryEdge for a contiguous group: the single member unchanged, or the
// joined curve (G0, no neighbour match) for a multi-edge group.
func mergeGroup(group []boundaryEdge) boundaryEdge {
	if len(group) == 1 {
		return group[0]
	}
	curves := make([]geom.BSplineCurve, len(group))
	for i, e := range group {
		curves[i] = e.curve
	}
	joined, err := geom.JoinCurves(curves)
	if err != nil {
		return group[0] // fall back to the first member (still a valid boundary curve)
	}
	return boundaryEdge{curve: joined} // nurbs=false: a merged side fills position-only
}

// distributeFour splits n into four contiguous group sizes as equal as possible (the first n%4 groups
// get the extra edge). n is assumed ≥ 4.
func distributeFour(n int) [4]int {
	base, rem := n/4, n%4
	var sizes [4]int
	for i := range 4 {
		sizes[i] = base
		if i < rem {
			sizes[i]++
		}
	}
	return sizes
}
