// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"cmp"
	"slices"
)

// Constraint (boundary segment) recovery by corridor-walk flips (#1409).
//
// Forcing a loop edge (a,b) into the triangulation means flipping the existing edges that the segment
// a→b crosses until a→b is itself an edge. The previous implementation found those crossings by an
// O(T) whole-mesh scan PER FLIP and re-tested presence with another O(T) scan, so recovering one edge
// was O(T²) and a whole refined patch O(T³) — the historical CDT freeze (#1073), papered over only by
// an iteration cap whose silent give-up enabled the watertightness leak fixed in #1410.
//
// This recovers each segment by the standard corridor walk (Anglada 1997; Shewchuk's Triangle segment
// recovery; de Berg et al. Computational Geometry §9.3): march the triangle strip the segment passes
// through using the n[3] adjacency, collecting ONLY the O(k) edges actually crossed, then resolve them
// by Lawson flips — a non-convex crossing is deferred until a neighbouring flip makes it convex, and a
// flip whose new diagonal still crosses the segment is itself enqueued. Cost is O(crossings), not O(T).
//
// A vertex→incident-triangle hint (incident, built once in constrain and kept current by touch) makes
// "find a triangle on this vertex's star" O(deg) instead of O(T). All real vertices sit strictly inside
// the super-triangle, so their stars are closed fans the rotation can circle without hitting a hull edge.
//
// Any degeneracy the clean corridor cannot classify (a vertex lying exactly on the segment, a star that
// is not a closed fan, a stall) falls back to recoverByFlips — the previous bounded flip loop — so the
// result is never worse than before, only the common case is now near-linear.

// insertConstraint forces edge (a,b) into the triangulation and records it constrained. A degenerate
// constraint between coincident endpoints (duplicate boundary samples) is skipped: it has no edge to
// recover (callers pass deduplicated representatives, see representatives/#1073). The constraint is
// recorded in con ONLY when its edge actually survives — marking it unconditionally registered a phantom
// boundary the flood could not toggle at, leaking the domain (#1410); a genuine non-recovery is collected
// so finalizeDomain falls back deterministically.
func (m *cdt) insertConstraint(a, b int) {
	if a == b || m.pts[a] == m.pts[b] {
		return
	}
	if m.recoverSegment(a, b) {
		m.con[conKey(a, b)]++
		m.legalizePending() // #1604: restore the Delaunay property around the recovered corridor
		return
	}
	m.legalizePending() // a failed recovery's flips are real corridor work: leave them Delaunay too
	if m.splitConstraintAtVertices(a, b) {
		return // recovered piecewise; each sub-segment reported its own status
	}
	m.unrecovered = append(m.unrecovered, [2]int{a, b})
}

// splitConstraintAtVertices handles the segment-through-vertex degeneracy (#1604): a constraint
// whose open interior passes EXACTLY through mesh vertices (the exact orient2d says so) can never
// be recovered as one edge — flips cannot remove a vertex from the segment's path. The standard
// resolution (Shewchuk's Triangle; Anglada 1997) splits the constraint at those vertices and
// recovers each sub-segment, which walls off the same geometric boundary for the domain flood.
// The unrolled periodic wall hits this on its seam: the hole corners' axial stations reappear as
// exact on-seam samples, and the whole seam used to be dropped as unrecoverable — the silent
// domain leak behind the #1410 fallback. Returns false when no on-segment vertex exists (a
// genuine non-recovery the caller records).
func (m *cdt) splitConstraintAtVertices(a, b int) bool {
	on := m.verticesOnSegment(a, b)
	if len(on) == 0 {
		return false
	}
	prev := a
	for _, v := range on {
		m.insertConstraint(prev, v)
		prev = v
	}
	m.insertConstraint(prev, b)
	return true
}

// verticesOnSegment returns the real vertices lying exactly on the open segment (a,b), ordered
// from a to b. Endpoint-coincident duplicates split nothing and are excluded; sub-segments
// between consecutive on-vertices contain no further on-vertices by construction, so the split
// recursion terminates after one level.
func (m *cdt) verticesOnSegment(a, b int) []int {
	pa, pb := m.pts[a], m.pts[b]
	var on []int
	for v := 0; v < m.nsup; v++ {
		p := m.pts[v]
		if v == a || v == b || p == pa || p == pb {
			continue
		}
		if orient2d(pa, pb, p) != 0 || !inSegmentBox(pa, pb, p) {
			continue
		}
		on = append(on, v)
	}
	slices.SortFunc(on, func(u, w int) int {
		du := sqDist(pa, m.pts[u])
		dw := sqDist(pa, m.pts[w])
		return cmp.Compare(du, dw)
	})
	return on
}

// inSegmentBox reports whether collinear point p lies within segment (pa,pb)'s bounding box —
// with exact collinearity established, this is the open-segment containment test.
func inSegmentBox(pa, pb, p [2]float64) bool {
	return p[0] >= min(pa[0], pb[0]) && p[0] <= max(pa[0], pb[0]) &&
		p[1] >= min(pa[1], pb[1]) && p[1] <= max(pa[1], pb[1])
}

// sqDist is the squared distance between two points (a monotone along-segment order for
// collinear points).
func sqDist(a, b [2]float64) float64 {
	dx, dy := b[0]-a[0], b[1]-a[1]
	return dx*dx + dy*dy
}

// recoverSegment ensures edge (a,b) exists, returning whether it does afterwards. It walks the
// constraint corridor and resolves the crossings by flips; on any degeneracy it falls back to the
// previous whole-mesh flip loop (rare, so the O(T²) cost is never on the hot path).
func (m *cdt) recoverSegment(a, b int) bool {
	if m.hasEdgeAround(a, b) {
		return true
	}
	if cross, ok := m.march(a, b); ok && m.resolveCrossings(a, b, cross) && m.hasEdgeAround(a, b) {
		return true
	}
	return m.recoverByFlips(a, b)
}

// march returns the triangulation edges segment a→b properly crosses, ordered from a to b, each as a
// vertex pair (p,q) with p strictly left of a→b and q strictly right. ok is false when a vertex lies on
// the segment or the corridor cannot be followed (the caller falls back). It assumes (a,b) is not
// already an edge (recoverSegment checks first).
func (m *cdt) march(a, b int) ([][2]int, bool) {
	pa, pb := m.pts[a], m.pts[b]
	t, p, q, ok := m.entry(a, b)
	if !ok {
		return nil, false
	}
	cross := [][2]int{{p, q}}
	cur := m.neighborAcross(t, p, q)
	for guard := 0; cur >= 0 && guard <= len(m.tris); guard++ {
		r := m.apex(cur, p, q)
		if r < 0 {
			return nil, false
		}
		if r == b {
			return cross, true // the corridor reached b
		}
		o := orient2d(pa, pb, m.pts[r])
		if o == 0 {
			return nil, false // a vertex lies on the segment: degenerate, fall back
		}
		if o > 0 { // r is left of a→b ⇒ the segment exits through edge (r,q)
			p = r
		} else { // r is right ⇒ it exits through edge (p,r)
			q = r
		}
		cross = append(cross, [2]int{p, q})
		cur = m.neighborAcross(cur, p, q)
	}
	return nil, false // ran off the hull or never reached b
}

// entry returns the star triangle of a whose opposite edge segment a→b crosses, with that edge's
// endpoints labelled (p left of a→b, q right). ok is false when b is collinear with a star edge
// (degenerate) or a's star is not a traversable closed fan.
func (m *cdt) entry(a, b int) (tri, p, q int, ok bool) {
	t := m.incidentTri(a)
	if t < 0 {
		return 0, 0, 0, false
	}
	start := t
	for guard := 0; guard <= len(m.tris); guard++ {
		p, q, found, degen := m.wedgeCrossing(a, b, t)
		if degen {
			return 0, 0, 0, false
		}
		if found {
			return t, p, q, true
		}
		nt := m.rotateAround(a, t)
		if nt < 0 || nt == start {
			return 0, 0, 0, false
		}
		t = nt
	}
	return 0, 0, 0, false
}

// wedgeCrossing tests whether segment a→b leaves star triangle t through the edge opposite a. When it
// does, it returns that edge's endpoints labelled p (left of a→b) and q (right), found=true. degen=true
// signals b is collinear with a star edge (the corridor cannot be classified cleanly — abandon it).
func (m *cdt) wedgeCrossing(a, b, t int) (p, q int, found, degen bool) {
	pa, pb := m.pts[a], m.pts[b]
	i := vertexLocal(m.tris[t], a)
	if i < 0 {
		return 0, 0, false, true
	}
	u, w := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3] // CCW order a,u,w
	du := orient2d(pa, m.pts[u], pb)                   // b vs ray a→u
	dw := orient2d(pa, m.pts[w], pb)                   // b vs ray a→w
	if du == 0 || dw == 0 {
		return 0, 0, false, true // b collinear with a star edge
	}
	if du > 0 && dw < 0 { // b strictly inside the wedge ⇒ a→b crosses opposite edge (u,w)
		if orient2d(pa, pb, m.pts[u]) > 0 {
			return u, w, true, false // u left, w right
		}
		return w, u, true, false // w left, u right
	}
	return 0, 0, false, false
}

// resolveCrossings flips the crossed edges until segment a→b is an edge, processing them as a FIFO
// queue: a convex crossing is flipped (its new diagonal, if it still crosses a→b, is enqueued); a
// non-convex one is deferred to the back until a neighbouring flip makes it convex. Returns false if it
// stalls (degenerate input) so the caller can fall back. cross must be the segment's crossings (march).
func (m *cdt) resolveCrossings(a, b int, cross [][2]int) bool {
	pa, pb := m.pts[a], m.pts[b]
	queue := append([][2]int(nil), cross...)
	for guard := 0; len(queue) > 0; guard++ {
		if guard > 8*len(cross)+16 {
			return false // not converging: degenerate, fall back
		}
		e := queue[0]
		queue = queue[1:]
		t, i, ok := m.edgeTriangle(e[0], e[1])
		if !ok {
			continue // already removed by an earlier flip
		}
		c, d := m.flipDiagonal(t, i)
		if d < 0 || !m.flip(t, i) {
			queue = append(queue, e) // non-convex (or no neighbour): retry after others resolve it
			continue
		}
		if c != a && c != b && d != a && d != b && segmentsCross(pa, pb, m.pts[c], m.pts[d]) {
			queue = append(queue, [2]int{c, d}) // the new diagonal still crosses: it too must be flipped
		}
	}
	return true
}

// flipDiagonal returns the diagonal (c,d) a successful flip(t,i) would create — c the apex of t
// opposite the flipped edge, d the apex of the neighbour across it — or (-1,-1) if there is no
// neighbour. Read before flipping so the caller can test whether the new edge still crosses the segment.
func (m *cdt) flipDiagonal(t, i int) (int, int) {
	s := m.tris[t].n[i]
	if s < 0 {
		return -1, -1
	}
	p, q := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3] // the edge flip(t,i) replaces
	c := m.tris[t].v[i]
	for k := range 3 {
		if v := m.tris[s].v[k]; v != p && v != q {
			return c, v
		}
	}
	return -1, -1
}

// hasEdgeAround reports whether edge (a,b) is present, found by circling a's (or b's) star in O(deg) —
// the local replacement for the O(T) hasEdge scan in the recovery hot path.
func (m *cdt) hasEdgeAround(a, b int) bool {
	_, _, ok := m.edgeTriangle(a, b)
	return ok
}

// edgeTriangle returns a live triangle holding edge (p,q) and the local edge index flip expects (the
// index whose opposite edge is (p,q)), or ok=false if (p,q) is not a current edge. It circles p's star
// and, if that star is not a closed fan (p is a super/hull vertex), q's — so an edge incident to a hull
// vertex is still found.
func (m *cdt) edgeTriangle(p, q int) (int, int, bool) {
	if t, i, ok := m.edgeInStar(p, q); ok {
		return t, i, ok
	}
	return m.edgeInStar(q, p)
}

// edgeInStar searches the fan of triangles around vertex p for the one holding edge (p,q).
func (m *cdt) edgeInStar(p, q int) (int, int, bool) {
	t := m.incidentTri(p)
	if t < 0 {
		return 0, 0, false
	}
	start := t
	for guard := 0; guard <= len(m.tris); guard++ {
		if i := m.localEdge(t, p, q); i >= 0 {
			return t, i, true
		}
		nt := m.rotateAround(p, t)
		if nt < 0 || nt == start {
			return 0, 0, false
		}
		t = nt
	}
	return 0, 0, false
}

// rotateAround returns the triangle adjacent to t across the edge (p, v_next) at vertex p — one step of
// circling p's star in a consistent direction. -1 if p is not in t or the edge is a hull boundary.
func (m *cdt) rotateAround(p, t int) int {
	i := vertexLocal(m.tris[t], p)
	if i < 0 {
		return -1
	}
	return m.tris[t].n[(i+2)%3] // neighbour across edge (v[i], v[(i+1)%3]) = (p, v_next)
}

// apex returns the vertex of triangle t that is neither p nor q (the corridor apex across edge (p,q)),
// or -1 if t does not have both p and q.
func (m *cdt) apex(t, p, q int) int {
	for _, v := range m.tris[t].v {
		if v != p && v != q {
			return v
		}
	}
	return -1
}

// incidentTri returns a live triangle containing vertex p, preferring the O(1) hint and validating it
// (a flip may have moved p off the hinted triangle); a stale or absent hint triggers one O(T) rescan
// that refreshes the hint, so repeated lookups stay cheap.
func (m *cdt) incidentTri(p int) int {
	if m.incident != nil && p >= 0 && p < len(m.incident) {
		if t := m.incident[p]; t >= 0 && t < len(m.dead) && !m.dead[t] && vertexLocal(m.tris[t], p) >= 0 {
			return t
		}
	}
	return m.findIncidentScan(p)
}

// findIncidentScan locates a live triangle containing p by a whole-mesh scan and refreshes the hint —
// the fallback when the incidence hint is stale or was never built.
func (m *cdt) findIncidentScan(p int) int {
	for t := range m.tris {
		if !m.dead[t] && vertexLocal(m.tris[t], p) >= 0 {
			if m.incident != nil && p >= 0 && p < len(m.incident) {
				m.incident[p] = t
			}
			return t
		}
	}
	return -1
}

// buildIncident (re)builds the vertex→triangle hint from the current live triangulation, called once
// before constraint recovery so the corridor walk can find any vertex's star in O(1)+O(deg).
func (m *cdt) buildIncident() {
	m.incident = make([]int, len(m.pts))
	for i := range m.incident {
		m.incident[i] = -1
	}
	for t := range m.tris {
		if !m.dead[t] {
			m.touch(t)
		}
	}
}

// touch points every vertex of triangle t at t in the incidence hint, keeping it current as addTri and
// flip create/rewrite triangles. A no-op until buildIncident allocates the hint (so insertion pays nothing).
func (m *cdt) touch(t int) {
	if m.incident == nil {
		return
	}
	for _, v := range m.tris[t].v {
		if v >= 0 && v < len(m.incident) {
			m.incident[v] = t
		}
	}
}

// recoverByFlips is the previous whole-mesh flip loop, kept as the degenerate-case fallback for
// segments the corridor walk cannot recover cleanly. It is bounded and gives up (leaving the segment
// unrecovered for finalizeDomain) rather than spinning, exactly as before #1409.
func (m *cdt) recoverByFlips(a, b int) bool {
	for tries := 0; !m.hasEdge(a, b) && tries < 4*len(m.tris)+8; tries++ {
		if m.recoverFlipWork >= m.recoverBudget {
			// The face has spent its whole flip-recovery budget without realizing this edge: it is
			// thoroughly degenerate (non-simple). Stop the O(n·T²) spin and let insertConstraint's
			// splitConstraintAtVertices try (it recovers valid on-segment cases); the flag routes a
			// genuinely-unrecoverable face to the deterministic earcut fallback in finalizeDomain.
			m.overBudget = true
			return false
		}
		m.recoverFlipWork++
		if !m.flipOneCrossing(a, b) {
			break
		}
	}
	return m.hasEdge(a, b)
}

// vertexLocal returns the local index (0..2) of vertex v in tri, or -1 if absent.
func vertexLocal(tri cdtTri, v int) int {
	for i := range 3 {
		if tri.v[i] == v {
			return i
		}
	}
	return -1
}
