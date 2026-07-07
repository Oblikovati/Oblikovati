// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DefaultTangentChainAngle is the angular band (radians, ~1°) within which two edges meeting
// at a vertex are treated as G1-tangent for chain propagation. A real crease (a 90° box edge
// sits π/2 from the antiparallel threshold) is far outside it, so the walk stops there; a
// clean line↔arc junction on a rounded rim matches to machine precision and passes.
const DefaultTangentChainAngle = stdmath.Pi / 180

// TangentEdgeChain expands a single seed edge into the maximal run of tangent-continuous edges —
// the "select tangent chain / loop" selection behind fillet/chamfer (Oblikovati/Oblikovati#1798).
// From the seed it walks both endpoint vertices, at each step onto the one neighbour edge that
// (a) is itself a fillet-able crease of the SAME convexity as the seed, (b) meets the current
// edge G1-tangentially at the shared vertex (their away-from-vertex tangents are antiparallel
// within angTol), and (c) is the only other crease at that vertex — every other incident edge is
// a smooth (tangent) edge, so the two supporting faces continue across the junction. This mirrors
// OCCT ChFi3d_Builder::PerformElement + ChFi3d_Builder::FaceTangency ("opposing faces must be
// tangent"). It returns the ordered edge keys (seed included) and whether the run closes a loop.
//
// Example: one click on a straight edge of a rounded-rectangle rim yields all 8 edges
// (4 lines + 4 G1 corner arcs) with closed=true.
func TangentEdgeChain(body *topo.Body, seedKey []byte, angTol float64) ([][]byte, bool, error) {
	seed, ok := body.FindEdgeByKey(seedKey)
	if !ok {
		return nil, false, fmt.Errorf("tangent chain: seed edge key not found in body of %d edges", len(body.Edges()))
	}
	conv := ClassifyEdgeConvexity(seed)
	if conv != EdgeConvex && conv != EdgeConcave {
		return [][]byte{seedKey}, false, nil // a smooth/boundary seed cannot anchor a fillet chain
	}
	if seed.StartVertex() == seed.EndVertex() {
		return [][]byte{seedKey}, true, nil // a lone closed edge (e.g. a cylinder rim)
	}
	c := newEdgeChain(seed)
	closed := c.grow(seed, seed.EndVertex(), seed.StartVertex(), conv, angTol, c.appendEdge)
	if !closed {
		c.grow(seed, seed.StartVertex(), seed.EndVertex(), conv, angTol, c.prependEdge)
	}
	return c.keys(), closed, nil
}

// edgeChain accumulates the ordered run of chained edges and the set already taken (so the walk
// never revisits an edge or steps back onto the seed).
type edgeChain struct {
	edges   []*topo.Edge
	visited map[uint64]bool
}

func newEdgeChain(seed *topo.Edge) *edgeChain {
	return &edgeChain{edges: []*topo.Edge{seed}, visited: map[uint64]bool{seed.ID(): true}}
}

func (c *edgeChain) appendEdge(e *topo.Edge) { c.edges = append(c.edges, e); c.visited[e.ID()] = true }
func (c *edgeChain) prependEdge(e *topo.Edge) {
	c.edges = append([]*topo.Edge{e}, c.edges...)
	c.visited[e.ID()] = true
}

func (c *edgeChain) keys() [][]byte {
	keys := make([][]byte, len(c.edges))
	for i, e := range c.edges {
		keys[i] = e.ReferenceKey()
	}
	return keys
}

// grow walks from cur across frontier onto tangent neighbours until the run dies or reaches
// target (closing a loop). add places each new edge at the correct end (append downstream,
// prepend upstream). Returns true when the run closed back onto target.
func (c *edgeChain) grow(cur *topo.Edge, frontier, target *topo.Vertex, conv EdgeConvexity, angTol float64, add func(*topo.Edge)) bool {
	for {
		next := c.pickContinuation(cur, frontier, conv, angTol)
		if next == nil {
			return false
		}
		nv := otherVertex(next, frontier)
		add(next)
		if nv == target {
			return true
		}
		cur, frontier = next, nv
	}
}

// pickContinuation returns the unvisited crease edge at frontier that best continues cur's
// guideline (most antiparallel away-from-vertex tangents, within angTol of straight-through),
// or nil when none qualifies.
func (c *edgeChain) pickContinuation(cur *topo.Edge, frontier *topo.Vertex, conv EdgeConvexity, angTol float64) *topo.Edge {
	tCur := unitTangentAwayFromVertex(cur, frontier)
	var best *topo.Edge
	bestAng := stdmath.Pi - angTol
	for _, cand := range frontier.Edges() {
		if cand == cur || c.visited[cand.ID()] || ClassifyEdgeConvexity(cand) != conv {
			continue
		}
		if !onlyCreasesAtVertex(frontier, cur, cand) {
			continue
		}
		if ang := float64(tCur.AngleTo(unitTangentAwayFromVertex(cand, frontier))); ang > bestAng {
			best, bestAng = cand, ang
		}
	}
	return best
}

// onlyCreasesAtVertex reports whether cur and cand are the sole fillet-able creases meeting at v
// — every other incident edge is smooth (tangent) or a boundary. This is OCCT FaceTangency's
// invariant: with no third crease, the two supporting faces pass tangentially through the
// junction, so the blend guideline continues from cur onto cand.
func onlyCreasesAtVertex(v *topo.Vertex, cur, cand *topo.Edge) bool {
	for _, e := range v.Edges() {
		if e == cur || e == cand {
			continue
		}
		if k := ClassifyEdgeConvexity(e); k == EdgeConvex || k == EdgeConcave {
			return false
		}
	}
	return true
}

// unitTangentAwayFromVertex is e's unit tangent pointing away from v along the edge (toward its
// other vertex). Two edges continue G1 through v when these point in opposite directions.
func unitTangentAwayFromVertex(e *topo.Edge, v *topo.Vertex) math.Vector3 {
	crv := e.Geometry()
	lo, hi := crv.Domain()
	t := crv.TangentAt(lo)
	if e.EndVertex() == v && e.StartVertex() != v {
		t = crv.TangentAt(hi).Negate()
	}
	if l := float64(t.Length()); l > 0 {
		t = t.Scale(math.Scalar(1 / l))
	}
	return t
}

// otherVertex returns the endpoint of e that is not v.
func otherVertex(e *topo.Edge, v *topo.Vertex) *topo.Vertex {
	if e.StartVertex() == v {
		return e.EndVertex()
	}
	return e.StartVertex()
}
