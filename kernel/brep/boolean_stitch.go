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
// keeps its reference key; edges/vertices still get fresh lineage (edge-key survival is a
// follow-up, as edges are routinely split by the operation).
// The second result is true when a tangent/grazing contact was resolved (an edge used by more than
// two faces), so the caller can note whether that contact shipped as a valid manifold.
func stitch(faces []subFace, prov []imprintSeg) (*topo.Body, bool, error) {
	if len(faces) == 0 {
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
		surf, _ := geom.NewPlane(centroid3(sf.outer), sf.normal)
		out[i] = builtFace{rings: rings, surf: surf, normal: sf.normal, fromB: sf.fromB, lineage: sf.lineage}
	}
	// Pass 2: with all vertices known, split every loop edge at any welded vertex lying on
	// it — propagating each imprint split-point to the neighbour face sharing that edge, so
	// shared edges subdivide identically (eliminates cross-face T-junctions).
	for fi := range out {
		for ri := range out[fi].rings {
			out[fi].rings[ri] = splitRingTJunctions(out[fi].rings[ri], w)
		}
	}
	reorientFaces(out, w.points)
	body, tangent := assemble(w.points, out, prov)
	return body, tangent, nil
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
func splitRingTJunctions(ring []int, w *welder3) []int {
	n := len(ring)
	out := make([]int, 0, n)
	for i := range n {
		a, b := ring[i], ring[(i+1)%n]
		out = append(out, a)
		mids := verticesOnSegment(a, b, ring, w)
		out = append(out, mids...)
	}
	return out
}

// segHit is a vertex found lying on a segment at parameter t along it.
type segHit struct {
	t float64
	v int
}

// verticesOnSegment returns the vertices (excluding the ring's own) lying strictly on the
// segment a→b, ordered by parameter along it.
func verticesOnSegment(a, b int, ring []int, w *welder3) []int {
	pa := w.points[a]
	ab := pa.VectorTo(w.points[b])
	lenSq := ab.LengthSquared()
	if lenSq == 0 {
		return nil
	}
	hits := collectSegHits(pa, ab, lenSq, ringSet(ring), w)
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
// (within the welder's own weld grid, the same coincidence scale the vertices merged on).
func collectSegHits(pa math.Point3, ab math.Vector3, lenSq float64, onRing map[int]bool, w *welder3) []segHit {
	var hits []segHit
	for c := range w.points {
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

// builtFace is a welded sub-face ready for assembly: its loop rings (vertex indices, outer
// first), its planar surface, and the source lineage to carry onto the result face (K1a).
type builtFace struct {
	rings   [][]int
	surf    geom.Plane
	normal  math.Vector3
	fromB   bool
	lineage topo.Lineage
}

// assemble builds the topo body from welded vertices and per-face loop rings. The Weiler
// radial-edge sew (radialSew) resolves every directed loop edge to a shared topo edge: the common
// case is two uses per undirected vertex pair (one shared edge). Where coincident geometry leaves a
// pair used by MORE than twice — a tangent/grazing contact between the two operands — the uses are
// split into manifold edge-groups by radial order around the edge, and pinched vertices are cut into
// per-disk coincident duplicates, so a tangent union stays a valid manifold solid (M20-F01, #1726).
// mintEntities then names and builds the resulting topo edges (ADR-0043 provenance).
func assemble(verts []math.Point3, faces []builtFace, prov []imprintSeg) (*topo.Body, bool) {
	uses := collectEdgeUses(faces)
	bld := topo.NewBuilder(allUsesPaired(uses), topo.NewLineage(topo.Tok("brep", "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	vlin := vertexLineages(verts, faces, prov)
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, vlin[i])
	}
	useEdge := mintEntities(bld, verts, tv, radialSew(verts, uses, planarFaceDir(faces)), prov)
	for fi, f := range faces {
		specs := make([]topo.LoopSpec, len(f.rings))
		for ri, r := range f.rings {
			specs[ri] = loopSpecResolved(ri == 0, r, fi, ri, useEdge)
		}
		bld.AddFace(f.surf, faceLineage(f, fi), specs...)
	}
	return bld.Build(), hasTangentContact(uses)
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
