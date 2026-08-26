// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import (
	"math/big"
	"sort"
)

// Constraint-edge recovery for the Delaunay mesh (Sloan 1993). To force a segment
// (ui,vi) to appear as a triangulation edge, repeatedly flip an edge the segment
// crosses whose quadrilateral is convex; each such flip is local and exactly
// decided, and there is always a convex crossing edge to flip, so the segment ends
// up present. This replaces the cavity-boundary extraction that #2084's near-tangent
// sliver fans defeat: there is no cavity to trace, only local flips, so a pinched
// sliver region cannot break it.

// forceEdge recovers the segment (ui,vi) as an edge. PRECONDITION: ui and vi are
// vertices with no vertex strictly between them (the caller splits at every
// intermediate vertex first), so the segment crosses only edge interiors.
func (d *delaunayMesh) forceEdge(ui, vi int) {
	if ui == vi {
		return
	}
	// The bound guards against a non-terminating flip on a degenerate input; Sloan
	// guarantees progress, so it is a safety net, not the exit path.
	for guard := 4*len(d.tris) + 16; guard > 0; guard-- {
		if d.edgeExists(ui, vi) {
			return
		}
		ti, e, ok := d.convexCrossing(ui, vi)
		if !ok {
			return
		}
		d.flipEdge(ti, e)
	}
}

// forceSegment forces the whole segment [a,b] as triangulation edges, splitting it
// at every vertex lying on it so each sub-edge meets forceEdge's no-vertex-between
// precondition. a and b must already be vertices (the co-refinement inserts every
// endpoint and crossing first).
func (d *delaunayMesh) forceSegment(a, b Point) {
	if a.Equal(b) {
		return
	}
	on := d.verticesOnSegment(a, b)
	for k := 0; k+1 < len(on); k++ {
		d.forceEdge(on[k], on[k+1])
	}
}

// verticesOnSegment returns the indices of all vertices lying on the closed segment
// [a,b], ordered from a to b by their exact parameter.
func (d *delaunayMesh) verticesOnSegment(a, b Point) []int {
	lenSq := segParam(a, b, b)
	type onVert struct {
		idx int
		t   *big.Rat
	}
	var on []onVert
	for i, v := range d.verts {
		if !rcollinear(a, b, v) {
			continue
		}
		if t := segParam(a, b, v); t.Sign() >= 0 && t.Cmp(lenSq) <= 0 {
			on = append(on, onVert{i, t})
		}
	}
	sort.Slice(on, func(i, j int) bool { return on[i].t.Cmp(on[j].t) < 0 })
	idx := make([]int, len(on))
	for i, e := range on {
		idx[i] = e.idx
	}
	return idx
}

// edgeExists reports whether ui and vi are both vertices of some triangle.
func (d *delaunayMesh) edgeExists(ui, vi int) bool {
	for _, t := range d.tris {
		var hits int
		for _, x := range t.v {
			if x == ui || x == vi {
				hits++
			}
		}
		if hits == 2 {
			return true
		}
	}
	return false
}

// convexCrossing finds an edge the segment (ui,vi) properly crosses whose two
// triangles form a convex quad (so the flip is valid), or reports none.
func (d *delaunayMesh) convexCrossing(ui, vi int) (int, int, bool) {
	u, v := d.verts[ui], d.verts[vi]
	for ti := range d.tris {
		for e := range 3 {
			if d.tris[ti].adj[e] < 0 {
				continue
			}
			a, b := d.tris[ti].v[e], d.tris[ti].v[(e+1)%3]
			if a == ui || a == vi || b == ui || b == vi {
				continue // shares an endpoint with the segment; not an interior crossing
			}
			if !segmentsProperlyCross(u, v, d.verts[a], d.verts[b], d.axis) {
				continue
			}
			p := d.tris[ti].v[(e+2)%3]
			q := d.oppositeVertex(d.tris[ti].adj[e], a, b)
			if orient2(d.verts[p], d.verts[a], d.verts[q], d.axis) > 0 &&
				orient2(d.verts[p], d.verts[q], d.verts[b], d.axis) > 0 {
				return ti, e, true // flip yields two CCW triangles: the quad is convex
			}
		}
	}
	return 0, 0, false
}

// flipEdge flips edge e of triangle ti unconditionally (constraint recovery, not
// Delaunay legalization).
func (d *delaunayMesh) flipEdge(ti, e int) {
	tj := d.tris[ti].adj[e]
	a, b := d.tris[ti].v[e], d.tris[ti].v[(e+1)%3]
	p := d.tris[ti].v[(e+2)%3]
	q := d.oppositeVertex(tj, a, b)
	d.flip(ti, tj, a, b, p, q)
}
