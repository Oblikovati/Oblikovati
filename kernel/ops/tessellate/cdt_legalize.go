// SPDX-License-Identifier: GPL-2.0-only

package tessellate

// In-circle legalization after constraint recovery (#1604, audit A8).
//
// recoverSegment flips edges only until the constraint segment exists, which retriangulates the
// corridor as two fans hanging off the segment — a valid triangulation but generally NOT Delaunay
// (arbitrarily thin triangles). Every flip therefore queues its new diagonal (see rebuildFlip),
// and insertConstraint drains the queue through recursive Lawson legalization once the segment is
// recorded, restoring the empty-circumcircle property everywhere except across constrained edges
// (Anglada 1997; Sloan 1993). With the mesh a true CDT again, Bowyer–Watson insertion can seed its
// cavity by the adjacency walk instead of the exhaustive firstBad scan (see locateSeed).

// legalizePending drains the diagonals queued by recovery flips, restoring the local Delaunay
// property. Terminates because each Lawson flip strictly lowers the lifted-paraboloid surface (the
// classic flip-algorithm argument) and the exact in-circle predicate admits no cocircular cycling;
// the iteration cap is insurance against a predicate bug — bailing leaves a valid, merely
// less-Delaunay mesh, never a broken one.
func (m *cdt) legalizePending() {
	for guard := 8*len(m.tris) + 64; len(m.pendingLegal) > 0 && guard > 0; guard-- {
		e := m.pendingLegal[len(m.pendingLegal)-1]
		m.pendingLegal = m.pendingLegal[:len(m.pendingLegal)-1]
		m.legalizeEdge(e[0], e[1])
	}
	m.pendingLegal = m.pendingLegal[:0]
}

// legalizeEdge applies one Lawson step at edge (u,w): if the apex across the edge lies strictly
// inside the adjacent triangle's circumcircle, flip the edge and re-enqueue the four quad sides —
// the only edges whose local Delaunay status the flip can change (the new diagonal is Delaunay by
// construction, so the queue shrinks). Constrained edges are walls the Delaunay property is
// exempt across; a stale queue entry whose edge an earlier flip already removed is skipped.
func (m *cdt) legalizeEdge(u, w int) {
	if m.con[conKey(u, w)] > 0 {
		return
	}
	t, i, ok := m.edgeTriangle(u, w)
	if !ok {
		return
	}
	s := m.tris[t].n[i]
	if s < 0 {
		return
	}
	c, d := m.tris[t].v[i], m.apex(s, u, w)
	if d < 0 || inCircle(m.pts[m.tris[t].v[0]], m.pts[m.tris[t].v[1]], m.pts[m.tris[t].v[2]], m.pts[d]) <= 0 {
		return // already locally Delaunay (or exactly cocircular: either diagonal is legal)
	}
	// A strict in-circle violation in a valid planar triangulation implies the quad is strictly
	// convex, so the flip cannot fail; the guard is belt-and-braces against degenerate coordinates.
	if !m.flip(t, i) {
		return
	}
	m.pendingLegal = append(m.pendingLegal, [2]int{u, c}, [2]int{c, w}, [2]int{w, d}, [2]int{d, u})
}
