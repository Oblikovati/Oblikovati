// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"fmt"
	stdmath "math"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// DefaultTangentChainAngle mirrors OCCT ChFi3d_FilBuilder's `angular` tolerance (1e-2 rad,
// ChFi3d_FilBuilder.hxx): the band within which a junction counts as DEGENERATE — the current or
// candidate edge's two support faces go tangent AT the shared vertex — where PerformElement demands
// near-parallel edge tangents (`av1v2 < ta`, ChFi3d_Builder_1.cxx) instead of its ordinary
// quarter-turn guard. It is NOT the continuation gate itself: chains continue on FaceTangency (every
// other edge at the vertex smooth) plus the no-turn-back `av1v2 < M_PI/2` rule, so a rounded rim's
// line↔arc junction passes at machine precision and a 90° box corner stops.
const DefaultTangentChainAngle = 1e-2

// TangentEdgeChain expands a single seed edge into the maximal run OCCT's `blend` propagates a
// pick along — the "select tangent chain / loop" selection behind fillet/chamfer
// (Oblikovati/Oblikovati#1798) and the spine PerformElement builds. From the seed it walks both
// endpoint vertices, at each step onto the one neighbour edge that (a) is itself a fillet-able
// crease, (b) is the only other crease at that vertex — every other incident edge is smooth, so
// the two supporting faces continue tangentially across the junction (ChFi3d_Builder::FaceTangency,
// ChFi3d_Builder_1.cxx), and (c) does not turn back — the away-from-vertex tangents subtend more
// than a right angle (`av1v2 < M_PI/2`, PerformElement's PRO9810 guard), tightened to angTol at a
// DEGENERATE junction where a support pair goes tangent at the vertex (TangentOnVertex, cf
// CTS21610_1). It returns the ordered edge keys (seed included) and whether the run closes a loop.
//
// ★ This REPLACED a 1° antiparallel gate + same-convexity constraint that were KNOWN-divergent from
// OCCT (they shortened two of complex/C5's chains from 5 edges to 2 and 3 —
// occt_tangent_chain_test.go's ledger note); the corpus-wide ratchet
// TestEveryPickBlendsItsWholeTangentChain now measures this walker against the harness's
// independent oracle port on every record.
//
// Example: one click on a straight edge of a rounded-rectangle rim yields all 8 edges
// (4 lines + 4 G1 corner arcs) with closed=true.
func TangentEdgeChain(body *topo.Body, seedKey []byte, angTol float64) ([][]byte, bool, error) {
	seed, ok := body.FindEdgeByKey(seedKey)
	if !ok {
		return nil, false, fmt.Errorf("tangent chain: seed edge key not found in body of %d edges", len(body.Edges()))
	}
	if !filletableCrease(seed) {
		return [][]byte{seedKey}, false, nil // a smooth/boundary seed cannot anchor a fillet chain
	}
	if seed.StartVertex() == seed.EndVertex() {
		return [][]byte{seedKey}, true, nil // a lone closed edge (e.g. a cylinder rim)
	}
	c := newEdgeChain(seed)
	closed := c.grow(seed, seed.EndVertex(), seed.StartVertex(), angTol, c.appendEdge)
	if !closed {
		c.grow(seed, seed.StartVertex(), seed.EndVertex(), angTol, c.prependEdge)
	}
	return c.keys(), closed, nil
}

// filletableCrease reports whether e is a crease a fillet can anchor on — convex or concave, not a
// smooth/tangent or boundary edge (the same 1e-3 rad smooth band as the harness oracle's
// occtChainCrease, via ClassifyEdgeConvexity).
func filletableCrease(e *topo.Edge) bool {
	k := ClassifyEdgeConvexity(e)
	return k == EdgeConvex || k == EdgeConcave
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
func (c *edgeChain) grow(cur *topo.Edge, frontier, target *topo.Vertex, angTol float64, add func(*topo.Edge)) bool {
	for {
		next := c.pickContinuation(cur, frontier, angTol)
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

// pickContinuation returns the unvisited crease edge at frontier OCCT would continue onto — the
// straightest one clearing the turn-back gate — or nil when none qualifies.
func (c *edgeChain) pickContinuation(cur *topo.Edge, frontier *topo.Vertex, angTol float64) *topo.Edge {
	tCur := unitTangentAwayFromVertex(cur, frontier)
	var best *topo.Edge
	bestAng := 0.0
	for _, cand := range frontier.Edges() {
		if cand == cur || c.visited[cand.ID()] || !filletableCrease(cand) {
			continue
		}
		if !onlyCreasesAtVertex(frontier, cur, cand) {
			continue
		}
		ang := float64(tCur.AngleTo(unitTangentAwayFromVertex(cand, frontier)))
		if ang > continuationGate(cur, cand, frontier, angTol) && ang > bestAng {
			best, bestAng = cand, ang
		}
	}
	return best
}

// continuationGate is the minimum away-tangent angle for cand to continue cur through frontier:
// OCCT PerformElement's no-turn-back rule (`av1v2 < M_PI/2` on forward tangents = away-tangents
// subtending MORE than a right angle), tightened to near-straight (π − angTol) at a DEGENERATE
// junction — one where cur's or cand's own support pair goes tangent at the vertex
// (TangentOnVertex; the CTS21610_1 case in ChFi3d_Builder_1.cxx).
func continuationGate(cur, cand *topo.Edge, v *topo.Vertex, angTol float64) float64 {
	if supportsTangentAtVertex(cur, v, angTol) || supportsTangentAtVertex(cand, v, angTol) {
		return stdmath.Pi - angTol
	}
	return stdmath.Pi / 2
}

// supportsTangentAtVertex reports whether e's two support faces are tangent AT vertex v — their
// material-outward normals at v within tol (OCCT's static TangentExtremity/TangentOnVertex).
func supportsTangentAtVertex(e *topo.Edge, v *topo.Vertex, tol float64) bool {
	faces := e.Faces()
	if len(faces) != 2 {
		return false
	}
	n1, ok1 := outwardFaceNormal(faces[0], v.Point())
	n2, ok2 := outwardFaceNormal(faces[1], v.Point())
	return ok1 && ok2 && float64(n1.AngleTo(n2)) < tol
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
