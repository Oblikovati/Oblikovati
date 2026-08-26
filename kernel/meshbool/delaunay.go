// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// A Delaunay triangulation with triangle adjacency — the base for the robust
// constrained triangulation (ADR-0052 layer 2) that replaces the cavity-force
// method, which #2084's near-tangent sliver fans defeat. Points are inserted
// incrementally (Lawson): locate the containing triangle, split it, then legalize
// by exact in-circle edge flips, so the triangulation stays Delaunay and avoids
// gratuitous slivers. The face triangle itself bounds the domain (every inserted
// point lies inside it), so no artificial super-triangle is needed. Every flip and
// orientation is decided by an exact predicate, so the flip loop cannot oscillate.

// dtri is one triangle: three vertex indices (CCW in the projection) and the three
// neighbouring triangles, adj[i] sitting across edge (v[i], v[(i+1)%3]); -1 is the
// domain boundary.
type dtri struct {
	v   [3]int
	adj [3]int
}

// delaunayMesh is the triangulation over a face's vertices in its projection.
type delaunayMesh struct {
	verts []Point
	tris  []dtri
	axis  int
}

// newDelaunayInTriangle starts the triangulation as the single face triangle,
// stored counterclockwise so an in-circle-illegal edge always spans a convex quad.
func newDelaunayInTriangle(face [3]Point) *delaunayMesh {
	d := &delaunayMesh{verts: []Point{face[0], face[1], face[2]}, axis: planeAxis(face)}
	t := dtri{v: [3]int{0, 1, 2}, adj: [3]int{-1, -1, -1}}
	if orient2(face[0], face[1], face[2], d.axis) < 0 {
		t.v = [3]int{0, 2, 1}
	}
	d.tris = []dtri{t}
	return d
}

// Insert adds p to the triangulation. A point equal to an existing vertex, or
// outside the domain, is a no-op.
func (d *delaunayMesh) Insert(p Point) {
	pi := d.addVertex(p)
	ti, edge := d.locate(pi)
	switch {
	case ti < 0:
		return
	case edge < 0:
		d.splitInterior(ti, pi)
	default:
		d.splitEdge(ti, edge, pi)
	}
}

// triangles returns the live triangles as point triples.
func (d *delaunayMesh) triangles() [][3]Point {
	out := make([][3]Point, len(d.tris))
	for i, t := range d.tris {
		out[i] = [3]Point{d.verts[t.v[0]], d.verts[t.v[1]], d.verts[t.v[2]]}
	}
	return out
}

// addVertex returns p's vertex index, reusing an exactly-coincident vertex.
func (d *delaunayMesh) addVertex(p Point) int {
	for i, v := range d.verts {
		if v.Equal(p) {
			return i
		}
	}
	d.verts = append(d.verts, p)
	return len(d.verts) - 1
}

// locate finds the triangle containing vertex pi and returns (triangle, edge):
// edge -1 when strictly interior, the edge index when on an edge, or (-1,-1) when
// pi already is a triangle vertex or lies outside the domain.
func (d *delaunayMesh) locate(pi int) (int, int) {
	p := d.verts[pi]
	for ti := range d.tris {
		t := d.tris[ti]
		if t.v[0] == pi || t.v[1] == pi || t.v[2] == pi {
			return -1, -1
		}
		s := [3]int{
			orient2(d.verts[t.v[0]], d.verts[t.v[1]], p, d.axis),
			orient2(d.verts[t.v[1]], d.verts[t.v[2]], p, d.axis),
			orient2(d.verts[t.v[2]], d.verts[t.v[0]], p, d.axis),
		}
		if s[0] < 0 || s[1] < 0 || s[2] < 0 {
			continue // outside this (CCW) triangle
		}
		edge := -1
		for e := range 3 {
			if s[e] == 0 {
				edge = e // on this edge; two zeros would be a vertex, already handled above
			}
		}
		return ti, edge
	}
	return -1, -1
}

// link sets tris[ti].adj[e] = tj and, for a real tj, the reverse back-pointer.
func (d *delaunayMesh) link(ti, e, tj int) {
	d.tris[ti].adj[e] = tj
	if tj < 0 {
		return
	}
	a, b := d.tris[ti].v[e], d.tris[ti].v[(e+1)%3]
	if se := d.edgeSlot(tj, b, a); se >= 0 {
		d.tris[tj].adj[se] = ti
	}
}

// edgeSlot returns the edge index of the directed edge (a,b) in triangle ti, or -1.
func (d *delaunayMesh) edgeSlot(ti, a, b int) int {
	t := d.tris[ti]
	for e := range 3 {
		if t.v[e] == a && t.v[(e+1)%3] == b {
			return e
		}
	}
	return -1
}

// oppositeVertex returns the vertex of triangle ti that is neither a nor b, where a
// and b are two of its three vertices (their sum with the third is the total).
func (d *delaunayMesh) oppositeVertex(ti, a, b int) int {
	t := d.tris[ti].v
	return t[0] + t[1] + t[2] - a - b
}

// splitInterior replaces triangle ti (containing pi in its interior) with the three
// triangles fanning from pi, then legalizes the three outer edges.
func (d *delaunayMesh) splitInterior(ti, pi int) {
	v0, v1, v2 := d.tris[ti].v[0], d.tris[ti].v[1], d.tris[ti].v[2]
	a0, a1, a2 := d.tris[ti].adj[0], d.tris[ti].adj[1], d.tris[ti].adj[2]
	t1, t2 := len(d.tris), len(d.tris)+1
	d.tris[ti] = dtri{v: [3]int{v0, v1, pi}, adj: [3]int{-1, -1, -1}}
	d.tris = append(d.tris, dtri{v: [3]int{v1, v2, pi}, adj: [3]int{-1, -1, -1}}, dtri{v: [3]int{v2, v0, pi}, adj: [3]int{-1, -1, -1}})
	d.link(ti, 1, t1)
	d.link(t1, 1, t2)
	d.link(t2, 1, ti)
	d.link(ti, 0, a0)
	d.link(t1, 0, a1)
	d.link(t2, 0, a2)
	d.legalize(ti, 0)
	d.legalize(t1, 0)
	d.legalize(t2, 0)
}

// legalize flips edge e of triangle ti if the opposite vertex across it lies inside
// the circumcircle of ti (Lawson), recursing on the two edges the flip exposes.
func (d *delaunayMesh) legalize(ti, e int) {
	tj := d.tris[ti].adj[e]
	if tj < 0 {
		return
	}
	a, b := d.tris[ti].v[e], d.tris[ti].v[(e+1)%3]
	p := d.tris[ti].v[(e+2)%3]
	q := d.oppositeVertex(tj, a, b)
	if inCircleSign(d.verts[a], d.verts[b], d.verts[p], d.verts[q], d.axis) <= 0 {
		return
	}
	d.flip(ti, tj, a, b, p, q)
	d.legalize(ti, 1) // edge (a,q) opposite p
	d.legalize(tj, 1) // edge (q,b) opposite p
}

// flip replaces the two triangles sharing edge (a,b) — with apexes p (in ti) and q
// (in tj) — by the two sharing edge (p,q): (p,a,q) reuses ti and (p,q,b) reuses tj.
func (d *delaunayMesh) flip(ti, tj, a, b, p, q int) {
	e := d.edgeSlot(ti, a, b)
	se := d.edgeSlot(tj, b, a)
	nbrBP := d.tris[ti].adj[(e+1)%3]
	nbrPA := d.tris[ti].adj[(e+2)%3]
	nbrAQ := d.tris[tj].adj[(se+1)%3]
	nbrQB := d.tris[tj].adj[(se+2)%3]
	d.tris[ti] = dtri{v: [3]int{p, a, q}, adj: [3]int{-1, -1, -1}}
	d.tris[tj] = dtri{v: [3]int{p, q, b}, adj: [3]int{-1, -1, -1}}
	d.link(ti, 2, tj)
	d.link(ti, 0, nbrPA)
	d.link(ti, 1, nbrAQ)
	d.link(tj, 1, nbrQB)
	d.link(tj, 2, nbrBP)
}

// legalizeOpp legalizes the edge of ti opposite vertex pi.
func (d *delaunayMesh) legalizeOpp(ti, pi int) {
	for e := range 3 {
		if d.tris[ti].v[e] != pi && d.tris[ti].v[(e+1)%3] != pi {
			d.legalize(ti, e)
			return
		}
	}
}
