// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Collinear-vertex dissolve for reconstruction (ADR-0056 / #2247). The mesh arrangement can drop a
// vertex in the INTERIOR of a straight edge — a grazing tangent facet, or a neighbour-tag change on
// the tangent line — so reconstruction stitches ONE straight edge as two, leaving a spurious 2-valent
// collinear vertex (a T-junction) that the exact planar boolean never makes. That extra edge/vertex is
// a real topology divergence: it breaks a downstream near-tangent boolean and misleads every consumer
// that walks the boundary. brep keeps the edge whole; a reconstructed body must too. This rebuilds the
// stitched body, merging every maximal chain of collinear straight edges joined only at 2-valent
// vertices back into a single edge and dropping the interior vertices. A vertex whose two edges bound a
// different face pair, or that carries a real (>2) valence or a genuine direction change, is kept — so
// only spurious splits dissolve. A body with none is returned unchanged.
func dissolveCollinearVertices(body *topo.Body) *topo.Body {
	if body == nil {
		return body
	}
	tol := geom.ResolutionForBox(body.RangeBox()).Weld()
	drop := dissolvableVertices(body, tol)
	if len(drop) == 0 {
		return body
	}
	chains := edgeChains(body, drop)
	return rebuildDissolved(body, drop, chains)
}

// dissolvableVertices returns the set of vertices with exactly two incident edges that are straight,
// collinear through the vertex, and bound the same pair of faces — a spurious mid-edge split.
func dissolvableVertices(body *topo.Body, tol float64) map[*topo.Vertex]bool {
	out := map[*topo.Vertex]bool{}
	for _, v := range body.Vertices() {
		es := v.Edges()
		if len(es) == 2 && edgeIsStraight(es[0], tol) && edgeIsStraight(es[1], tol) &&
			collinearThrough(es[0], es[1], v, tol) && sameFacePair(es[0], es[1]) {
			out[v] = true
		}
	}
	return out
}

// edgeChain is a maximal run of edges linked only at dissolvable vertices, plus its two surviving end
// vertices and the merged straight curve between them.
type edgeChain struct {
	edges  map[*topo.Edge]bool
	a, b   *topo.Vertex
	merged geom.Curve3
}

// edgeChains groups the body's edges into maximal chains joined at dissolvable vertices, returning one
// chain per group that spans more than one edge; single edges are left out (kept as-is).
func edgeChains(body *topo.Body, drop map[*topo.Vertex]bool) map[*topo.Edge]*edgeChain {
	out := map[*topo.Edge]*edgeChain{}
	for _, es := range groupEdgesAtVertices(body, drop) {
		if len(es) < 2 {
			continue
		}
		ch := chainOf(es)
		for _, e := range es {
			out[e] = ch
		}
	}
	return out
}

// groupEdgesAtVertices union-finds the body's edges into connected groups, joining the two edges at
// every dissolvable vertex, and returns the groups keyed by an arbitrary representative edge.
func groupEdgesAtVertices(body *topo.Body, drop map[*topo.Vertex]bool) map[*topo.Edge][]*topo.Edge {
	parent := map[*topo.Edge]*topo.Edge{}
	for _, e := range body.Edges() {
		parent[e] = e
	}
	find := func(e *topo.Edge) *topo.Edge {
		for parent[e] != e {
			parent[e] = parent[parent[e]]
			e = parent[e]
		}
		return e
	}
	for v := range drop {
		es := v.Edges()
		if ra, rb := find(es[0]), find(es[1]); ra != rb {
			parent[ra] = rb
		}
	}
	groups := map[*topo.Edge][]*topo.Edge{}
	for _, e := range body.Edges() {
		groups[find(e)] = append(groups[find(e)], e)
	}
	return groups
}

// chainOf builds the chain record for a group of linked edges: its ends are the vertices used by
// exactly one edge, and its merged curve is the straight segment between them.
func chainOf(es []*topo.Edge) *edgeChain {
	count := map[*topo.Vertex]int{}
	for _, e := range es {
		count[e.StartVertex()]++
		count[e.EndVertex()]++
	}
	var ends []*topo.Vertex
	for v, n := range count {
		if n == 1 {
			ends = append(ends, v)
		}
	}
	set := map[*topo.Edge]bool{}
	for _, e := range es {
		set[e] = true
	}
	ch := &edgeChain{edges: set}
	if len(ends) == 2 {
		ch.a, ch.b = orderEnds(ends[0], ends[1])
		ch.merged = geom.NewLineSegment(ch.a.Point(), ch.b.Point())
	}
	return ch
}

// orderEnds returns the two end vertices in a deterministic order (by point), so the merged edge and
// its lineage are stable across runs and platforms.
func orderEnds(x, y *topo.Vertex) (*topo.Vertex, *topo.Vertex) {
	if lessPoint(y.Point(), x.Point()) {
		return y, x
	}
	return x, y
}

func lessPoint(p, q math.Point3) bool {
	if p.X != q.X {
		return p.X < q.X
	}
	if p.Y != q.Y {
		return p.Y < q.Y
	}
	return p.Z < q.Z
}

// collinearThrough reports whether e1 and e2 (sharing v) lie on one straight line — v's far neighbours
// and v are collinear.
func collinearThrough(e1, e2 *topo.Edge, v *topo.Vertex, tol float64) bool {
	o1, o2 := otherEnd(e1, v), otherEnd(e2, v)
	return pointOnLine(o1.Point(), o2.Point(), v.Point(), tol)
}

func otherEnd(e *topo.Edge, v *topo.Vertex) *topo.Vertex {
	if e.StartVertex() == v {
		return e.EndVertex()
	}
	return e.StartVertex()
}

// edgeIsStraight reports whether an edge traces a straight line (its mid sample lies on the chord).
func edgeIsStraight(e *topo.Edge, tol float64) bool {
	c := e.Geometry()
	lo, hi := c.Domain()
	return pointOnLine(e.StartVertex().Point(), e.EndVertex().Point(), c.PointAt((lo+hi)/2), tol)
}

// sameFacePair reports whether two edges bound the same set of faces, so merging them across their
// shared vertex keeps every face's boundary intact.
func sameFacePair(e1, e2 *topo.Edge) bool {
	fa, fb := e1.Faces(), e2.Faces()
	if len(fa) != len(fb) {
		return false
	}
	set := map[*topo.Face]bool{}
	for _, f := range fa {
		set[f] = true
	}
	for _, f := range fb {
		if !set[f] {
			return false
		}
	}
	return true
}
