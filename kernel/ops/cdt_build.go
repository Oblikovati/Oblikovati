// SPDX-License-Identifier: GPL-2.0-only

package ops

// insert adds point index ip by connected-cavity Bowyer–Watson: find the triangles whose
// circumcircle contains the point (a connected star-shaped region), delete them, and fan the
// cavity boundary to the new point. Skips a point that coincides with an existing vertex (no
// strictly-bad triangle), which keeps a duplicated boundary sample from corrupting the mesh.
func (m *cdt) insert(ip int) {
	seed := m.firstBad(m.pts[ip])
	if seed < 0 {
		return // coincides with an existing vertex (no strictly-bad triangle)
	}
	m.fanCavity(ip, m.collectCavity(seed, m.pts[ip]))
}

// cavity is a Bowyer–Watson cavity: the triangles to retriangulate, kept BOTH as a BFS-ordered slice
// (so the boundary scan and triangle creation are deterministic — ranging a Go map is randomized,
// which made the whole triangulation non-reproducible) and as a membership set for the inside test.
type cavity struct {
	order []int
	in    map[int]bool
}

// firstBad returns any live triangle whose circumcircle strictly contains p (the triangle that
// contains p always qualifies), or -1 if none.
func (m *cdt) firstBad(p [2]float64) int {
	for t := range m.tris {
		if !m.dead[t] && inCircle(m.pts[m.tris[t].v[0]], m.pts[m.tris[t].v[1]], m.pts[m.tris[t].v[2]], p) > 0 {
			return t
		}
	}
	return -1
}

// collectCavity grows the connected set of triangles whose circumcircle contains p, starting from
// a known-bad seed (the Delaunay cavity is connected, so a BFS finds all of it). The BFS visit order
// is recorded so downstream processing is deterministic (see [cavity]).
func (m *cdt) collectCavity(seed int, p [2]float64) cavity {
	c := cavity{order: []int{seed}, in: map[int]bool{seed: true}}
	for queue := []int{seed}; len(queue) > 0; {
		t := queue[0]
		queue = queue[1:]
		for i := 0; i < 3; i++ {
			ne := m.tris[t].n[i]
			if ne < 0 || c.in[ne] {
				continue
			}
			if inCircle(m.pts[m.tris[ne].v[0]], m.pts[m.tris[ne].v[1]], m.pts[m.tris[ne].v[2]], p) > 0 {
				c.in[ne] = true
				c.order = append(c.order, ne)
				queue = append(queue, ne)
			}
		}
	}
	return c
}

type cavityEdge struct{ a, b, ne int }

// fanCavity deletes the cavity triangles and creates one new triangle per cavity-boundary edge,
// fanning to the inserted point ip; it relinks outside neighbours and stitches the new fan.
func (m *cdt) fanCavity(ip int, c cavity) {
	bnd := m.cavityBoundary(c)
	for _, t := range c.order {
		m.dead[t] = true
	}
	pending := map[int][2]int{} // shared (ip,x) edges: other-vertex x → first (tri, localIndex)
	link := func(t, i, other int) {
		if f, ok := pending[other]; ok {
			m.tris[t].n[i] = f[0]
			m.tris[f[0]].n[f[1]] = t
			delete(pending, other)
		} else {
			pending[other] = [2]int{t, i}
		}
	}
	for _, e := range bnd {
		nt := m.addTri(e.a, e.b, ip) // CCW: ip is left of a→b (cavity interior side)
		m.tris[nt].n[2] = e.ne       // edge opposite ip is (a,b), shared with the outside neighbour
		m.relinkOpposite(e.ne, e.a, e.b, nt)
		link(nt, 0, e.b) // edge opposite a = (b, ip)
		link(nt, 1, e.a) // edge opposite b = (ip, a)
	}
}

// cavityBoundary returns the directed CCW edges on the boundary of the cavity (edges whose other
// side is outside the cavity), each tagged with the outside neighbour triangle. It scans the cavity
// triangles in BFS order so the boundary (and the fan built from it) is deterministic.
func (m *cdt) cavityBoundary(c cavity) []cavityEdge {
	var bnd []cavityEdge
	for _, t := range c.order {
		for i := 0; i < 3; i++ {
			ne := m.tris[t].n[i]
			if ne >= 0 && c.in[ne] {
				continue
			}
			bnd = append(bnd, cavityEdge{m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3], ne})
		}
	}
	return bnd
}

func (m *cdt) addTri(a, b, c int) int {
	m.tris = append(m.tris, cdtTri{v: [3]int{a, b, c}, n: [3]int{-1, -1, -1}})
	m.dead = append(m.dead, false)
	return len(m.tris) - 1
}

// hasEdge reports whether edge (a,b) is present in some live triangle.
func (m *cdt) hasEdge(a, b int) bool {
	for t := range m.tris {
		if !m.dead[t] && m.localEdge(t, a, b) >= 0 {
			return true
		}
	}
	return false
}

// neighborAcross returns the triangle sharing edge (a,b) with triangle s (-1 if none).
func (m *cdt) neighborAcross(s, a, b int) int {
	if i := m.localEdge(s, a, b); i >= 0 {
		return m.tris[s].n[i]
	}
	return -1
}

// flip replaces the diagonal of the convex quad formed by triangle t and its neighbour across
// edge i with the other diagonal (Lawson flip). Returns false if there is no neighbour or the
// quad is not convex (the flip would make a zero/negative-area triangle).
func (m *cdt) flip(t, i int) bool {
	s := m.tris[t].n[i]
	if s < 0 {
		return false
	}
	c, pp, q := m.tris[t].v[i], m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
	d := -1
	for k := 0; k < 3; k++ {
		if v := m.tris[s].v[k]; v != pp && v != q {
			d = v
		}
	}
	if d < 0 || orient2d(m.pts[c], m.pts[pp], m.pts[d]) <= 0 || orient2d(m.pts[c], m.pts[d], m.pts[q]) <= 0 {
		return false
	}
	nCP, nQC := m.tris[t].n[(i+2)%3], m.tris[t].n[(i+1)%3]
	nPD, nDQ := m.neighborAcross(s, pp, d), m.neighborAcross(s, d, q)
	m.tris[t] = cdtTri{v: [3]int{c, pp, d}, n: [3]int{nPD, s, nCP}}
	m.tris[s] = cdtTri{v: [3]int{c, d, q}, n: [3]int{nDQ, nQC, t}}
	m.relinkOpposite(nPD, pp, d, t)
	m.relinkOpposite(nCP, c, pp, t)
	m.relinkOpposite(nDQ, d, q, s)
	m.relinkOpposite(nQC, q, c, s)
	return true
}

// insertConstraint forces edge (a,b) into the triangulation (flipping the edges it crosses) and
// marks it constrained. A non-flippable (non-convex) crossing is retried after others resolve it;
// it gives up if no progress is possible (a degenerate, near-collinear boundary).
func (m *cdt) insertConstraint(a, b int) {
	for tries := 0; !m.hasEdge(a, b) && tries < 4*len(m.tris)+8; tries++ {
		if !m.flipOneCrossing(a, b) {
			break
		}
	}
	m.con[conKey(a, b)] = true
}

// flipOneCrossing flips one flippable triangulation edge that properly crosses segment (a,b),
// returning whether it flipped one.
func (m *cdt) flipOneCrossing(a, b int) bool {
	pa, pb := m.pts[a], m.pts[b]
	for t := range m.tris {
		if m.dead[t] {
			continue
		}
		for i := 0; i < 3; i++ {
			u, w := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
			if u == a || u == b || w == a || w == b {
				continue // edges touching an endpoint never cross the segment
			}
			if segmentsCross(pa, pb, m.pts[u], m.pts[w]) && m.flip(t, i) {
				return true
			}
		}
	}
	return false
}

// segmentsCross reports whether open segments p1p2 and p3p4 properly intersect.
func segmentsCross(p1, p2, p3, p4 [2]float64) bool {
	d1 := orient2d(p3, p4, p1)
	d2 := orient2d(p3, p4, p2)
	d3 := orient2d(p1, p2, p3)
	d4 := orient2d(p1, p2, p4)
	return ((d1 > 0) != (d2 > 0)) && ((d3 > 0) != (d4 > 0))
}

// extractDomain 2-colours the triangulation inside/outside by flooding adjacency from the
// super-triangle region (outside) and toggling at every constrained edge, then returns the
// inside triangles (excluding any touching a super vertex) as CCW index triples.
func (m *cdt) extractDomain() [][3]int {
	inside := m.floodInside()
	if inside == nil {
		return nil
	}
	var out [][3]int
	for t := range m.tris {
		if !m.dead[t] && inside[t] && !m.hasSuper(t) {
			out = append(out, [3]int{m.tris[t].v[0], m.tris[t].v[1], m.tris[t].v[2]})
		}
	}
	return out
}

// floodInside 2-colours triangles inside/outside: BFS from the super-triangle region (outside),
// flipping the flag at every constrained edge. Returns nil if there is no live super triangle.
func (m *cdt) floodInside() []bool {
	seed := -1
	for t := range m.tris {
		if !m.dead[t] && m.hasSuper(t) {
			seed = t
			break
		}
	}
	if seed < 0 {
		return nil
	}
	inside := make([]bool, len(m.tris))
	visited := make([]bool, len(m.tris))
	visited[seed] = true
	for queue := []int{seed}; len(queue) > 0; {
		t := queue[0]
		queue = queue[1:]
		queue = m.floodStep(t, inside, visited, queue)
	}
	return inside
}

// floodStep visits triangle t's not-yet-seen neighbours, setting each one's inside flag (flipped
// across a constrained edge) and enqueuing it; returns the extended queue.
func (m *cdt) floodStep(t int, inside, visited []bool, queue []int) []int {
	for i := 0; i < 3; i++ {
		s := m.tris[t].n[i]
		if s < 0 || m.dead[s] || visited[s] {
			continue
		}
		a, b := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
		visited[s] = true
		inside[s] = inside[t] != m.con[conKey(a, b)]
		queue = append(queue, s)
	}
	return queue
}

func (m *cdt) hasSuper(t int) bool {
	return m.tris[t].v[0] >= m.nsup || m.tris[t].v[1] >= m.nsup || m.tris[t].v[2] >= m.nsup
}

// constrainedDelaunay triangulates the planar domain bounded by the given loops (the first/largest
// is the outer boundary, the rest are holes — but any nesting works: the flood toggles inside at
// each loop), returning CCW triangles inside the domain as index triples into pts. Every loop edge
// is a hard constraint; points not on a loop are interior Steiner points refining the mesh.
func constrainedDelaunay(pts [][2]float64, loops [][]int) [][3]int {
	if len(pts) < 3 {
		return nil
	}
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	for _, lp := range loops {
		for k := 0; k < len(lp); k++ {
			m.insertConstraint(lp[k], lp[(k+1)%len(lp)])
		}
	}
	return m.extractDomain()
}
