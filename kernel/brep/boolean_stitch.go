// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// stitch welds the kept sub-faces into a watertight B-rep: coincident vertices merge, one
// shared edge per undirected vertex pair, and a face per sub-face (outer loop oriented CCW
// about its normal, holes CW). The body is a solid when every edge is used exactly twice.
// Result faces carry their source face's lineage (K1a), so a face surviving the boolean
// keeps its reference key. Since ADR-0058 the assembly itself is the ONE unified radial
// stitch (curvedStitch): the planar fragments are welded and T-junction-conformed here, then
// lifted onto the shared curvedFace model and built by the same surface-agnostic
// weld/radial-sew/mint pass the curved boolean uses — tangent contacts resolve through the
// Weiler radial sew and pinched vertices split per disk, exactly as before, in shared code.
// The second result is true when a tangent/grazing contact was present (an edge used by more
// than two faces), so the caller can note whether that contact shipped as a valid manifold.
func stitch(faces []subFace, pass []curvedFace, prov []imprintSeg) (*topo.Body, bool, error) {
	if len(faces) == 0 && len(pass) == 0 {
		return nil, false, nil
	}
	w := newWelder3(planarStitchGrid)
	// Pass 1: weld every face's loops to vertex indices (collect the full vertex set).
	out := make([]builtFace, len(faces))
	for i, sf := range faces {
		rings := [][]int{w.ring(orientRing(sf.outer, sf.normal, true))}
		for _, h := range sf.holes {
			rings = append(rings, w.ring(orientRing(h, sf.normal, false)))
		}
		out[i] = builtFace{rings: rings, normal: sf.normal, fromB: sf.fromB, lineage: sf.lineage, exactHoles: sf.exactHoles}
	}
	// Pass 2: with all vertices known, split every loop edge at any welded vertex lying on
	// it — propagating each imprint split-point to the neighbour face sharing that edge, so
	// shared edges subdivide identically (eliminates cross-face T-junctions). The vertex set is
	// indexed once (a BoxTree over the welded points) so each edge queries only its own
	// neighbourhood instead of scanning every vertex — the O(V·E) scan was the boolean's top
	// profile cost after the stitch unification (ADR-0058).
	pass = conformSharedEdges(out, pass, w)
	if len(pass) == 0 {
		reorientFaces(out, w.points) // the legacy closed-set path, byte-identical
	} else {
		reorientFacesConsistent(out) // partial subset: classification is the orientation authority
	}
	tangent := hasTangentContact(collectEdgeUses(out))
	// The mixed dispatch assembles one shell from faces built by DIFFERENT machinery — the
	// polygonal split, the exact-frame (u,v) trims, the ruled-wall trims and the whole pass-through
	// faces — each wound by its own construction. reorientFacesConsistent above two-colours only
	// the polygonal half, so a shell that is watertight and manifold could still have shared edges
	// traversed the same way by both faces, which Validate rejects. curvedReorient is the same
	// flood fill over the WHOLE set, and is a no-op on already-consistent input by construction,
	// so the paths that were already consistent are unchanged (#1818, #3459).
	all := curvedReorient(append(builtFacesToCurved(out, w.points), pass...))
	body := curvedStitchNamed(all, planarStitchNaming(prov, w, out))
	return body, tangent, nil
}

// planarStitchNaming is the planar boolean's ADR-0043 naming policy for the unified stitch: every
// intersection edge named by its generating parent face pair (disambiguated by rank along the parents'
// intersection line), every intersection vertex by its meeting faces, originals keeping ordinal keys —
// the same names the retired planar assemble minted. The curved relineage is off: these names are
// already build-order-independent.
func planarStitchNaming(prov []imprintSeg, w *welder3, faces []builtFace) stitchNaming {
	return stitchNaming{
		edges: func(groups []edgeGroup, verts []math.Point3) []topo.Lineage {
			return planarEdgeLineages(groups, verts, prov)
		},
		vertex: planarVertexNamer(prov, w, faces),
	}
}

// planarEdgeLineages resolves each stitch group's edge lineage from the imprint provenance
// (nameEdgeGroups), assigning the unparented ordinal fallbacks in sorted-pair order — the retired
// assemble's deterministic order.
func planarEdgeLineages(groups []edgeGroup, verts []math.Point3, prov []imprintSeg) []topo.Lineage {
	named := nameEdgeGroups(groups, verts, prov)
	order := make([]int, len(groups))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return pairLess(groups[order[a]].pair, groups[order[b]].pair) })
	out := make([]topo.Lineage, len(groups))
	idx := 0
	for _, gi := range order {
		out[gi] = edgeGroupLineage(&named[gi], &idx)
	}
	return out
}

// pairLess orders vertex pairs ascending (the retired sortedPairKeys order).
func pairLess(a, b [2]int) bool {
	if a[0] != b[0] {
		return a[0] < b[0]
	}
	return a[1] < b[1]
}

// planarVertexNamer names each shared vertex by the retired assemble's vertexLineages verdict —
// intersection vertices by their meeting faces, original corners by ordinal — resolved through the
// planar welder (the coordinates handed to the unified stitch are exactly its welded points).
func planarVertexNamer(prov []imprintSeg, w *welder3, faces []builtFace) func(math.Point3) topo.Lineage {
	vlin := vertexLineages(w.points, faces, prov)
	extra := 0
	return func(p math.Point3) topo.Lineage {
		if i := w.add(p); i < len(vlin) {
			return vlin[i]
		}
		// A pass-through face's vertex (per-face dispatch, ADR-0058): not in the planar weld set, so
		// it gets a deterministic ordinal in mint order.
		lin := topo.NewLineage(topo.Tok("brep", "vx", extra))
		extra++
		return lin
	}
}

// builtFacesToCurved lifts the welded, T-junction-conformed planar fragments onto the unified
// curvedFace model: each face's plane through its outer ring's centroid along its
// (reorient-consistent) normal, its rings as exact straight-edged loops (loopEdge.v0 carries the
// welded corner coordinates bit-for-bit). Fragment lineage is carried (K1a); a fragment without one
// gets the ordinal fallback, as the retired planar assemble did.
func builtFacesToCurved(faces []builtFace, verts []math.Point3) []curvedFace {
	out := make([]curvedFace, len(faces))
	for i, f := range faces {
		rings := ringsPoints(f.rings, verts)
		pl, _ := geom.NewPlane(ringCentroid(rings), f.normal)
		cf := planarFaceFromRings(pl, rings, faceLineage(f, i))
		for _, h := range f.exactHoles {
			cf.loops = append(cf.loops, holeWoundAgainst(h, f.normal)) // detached curved holes re-attach as exact inner loops
		}
		out[i] = cf
	}
	return out
}

// ringsPoints resolves welded vertex-index rings back to their 3D points.
func ringsPoints(rings [][]int, verts []math.Point3) [][]math.Point3 {
	out := make([][]math.Point3, len(rings))
	for ri, r := range rings {
		ring := make([]math.Point3, len(r))
		for j, vi := range r {
			ring[j] = verts[vi]
		}
		out[ri] = ring
	}
	return out
}

// ringCentroid is the outer ring's centroid (the plane anchor), or the origin for a degenerate
// ringless face — which mints no loops and vanishes downstream.
func ringCentroid(rings [][]math.Point3) math.Point3 {
	if len(rings) == 0 || len(rings[0]) == 0 {
		return math.Point3{}
	}
	return centroid3(rings[0])
}

// hasTangentContact reports whether any vertex pair is used by more than two face half-edges — a
// tangent/grazing contact between the operands the radial-edge sew resolved into manifold
// edge-groups. The caller notes whether that contact shipped as a valid manifold.
func hasTangentContact(uses map[[2]int][]loopEdgeUse) bool {
	for _, u := range uses {
		if len(u) > 2 {
			return true
		}
	}
	return false
}

// splitRingTJunctions inserts, into each edge of the ring, any other vertex that lies in
// its interior (sorted along the edge) — so a vertex that is a corner of a neighbour face
// also subdivides this face's coincident edge.
func splitRingTJunctions(ring []int, w *welder3, tree *geom.BoxTree) []int {
	n := len(ring)
	out := make([]int, 0, n)
	for i := range n {
		a, b := ring[i], ring[(i+1)%n]
		out = append(out, a)
		mids := verticesOnSegment(a, b, ring, w, tree)
		out = append(out, mids...)
	}
	return out
}

// newTJPointTree indexes the welded vertex set for the T-junction pass's segment-neighbourhood
// queries (each point as a degenerate box).
func newTJPointTree(points []math.Point3) *geom.BoxTree {
	boxes := make([]math.Box, len(points))
	for i, p := range points {
		boxes[i] = math.NewBox(p, p)
	}
	return geom.NewBoxTree(boxes)
}

// segHit is a vertex found lying on a segment at parameter t along it.
type segHit struct {
	t float64
	v int
}

// verticesOnSegment returns the vertices (excluding the ring's own) lying strictly on the
// segment a→b, ordered by parameter along it.
func verticesOnSegment(a, b int, ring []int, w *welder3, tree *geom.BoxTree) []int {
	pa := w.points[a]
	ab := pa.VectorTo(w.points[b])
	lenSq := ab.LengthSquared()
	if lenSq == 0 {
		return nil
	}
	hits := collectSegHits(pa, ab, lenSq, ringSet(ring), w, tree)
	sort.Slice(hits, func(i, j int) bool { return hits[i].t < hits[j].t })
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.v
	}
	return out
}

// ringSet is the set of vertex indices used by a ring.
func ringSet(ring []int) map[int]bool {
	s := make(map[int]bool, len(ring))
	for _, v := range ring {
		s[v] = true
	}
	return s
}

// collectSegHits gathers the off-ring vertices that lie strictly interior to segment pa+ab
// (within the welder's own weld grid, the same coincidence scale the vertices merged on). The
// BoxTree query over the grid-inflated segment box is a strict SUPERSET of every vertex the exact
// predicate can accept (a hit lies within grid of a point on the segment, so inside the inflated
// box); candidates are then evaluated in ascending index order — the retired full scan's order —
// so the hit list, and therefore the (unstable) t-sort downstream, is byte-identical.
func collectSegHits(pa math.Point3, ab math.Vector3, lenSq float64, onRing map[int]bool, w *welder3, tree *geom.BoxTree) []segHit {
	cands := segCandidates(pa, pa.TranslateBy(ab), w.grid, tree)
	var hits []segHit
	for _, c := range cands {
		if onRing[c] {
			continue
		}
		t := pa.VectorTo(w.points[c]).Dot(ab) / lenSq
		if t <= 1e-7 || t >= 1-1e-7 { // tol:parametric — edge parameter t in [0,1]
			continue
		}
		if float64(pa.TranslateBy(ab.Scale(t)).DistanceTo(w.points[c])) < w.grid {
			hits = append(hits, segHit{t, c})
		}
	}
	return hits
}

// segCandidates returns the indexed vertices inside the segment's grid-inflated bounding box, in
// ascending index order (the deterministic evaluation order the exact pass preserves).
func segCandidates(pa, pb math.Point3, grid float64, tree *geom.BoxTree) []int {
	g := math.Scalar(grid)
	box := math.NewBox(pa, pa).ExtendPoint(pb)
	box = math.NewBox(box.Min.TranslateBy(math.V3(-g, -g, -g)), box.Max.TranslateBy(math.V3(g, g, g)))
	var cands []int
	tree.Query(box, func(i int) bool {
		cands = append(cands, i)
		return false // collect every candidate (true would stop the query early)
	})
	sort.Ints(cands)
	return cands
}

// builtFace is a welded sub-face ready for the unified stitch: its loop rings (vertex indices, outer
// first), its outward normal (the plane is re-derived from ring + normal at conversion), and the
// source lineage to carry onto the result face (K1a).
type builtFace struct {
	rings      [][]int
	normal     math.Vector3
	fromB      bool
	lineage    topo.Lineage
	exactHoles []curvedLoop // curved hole loops re-attached exactly after the polygonal split (ADR-0058)
}

// faceLineage uses the source face's carried lineage when present (K1a reference-key
// survival), falling back to a synthesized one for any face without a source.
func faceLineage(f builtFace, fi int) topo.Lineage {
	if len(f.lineage.Tokens()) > 0 {
		return f.lineage
	}
	return topo.NewLineage(topo.Tok("brep", "face", fi))
}

// allEdgesPaired reports whether every undirected edge is used exactly twice — the
// combinatorial test for a closed (solid) shell. Used by the drill paths, which key edges by
// vertex pair directly (no tangent-contact resolution needed for a single drilled hole).
func allEdgesPaired(edgeUse map[[2]int]int) bool {
	for _, c := range edgeUse {
		if c != 2 {
			return false
		}
	}
	return true
}

// loopSpec builds a face loop (outer or inner) from a ring of vertex indices, resolving each
// directed pair to its shared edge (reversed when traversed high→low).
func loopSpec(outer bool, ring []int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, len(ring))
	for i := range ring {
		a, b := ring[i], ring[(i+1)%len(ring)]
		uses[i] = topo.Use{Edge: edges[canonEdge(a, b)], Reversed: a > b}
	}
	if outer {
		return topo.OuterLoop(uses...)
	}
	return topo.InnerLoop(uses...)
}

// buildEdges creates one shared topo edge per undirected vertex pair (sorted for stable
// lineage).
func buildEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, edgeUse map[[2]int]int) map[[2]int]*topo.Edge {
	keys := make([][2]int, 0, len(edgeUse))
	for k := range edgeUse {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	edges := make(map[[2]int]*topo.Edge, len(keys))
	for i, k := range keys {
		edges[k] = bld.AddEdge(geom.NewLineSegment(verts[k[0]], verts[k[1]]), tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok("brep", "edge", i)))
	}
	return edges
}

// planarStitchGrid is the weld grid of the PLANAR boolean family (stitch, split-ring signatures,
// T-junction hits, drill assembly) and stays deliberately ABSOLUTE, unlike the SSI-fed curved
// stitch (which scales with the model, geom.Resolution.Stitch, #1602). The planar path's producers
// bound their noise absolutely, not proportionally to part size: exact plane–plane arithmetic
// errs by ~1e-16·|coordinate| (2e-14 even at 2 m), and triangle-soup CSG facets legitimately carry
// slivers just above 1e-6 that a coarser, size-scaled grid would collapse into degenerate rings —
// TestNopCapScrewCSG catches exactly that over-merge. See also the arrange2d arrTol calibration note.
const planarStitchGrid = 1e-6 // tol:calibrated — planar stitch weld grid (see arrange2d arrTol)

// welder3 merges 3D points within its weld grid onto a shared index list. The grid is
// model-relative (geom.Resolution.Stitch of the geometry being welded, ADR-0042): the retired
// absolute 1e-6 grid was calibrated for ~1 cm parts and left independently computed copies of the
// same seam point unmerged as soon as their producer noise (1e-7 of the extent) outgrew it (#1602).
type welder3 struct {
	grid   float64
	index  map[[3]int64]int
	points []math.Point3
}

func newWelder3(grid float64) *welder3 { return &welder3{grid: grid, index: map[[3]int64]int{}} }

// add returns the index of the welded vertex for p, merging it with any existing vertex within
// the weld grid. The point is hashed to a grid cell, but the 26 neighbouring cells are also searched:
// two coincident points either side of a cell boundary hash to different cells, so a cell-exact
// lookup would leave them unmerged — the failure that shredded dense self-proximate geometry
// (a fine-pitch coil-join) into unpaired, coincident open edges (#879).
func (w *welder3) add(p math.Point3) int {
	cx := int64(stdmath.Round(p.X / w.grid))
	cy := int64(stdmath.Round(p.Y / w.grid))
	cz := int64(stdmath.Round(p.Z / w.grid))
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			for dz := int64(-1); dz <= 1; dz++ {
				if i, ok := w.index[[3]int64{cx + dx, cy + dy, cz + dz}]; ok && w.points[i].DistanceTo(p) <= w.grid {
					return i
				}
			}
		}
	}
	w.index[[3]int64{cx, cy, cz}] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

// ring welds a 3D loop to vertex indices, dropping consecutive duplicates.
func (w *welder3) ring(loop []math.Point3) []int {
	var out []int
	for _, p := range loop {
		i := w.add(p)
		if len(out) == 0 || out[len(out)-1] != i {
			out = append(out, i)
		}
	}
	if len(out) > 1 && out[0] == out[len(out)-1] {
		out = out[:len(out)-1]
	}
	return out
}

// orientRing returns the loop wound so its Newell normal points along (outer) or against
// (hole) the face normal.
func orientRing(loop []math.Point3, normal math.Vector3, outer bool) []math.Point3 {
	aligned := newell3(loop).Dot(normal) > 0
	if aligned == outer {
		return loop
	}
	return reverseRing(loop)
}

// newell3 returns a 3D loop's (unnormalized) Newell normal.
func newell3(loop []math.Point3) math.Vector3 {
	var nx, ny, nz float64
	n := len(loop)
	for i := range n {
		c, d := loop[i], loop[(i+1)%n]
		nx += (c.Y - d.Y) * (c.Z + d.Z)
		ny += (c.Z - d.Z) * (c.X + d.X)
		nz += (c.X - d.X) * (c.Y + d.Y)
	}
	return math.V3(nx, ny, nz)
}

func centroid3(loop []math.Point3) math.Point3 {
	var sx, sy, sz float64
	for _, p := range loop {
		sx, sy, sz = sx+p.X, sy+p.Y, sz+p.Z
	}
	n := float64(len(loop))
	return math.P3(sx/n, sy/n, sz/n)
}

// conformSharedEdges runs the T-junction pass over the polygonal rings AND the pass-through faces
// against one welded vertex set, so every shared edge subdivides identically on both faces.
func conformSharedEdges(out []builtFace, pass []curvedFace, w *welder3) []curvedFace {
	weldPassVertices(pass, w)
	tjTree := newTJPointTree(w.points)
	for fi := range out {
		for ri := range out[fi].rings {
			out[fi].rings[ri] = splitRingTJunctions(out[fi].rings[ri], w, tjTree)
		}
	}
	return splitPassTJunctions(pass, w, tjTree)
}

// weldPassVertices adds every pass-through face's loop vertices to the planar weld set BEFORE the
// T-junction pass, so a polygonal fragment's edge splits where an exact-frame fragment's vertex lands on
// it — a block's side face at the two corners of the cap's hole its coplanar bottom face was trimmed
// against (ADR-0060). Without it the pass faces were conformed to nothing and the polygonal edge stayed
// whole across two neighbours.
func weldPassVertices(pass []curvedFace, w *welder3) {
	for _, f := range pass {
		for _, l := range f.loops {
			for _, e := range l.edges {
				w.add(e.start())
			}
		}
	}
}

// splitPassTJunctions is splitRingTJunctions for the pass-through faces' STRAIGHT edges: each is cut at
// every welded vertex lying strictly inside it, so the shared edge subdivides identically on both faces.
// A conic edge needs no pass — the exact-frame chart already injected every incidence on it as a vertex.
func splitPassTJunctions(pass []curvedFace, w *welder3, tree *geom.BoxTree) []curvedFace {
	out := make([]curvedFace, len(pass))
	for i, f := range pass {
		loops := make([]curvedLoop, len(f.loops))
		for li, l := range f.loops {
			own := w.ring(loopStarts(l))
			var edges []loopEdge
			for _, e := range l.edges {
				edges = append(edges, splitStraightEdgeAt(e, own, w, tree)...)
			}
			loops[li] = curvedLoop{edges: edges}
		}
		f.loops = loops
		out[i] = f
	}
	return out
}

// loopStarts is a loop's vertex chain: each edge's start point.
func loopStarts(l curvedLoop) []math.Point3 {
	pts := make([]math.Point3, 0, len(l.edges))
	for _, e := range l.edges {
		pts = append(pts, e.start())
	}
	return pts
}

// splitStraightEdgeAt returns an edge cut at the welded vertices strictly inside it (in order along
// it), or the edge itself when none lie on it. A straight edge takes the ring pass's exact test; a
// conic edge takes the vertices its sampled box holds that invert onto its curve within the weld.
func splitStraightEdgeAt(e loopEdge, own []int, w *welder3, tree *geom.BoxTree) []loopEdge {
	if !geom.IsStraightCurve(e.curve) {
		return splitEdgeAtPoints(e, offEdgeVertices(e, own, w, tree), geom.ResolutionForBox(edgeSampleBox(e)))
	}
	a, b := w.add(e.start()), w.add(e.end())
	mids := verticesOnSegment(a, b, own, w, tree)
	if len(mids) == 0 {
		return []loopEdge{e}
	}
	chain := append([]math.Point3{e.start()}, pointsAt(mids, w)...)
	chain = append(chain, e.end())
	out := make([]loopEdge, 0, len(chain)-1)
	for i := 1; i < len(chain); i++ {
		out = append(out, loopEdge{curve: geom.NewLineSegment(chain[i-1], chain[i]), t0: 0, t1: 1})
	}
	return out
}

// offEdgeVertices returns the welded vertices inside a conic edge's sampled box that are not the
// loop's own, as candidate split points.
func offEdgeVertices(e loopEdge, own []int, w *welder3, tree *geom.BoxTree) []math.Point3 {
	onRing := ringSet(own)
	var pts []math.Point3
	tree.Query(edgeSampleBox(e), func(v int) bool {
		if !onRing[v] {
			pts = append(pts, w.points[v])
		}
		return false
	})
	return pts
}

// edgeSampleBox bounds an edge by its samples, padded by the welder grid.
func edgeSampleBox(e loopEdge) math.Box {
	box := math.EmptyBox()
	for k := 0; k <= imprintSampleCount; k++ {
		box = box.ExtendPoint(e.curve.PointAt(e.t0 + (e.t1-e.t0)*float64(k)/imprintSampleCount))
	}
	return paddedBox(box, probeBoxPadRel)
}

// pointsAt resolves welded vertex indices to their coordinates.
func pointsAt(idx []int, w *welder3) []math.Point3 {
	pts := make([]math.Point3, len(idx))
	for i, v := range idx {
		pts[i] = w.points[v]
	}
	return pts
}

// holeWoundAgainst returns the exact hole wound CLOCKWISE about the face's outward normal — the sense
// every hole of a consistently wound face has (orientRing gives the polygonal holes exactly that). A
// re-attached hole kept the traversal its SOURCE face stored, and a source wound the other way about
// its own normal left the hole running with the outer ring, so the rim it shares with the boss's wall
// was traversed the same way by both faces (ADR-0060).
func holeWoundAgainst(h curvedLoop, normal math.Vector3) curvedLoop {
	var pts []math.Point3
	for _, e := range h.edges {
		for k := 0; k < imprintSampleCount; k++ {
			pts = append(pts, e.curve.PointAt(e.t0+(e.t1-e.t0)*float64(k)/imprintSampleCount))
		}
	}
	if len(pts) < 3 {
		return h
	}
	area := 0.0
	for k := range pts {
		a, b := pts[0].VectorTo(pts[k]), pts[0].VectorTo(pts[(k+1)%len(pts)])
		area += float64(a.Cross(b).Dot(normal))
	}
	if area > 0 {
		return reverseCurvedLoop(h)
	}
	return h
}
