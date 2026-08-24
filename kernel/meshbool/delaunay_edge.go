// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// On-edge point insertion for the Delaunay mesh. A point on an edge splits the two
// triangles sharing it into four (or one boundary triangle into two), then
// legalizes the outer edges — the exact-arithmetic analogue of the interior split.
// It is separate from delaunay.go because the four-triangle re-linking is the
// intricate case that most warrants isolation.

// splitEdge inserts pi, which lies on edge e = (a,b) of triangle ti.
func (d *delaunayMesh) splitEdge(ti, e, pi int) {
	a, b := d.tris[ti].v[e], d.tris[ti].v[(e+1)%3]
	if d.tris[ti].adj[e] < 0 {
		d.splitBoundaryEdge(ti, a, b, pi)
		return
	}
	d.splitSharedEdge(ti, a, b, pi)
}

// splitBoundaryEdge splits triangle ti (edge (a,b) on the domain boundary) at pi
// into (a,pi,c) and (pi,b,c).
func (d *delaunayMesh) splitBoundaryEdge(ti, a, b, pi int) {
	c := d.oppositeVertex(ti, a, b)
	e := d.edgeSlot(ti, a, b)
	nCA := d.tris[ti].adj[(e+2)%3]
	nBC := d.tris[ti].adj[(e+1)%3]
	tb := len(d.tris)
	d.tris[ti] = dtri{v: [3]int{a, pi, c}, adj: [3]int{-1, -1, -1}}
	d.tris = append(d.tris, dtri{v: [3]int{pi, b, c}, adj: [3]int{-1, -1, -1}})
	d.link(ti, 1, tb)  // (pi,c)
	d.link(ti, 2, nCA) // (c,a)
	d.link(tb, 1, nBC) // (b,c)
	d.legalizeOpp(ti, pi)
	d.legalizeOpp(tb, pi)
}

// splitSharedEdge splits the two triangles sharing edge (a,b) at pi into the four
// triangles fanning from pi.
func (d *delaunayMesh) splitSharedEdge(ti, a, b, pi int) {
	e := d.edgeSlot(ti, a, b)
	tj := d.tris[ti].adj[e]
	c := d.oppositeVertex(ti, a, b)
	nCA := d.tris[ti].adj[(e+2)%3]
	nBC := d.tris[ti].adj[(e+1)%3]
	se := d.edgeSlot(tj, b, a)
	dd := d.oppositeVertex(tj, a, b)
	nAD := d.tris[tj].adj[(se+1)%3]
	nDB := d.tris[tj].adj[(se+2)%3]

	tb, td := len(d.tris), len(d.tris)+1
	d.tris[ti] = dtri{v: [3]int{a, pi, c}, adj: [3]int{-1, -1, -1}}  // Ta
	d.tris[tj] = dtri{v: [3]int{b, pi, dd}, adj: [3]int{-1, -1, -1}} // Tc
	d.tris = append(d.tris,
		dtri{v: [3]int{pi, b, c}, adj: [3]int{-1, -1, -1}},  // Tb
		dtri{v: [3]int{pi, a, dd}, adj: [3]int{-1, -1, -1}}) // Td

	d.link(ti, 1, tb)  // Ta (pi,c) ↔ Tb (c,pi)
	d.link(ti, 2, nCA) // Ta (c,a)
	d.link(ti, 0, td)  // Ta (a,pi) ↔ Td (pi,a)
	d.link(tb, 1, nBC) // Tb (b,c)
	d.link(tb, 0, tj)  // Tb (pi,b) ↔ Tc (b,pi)
	d.link(tj, 1, td)  // Tc (pi,dd) ↔ Td (dd,pi)
	d.link(tj, 2, nDB) // Tc (dd,b)
	d.link(td, 1, nAD) // Td (a,dd)

	d.legalizeOpp(ti, pi)
	d.legalizeOpp(tb, pi)
	d.legalizeOpp(tj, pi)
	d.legalizeOpp(td, pi)
}
