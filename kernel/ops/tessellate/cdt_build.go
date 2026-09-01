// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import "slices"

// insert adds point index ip by connected-cavity Bowyer–Watson: find a triangle whose circumcircle
// contains the point, delete the connected star of such triangles, and fan the cavity boundary to the
// new point. Skips a point coinciding with an existing vertex (no strictly-bad triangle).
//
// The cavity is always seeded by the O(√N) adjacency WALK to the triangle containing p (#1408).
// Constraint recovery once left the mesh momentarily FOLDED — where the geometric walk lands in the
// wrong topological region — which forced an exact O(T) circumcircle scan per post-constraint
// insertion; recovery now legalizes its corridor (#1604), so the mesh is a true CDT between
// insertions and the walk is valid in both phases. collectCavity keeps the cavity star-shaped by
// never growing across a constrained edge.
func (m *cdt) insert(ip int) {
	p := m.pts[ip]
	seed := m.locateSeed(p)
	if seed < 0 {
		return // coincides with an existing vertex (no strictly-bad triangle)
	}
	m.fanCavity(ip, m.collectCavity(seed, p))
}

// locateSeed returns the circumcircle-bad triangle containing p to seed the Bowyer–Watson cavity, or
// -1 when p coincides with an existing vertex. The walk is valid with or without constraints because
// post-recovery legalization keeps the mesh a true CDT (#1604); a point strictly inside (or on an
// open edge of) its containing triangle is strictly inside that triangle's circumdisk, so
// inCircle ≤ 0 exactly characterizes a vertex-coincident duplicate.
func (m *cdt) locateSeed(p [2]float64) int {
	loc := m.locate(p)
	t := m.tris[loc]
	if inCircle(m.pts[t.v[0]], m.pts[t.v[1]], m.pts[t.v[2]], p) <= 0 {
		return -1 // p is on the located triangle's circumcircle (a vertex of it): a coincident duplicate
	}
	return loc
}

// locate returns a live triangle whose circumdisk contains p, by an adjacency WALK from the last
// insertion's triangle instead of the old O(N) scan: at each triangle it crosses the edge p lies
// outside of (right of the CCW edge), stepping toward p over the n[3] adjacency until p is inside
// (left of all three edges). Seeded from the previous insertion, this is O(√N) amortised on
// spatially-coherent input (Mücke–Saias–Zhu; the boundary and grid Steiner points already arrive in
// locality order), turning the whole Bowyer–Watson insertion from O(N²) to near-linear (#1408). The
// mesh is Delaunay between insertions, where the straight walk cannot cycle; the step cap with a
// [firstBad] fallback keeps it correct even on a degenerate intermediate state.
func (m *cdt) locate(p [2]float64) int {
	t := m.liveSeed()
	for steps := 0; steps <= len(m.tris); steps++ {
		next := m.walkAcross(t, p)
		if next < 0 {
			return t // p is left of (or on) all three edges → inside t
		}
		t = next
	}
	if bad := m.firstBad(p); bad >= 0 {
		return bad // walk did not converge (degenerate): fall back to the exhaustive scan
	}
	return t
}

// walkAcross returns the neighbour across an edge p lies strictly outside of (right of the directed
// CCW edge), or -1 when p is inside t (left of or on every edge). It also counts the visit for the
// near-linear test.
func (m *cdt) walkAcross(t int, p [2]float64) int {
	m.walkSteps++
	for i := range 3 {
		a, b := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
		if orient2d(m.pts[a], m.pts[b], p) < 0 { // p is right of the CCW edge (a,b): cross to n[i]
			if ne := m.tris[t].n[i]; ne >= 0 {
				return ne
			}
		}
	}
	return -1
}

// liveSeed returns the walk's starting triangle: the cached hint when still live, else the newest
// live triangle (nearest the most recent work), else the super-triangle.
func (m *cdt) liveSeed() int {
	if m.last >= 0 && m.last < len(m.dead) && !m.dead[m.last] {
		return m.last
	}
	for t := range slices.Backward(m.tris) {
		if !m.dead[t] {
			return t
		}
	}
	return 0
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
		for i := range 3 {
			ne := m.tris[t].n[i]
			if ne < 0 || c.in[ne] {
				continue
			}
			// Never grow the cavity across a constrained (frontier) edge: a point inserted after the
			// boundary is constrained must not engulf triangles on the far side of a wire and erase it
			// (OCCT's BRepMesh protects frontier edges the same way). con is empty while the boundary
			// itself is being inserted, so existing callers are unaffected.
			if a, b := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]; m.con[conKey(a, b)] > 0 {
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
		m.last = nt      // seed the next point-location walk from this fresh fan triangle (#1408)
	}
}

// cavityBoundary returns the directed CCW edges on the boundary of the cavity (edges whose other
// side is outside the cavity), each tagged with the outside neighbour triangle. It scans the cavity
// triangles in BFS order so the boundary (and the fan built from it) is deterministic.
func (m *cdt) cavityBoundary(c cavity) []cavityEdge {
	var bnd []cavityEdge
	for _, t := range c.order {
		for i := range 3 {
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
	m.touch(len(m.tris) - 1) // keep the incidence hint current once recovery has built it (#1409)
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
	for k := range 3 {
		if v := m.tris[s].v[k]; v != pp && v != q {
			d = v
		}
	}
	if d < 0 || orient2d(m.pts[c], m.pts[pp], m.pts[d]) <= 0 || orient2d(m.pts[c], m.pts[d], m.pts[q]) <= 0 {
		return false
	}
	nCP, nQC := m.tris[t].n[(i+2)%3], m.tris[t].n[(i+1)%3]
	nPD, nDQ := m.neighborAcross(s, pp, d), m.neighborAcross(s, d, q)
	m.rebuildFlip(t, s, c, pp, d, q, nPD, nCP, nDQ, nQC)
	return true
}

// rebuildFlip rewrites the two triangles of a Lawson flip to the new diagonal (c,d), relinks their
// outside neighbours, counts the flip, and refreshes the incidence hint for both (#1409). Split out of
// flip so each stays small; the parameters are the quad's vertices and the four outside neighbours.
func (m *cdt) rebuildFlip(t, s, c, pp, d, q, nPD, nCP, nDQ, nQC int) {
	m.tris[t] = cdtTri{v: [3]int{c, pp, d}, n: [3]int{nPD, s, nCP}}
	m.tris[s] = cdtTri{v: [3]int{c, d, q}, n: [3]int{nDQ, nQC, t}}
	m.relinkOpposite(nPD, pp, d, t)
	m.relinkOpposite(nCP, c, pp, t)
	m.relinkOpposite(nDQ, d, q, s)
	m.relinkOpposite(nQC, q, c, s)
	m.flipSteps++ // recovery + legalization flips (flip is called nowhere else) — instrumented for #1409
	m.touch(t)    // both reused triangles changed their vertex sets; refresh the incidence hint
	m.touch(s)
	// Queue the new diagonal for post-recovery legalization (#1604). During legalization itself the
	// entry is redundant (a fresh Lawson diagonal is locally Delaunay) and pops in O(1).
	m.pendingLegal = append(m.pendingLegal, [2]int{c, d})
}

// representatives maps each input point index to the inserted vertex that carries its coordinates:
// itself when unique, or the first earlier point with the same coords (the one insert kept — later
// duplicates are skipped, owning no triangle). Constraints are recovered between representatives so a
// duplicate boundary sample never references a vertex absent from the mesh (which the flip recovery
// would spin on forever, #1073).
func (m *cdt) representatives() []int {
	rep := make([]int, m.nsup)
	canon := map[[2]float64]int{}
	for i := 0; i < m.nsup; i++ {
		if c, ok := canon[m.pts[i]]; ok {
			rep[i] = c
		} else {
			canon[m.pts[i]] = i
			rep[i] = i
		}
	}
	return rep
}

// flipOneCrossing flips one flippable triangulation edge that properly crosses segment (a,b),
// returning whether it flipped one.
func (m *cdt) flipOneCrossing(a, b int) bool {
	pa, pb := m.pts[a], m.pts[b]
	for t := range m.tris {
		if m.dead[t] {
			continue
		}
		for i := range 3 {
			u, w := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
			if u == a || u == b || w == a || w == b {
				continue // edges touching an endpoint never cross the segment
			}
			if SegmentsCross(pa, pb, m.pts[u], m.pts[w]) && m.flip(t, i) {
				return true
			}
		}
	}
	return false
}

// SegmentsCross reports whether open segments p1p2 and p3p4 PROPERLY intersect: each segment's
// endpoints strictly straddle the other's line. An endpoint lying exactly ON the other segment
// (orient2d == 0) is a touch, not a crossing — the old `(d > 0) != (d > 0)` form lumped the exact
// zero in with the negative side, so recoverByFlips "resolved" a segment-through-vertex constraint
// by flipping an edge that merely touched it, wrecking the corridor without ever recovering the
// segment (#1604; the split-at-vertex path handles that degeneracy instead).
func SegmentsCross(p1, p2, p3, p4 [2]float64) bool {
	d1 := orient2d(p3, p4, p1)
	d2 := orient2d(p3, p4, p2)
	d3 := orient2d(p1, p2, p3)
	d4 := orient2d(p1, p2, p4)
	return d1*d2 < 0 && d3*d4 < 0
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
	for i := range 3 {
		s := m.tris[t].n[i]
		if s < 0 || m.dead[s] || visited[s] {
			continue
		}
		a, b := m.tris[t].v[(i+1)%3], m.tris[t].v[(i+2)%3]
		visited[s] = true
		inside[s] = inside[t] != (m.con[conKey(a, b)]%2 == 1) // odd wall multiplicity toggles; a doubled (pinched) wall cancels
		queue = append(queue, s)
	}
	return queue
}

func (m *cdt) hasSuper(t int) bool {
	return m.tris[t].v[0] >= m.nsup || m.tris[t].v[1] >= m.nsup || m.tris[t].v[2] >= m.nsup
}

// constrainedDelaunayRefinedChecked triangulates the domain bounded by loops with interior refinement,
// inserting the boundary in the OCCT BRepMesh order: the loop (frontier) points FIRST, then the
// constraints are recovered on that small point set (robust — no interior nodes fighting the flip
// recovery), then the interior Steiner points are inserted into the already-constrained mesh, where
// collectCavity protects the frontier edges. pts[:nFrontier] are the loop points (indexed by loops);
// pts[nFrontier:] are interior. This is the accurate path for a large face with several concave holes,
// where inserting everything at once (constrainedDelaunay) makes the constraint recovery leak and tear.
// It also reports the recovery status: the constraint endpoint pairs whose edge never recovered, and
// whether the deterministic earcut fallback was taken because the domain actually leaked across a missing
// boundary (#1410). See finalizeDomain for how a benign non-recovery (an outer seam bordering the excluded
// super region) keeps the higher-quality refined mesh while a genuine leak (a filled hole/notch) is
// replaced.
func constrainedDelaunayRefinedChecked(pts [][2]float64, loops [][]int, nFrontier int) ([][3]int, [][2]int, bool) {
	if len(pts) < 3 || nFrontier < 3 {
		return nil, nil, false
	}
	m := newCDT(pts)
	for i := range nFrontier {
		m.insert(i)
	}
	m.constrain(loops)
	for i := nFrontier; i < m.nsup; i++ {
		m.insert(i) // interior nodes, now respecting the frontier edges (see collectCavity)
	}
	return m.finalizeDomain(pts, loops)
}

// finalizeDomain extracts the inside domain and, when constraint recovery left a boundary edge
// unrealized, decides between the refined extraction and the deterministic earcut fallback by whether the
// domain ACTUALLY leaked (domainLeaked) — a missing boundary that filled a hole or concave notch — rather
// than assuming every non-recovery is harmful. A non-recovery that does not leak (the common holed-wall
// seam, whose far side is the excluded super region) keeps the higher-quality refined mesh. Returns the
// triangles, the unrecovered constraints (for the caller's diagnostic), and whether the fallback was used.
func (m *cdt) finalizeDomain(pts [][2]float64, loops [][]int) ([][3]int, [][2]int, bool) {
	ed := m.extractDomain()
	if len(m.unrecovered) == 0 {
		return ed, nil, false
	}
	// A budget bail means the boundary is non-simple: the extracted domain is untrustworthy (many
	// unrecovered constraints), so take the deterministic, bounded earcut fallback unconditionally
	// rather than probing domainLeaked on a mesh we already know is degenerate (see recoverFlipWork).
	if m.overBudget || m.domainLeaked(ed, loops) {
		return earcutFromLoops(pts, loops), m.unrecovered, true
	}
	return ed, m.unrecovered, false
}

// constrain recovers every loop edge as a hard constraint between the inserted representatives (see
// representatives) and records it in con. It builds the vertex→triangle incidence hint once up front so
// each recovery walks the constraint corridor in O(crossings), never rescanning the whole mesh (#1409).
func (m *cdt) constrain(loops [][]int) {
	rep := m.representatives()
	m.buildIncident()
	for _, lp := range loops {
		for k := range lp {
			m.insertConstraint(rep[lp[k]], rep[lp[(k+1)%len(lp)]])
		}
	}
}

// ConstrainedDelaunay triangulates the planar domain bounded by the given loops (the first/largest
// is the outer boundary, the rest are holes — but any nesting works: the flood toggles inside at
// each loop), returning CCW triangles inside the domain as index triples into pts. Every loop edge
// is a hard constraint; points not on a loop are interior Steiner points refining the mesh.
func ConstrainedDelaunay(pts [][2]float64, loops [][]int) [][3]int {
	tris, _, _ := constrainedDelaunayChecked(pts, loops)
	return tris
}

// constrainedDelaunayChecked is constrainedDelaunay plus the recovery status: it also returns the
// constraint endpoint pairs whose edge never recovered and whether the deterministic earcut fallback was
// taken because the domain leaked across a missing boundary (#1410, see finalizeDomain).
func constrainedDelaunayChecked(pts [][2]float64, loops [][]int) ([][3]int, [][2]int, bool) {
	if len(pts) < 3 {
		return nil, nil, false
	}
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	// A duplicate boundary sample (same coords as an earlier point) is SKIPPED by insert (it has no
	// strictly-bad triangle), so it owns no triangle; a constraint referencing it can be recovered by
	// neither the corridor walk (no incident triangle) nor the flips (which then spin to exhaustion —
	// the real O(T²) freeze on imported faces, #1073). constrain works between the inserted REPRESENTATIVES.
	m.constrain(loops)
	return m.finalizeDomain(pts, loops)
}

// constrainedTriangulationAll triangulates pts with every loop edge recovered as a hard constraint and
// returns ALL real (non-super) triangles WITHOUT the inside/outside flood. The covering-space periodic
// mesher (#1510) needs this: it triangulates three copies of a periodic chart and selects the canonical
// period itself (by triangle centroid + a material-region test), so the loop-parity flood — which assumes
// a single simply-bounded domain — cannot do the selection. Constraints still align the triangulation to
// the rim/mouth edges; the caller decides which triangles are kept.
func constrainedTriangulationAll(pts [][2]float64, loops [][]int) [][3]int {
	if len(pts) < 3 {
		return nil
	}
	m := newCDT(pts)
	for i := 0; i < m.nsup; i++ {
		m.insert(i)
	}
	m.constrain(loops)
	return m.extractAllNonSuper()
}

// extractAllNonSuper returns every live triangle that uses no super-triangle vertex, as CCW index
// triples into pts — the full constrained triangulation of the point set's convex hull, before any
// domain flood (see constrainedTriangulationAll).
func (m *cdt) extractAllNonSuper() [][3]int {
	var out [][3]int
	for t := range m.tris {
		if !m.dead[t] && !m.hasSuper(t) {
			out = append(out, [3]int{m.tris[t].v[0], m.tris[t].v[1], m.tris[t].v[2]})
		}
	}
	return out
}
