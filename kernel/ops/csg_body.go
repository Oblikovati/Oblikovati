// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// booleanInputQuality is the faceting the BSP-tree CSG meshes its operands at. It keeps the
// display chord tolerance but DISABLES the angular-deflection refinement (a huge angle bound
// → chord-only), because the faceted CSG fallback is numerically fragile: feeding it the
// finer angle-bounded display mesh (e.g. 32-facet small cylinders) makes BSP plane splits
// leave hairline cracks (a few unshared boundary edges). The boolean is a geometry op with
// its own robustness ceiling, separate from how smoothly we DISPLAY analytic surfaces — so it
// pins its input to the proven-robust chord-only resolution. (Analytic faces, including
// modeled threads and drilled holes, still render at full DefaultQuality smoothness.)
func booleanInputQuality() Quality {
	return Quality{ChordTolerance: DefaultQuality().ChordTolerance, AngleTolerance: stdmath.Pi}
}

// Facet converts a body with analytic curved faces (a cylinder, cone, surface of revolution)
// into an all-planar, watertight triangle-cage B-rep — its tessellation welded back into topology.
// The exact planar B-rep boolean hangs on a full periodic curved face, so a curved operand must be
// faceted before ops.Boolean (Oblikovati/Oblikovati#129); this is the general fallback for bodies
// the feature layer's analytic-cylinder re-faceter doesn't special-case. Returns nil when empty.
func Facet(b *topo.Body, feat string) *topo.Body {
	return trianglesToBody(bodyTriangles(b), feat)
}

// bodyTriangles returns a body's tessellation as CSG triangles, each oriented
// outward. The BSP-tree CSG depends on globally consistent outward winding, but
// TessellateBody does not guarantee it for curved faces (a cylinder's side
// triangles can wind either way — the same reason meshGeometryProperties
// re-orients before its volume sum). Trusting the raw winding silently breaks any
// boolean whose minuend is a curved body: the BSP misclassifies inside/outside and
// subtracts nothing (cylinder − tool returned the uncut cylinder). We fix the
// winding here with the per-vertex shading normals, which point outward.
func bodyTriangles(b *topo.Body) []tri {
	mesh, _ := TessellateBody(b, booleanInputQuality())
	var out []tri
	for i := 0; i+2 < len(mesh.Indices); i += 3 {
		ia, ib, ic := mesh.Indices[i], mesh.Indices[i+1], mesh.Indices[i+2]
		a, bb, c := mesh.Positions[ia], mesh.Positions[ib], mesh.Positions[ic]
		if outwardRef(mesh, ia, ib, ic).Dot(a.VectorTo(bb).Cross(a.VectorTo(c))) < 0 {
			bb, c = c, bb // flip to outward winding
		}
		if t, ok := newTri(a, bb, c); ok {
			out = append(out, t)
		}
	}
	return out
}

// trianglesToBody welds CSG output triangles into a watertight B-rep: coincident
// vertices merge, T-junctions are split out so the cage is combinatorially closed, and
// the welded triangle cage becomes a body. Returns nil when the result is empty.
func trianglesToBody(tris []tri, feat string) *topo.Body {
	// One model-relative resolution for the whole triangle set (ADR-0042): a tight
	// vertex weld and a wider on-line tolerance, both scaling with the operand size,
	// so a sub-µm part is no longer welded out of existence while a finely-detailed
	// large part is not over-merged.
	res := resolutionForTris(tris)
	verts, faces := weldTriangles(tris, res.Weld())
	faces = dedupTriangles(faces)
	if len(faces) == 0 {
		return nil
	}
	verts, faces = removeTJunctions(verts, faces, res.Plane())
	return cageToBody(verts, dropDegenerate(faces), feat)
}

// dedupTriangles cancels coincident triangles produced where coplanar faces of the two
// operands overlap: a triangle and its reverse (opposite orientation) annihilate (an
// internal face); identical-orientation duplicates collapse to one. This keeps each
// surface patch represented exactly once so the weld is 2-manifold.
func dedupTriangles(faces [][3]int) [][3]int {
	type bal struct {
		fwd, rev int
		face     [3]int
	}
	seen := map[[3]int]*bal{}
	order := [][3]int{}
	for _, f := range faces {
		key, reversed := sortedTri(f)
		b := seen[key]
		if b == nil {
			b = &bal{face: f}
			seen[key] = b
			order = append(order, key)
		}
		if reversed {
			b.rev++
		} else {
			b.fwd++
		}
	}
	var out [][3]int
	for _, key := range order {
		b := seen[key]
		if net := b.fwd - b.rev; net != 0 { // surviving orientation wins; equal ⇒ cancelled
			out = append(out, b.face)
		}
	}
	return out
}

// sortedTri returns a triangle's canonical (ascending) vertex key and whether the given
// winding runs opposite the canonical cyclic order (i.e. it is the reversed face).
func sortedTri(f [3]int) ([3]int, bool) {
	a, b, c := f[0], f[1], f[2]
	even := (a < b && b < c) || (b < c && c < a) || (c < a && a < b) // cyclic-sorted ⇒ same orientation
	key := f
	if key[1] < key[0] {
		key[0], key[1] = key[1], key[0]
	}
	if key[2] < key[1] {
		key[1], key[2] = key[2], key[1]
	}
	if key[1] < key[0] {
		key[0], key[1] = key[1], key[0]
	}
	return key, !even
}

// weldTriangles merges coincident triangle corners onto a shared vertex list, dropping
// triangles that collapse to a degenerate (a repeated corner).
func weldTriangles(tris []tri, grid float64) ([]math.Point3, [][3]int) {
	index := map[[3]int64]int{}
	var verts []math.Point3
	weld := func(p math.Point3) int {
		k := [3]int64{quantize(p.X, grid), quantize(p.Y, grid), quantize(p.Z, grid)}
		if i, ok := index[k]; ok {
			return i
		}
		index[k] = len(verts)
		verts = append(verts, p)
		return len(verts) - 1
	}
	var faces [][3]int
	for _, t := range tris {
		a, b, c := weld(t.a), weld(t.b), weld(t.c)
		if a != b && b != c && a != c {
			faces = append(faces, [3]int{a, b, c})
		}
	}
	return verts, faces
}

// quantize snaps a coordinate to a weld grid (database units), so points within a
// grid cell collapse to one vertex. The grid is the model-relative resolution the
// caller derives (ADR-0042), not a fixed constant.
func quantize(v, grid float64) int64 { return int64(stdmath.Round(v / grid)) }

// removeTJunctions eliminates T-junctions so every undirected edge of the cage is shared by exactly two
// triangles (the prerequisite for a closed solid). For each triangle it collects the mesh vertices lying
// on the interior of its three edges and re-fans the triangle through them in ONE pass — no cascade and
// no full-vertex scan: a vertex spatial hash (vertexGrid) makes the per-edge lookup local, so the pass is
// near-linear in the triangle count and needs NO face budget. The cage always comes back combinatorially
// closed, however many facets a curved wall contributed — this is acceptance #3 of the curved-boolean
// umbrella (Oblikovati/Oblikovati#1336, #1320 #3), replacing the bounded O(faces·verts) cascade of
// M20-F01 #470 that BAILED above tjunctionFaceBudget and left the cage open (the chained-bore drift the
// guard catches). Two triangles sharing an edge collect the same on-edge points off the same segment, so
// their subdivisions match and weld. Returns the (possibly grown) vertex list — a subdivided triangle
// gains an interior centroid hub — alongside the new faces.
func removeTJunctions(verts []math.Point3, faces [][3]int, lineTol float64) ([]math.Point3, [][3]int) {
	grid := newVertexGrid(verts, meanEdgeLength(verts, faces))
	out := make([][3]int, 0, len(faces))
	for _, f := range faces {
		poly := edgeSubdividedBoundary(verts, f, grid, lineTol)
		if len(poly) == 3 {
			out = append(out, f) // no T-junction on this triangle: keep it as-is
			continue
		}
		hub := len(verts)
		verts = append(verts, triangleCentroid(verts, f))
		out = append(out, fanFromHub(hub, poly)...)
	}
	return verts, out
}

// edgeSubdividedBoundary walks the triangle boundary, emitting each corner followed by the mesh vertices
// lying on the interior of the edge leaving it (ordered along that edge). With no on-edge point it returns
// the three corners unchanged.
func edgeSubdividedBoundary(verts []math.Point3, f [3]int, grid *vertexGrid, lineTol float64) []int {
	poly := make([]int, 0, 3)
	for e := 0; e < 3; e++ {
		p, q := f[e], f[(e+1)%3]
		poly = append(poly, p)
		poly = append(poly, edgeInteriorPoints(verts, p, q, grid, lineTol)...)
	}
	return poly
}

// edgeInteriorPoints returns the mesh vertices on the interior of segment p→q, ordered from p to q. Only
// the vertices the grid reports near the segment are tested, so the cost is local, not O(verts).
func edgeInteriorPoints(verts []math.Point3, p, q int, grid *vertexGrid, lineTol float64) []int {
	a, b := verts[p], verts[q]
	ab := a.VectorTo(b)
	type onPt struct {
		idx int
		t   float64
	}
	var hits []onPt
	for _, ci := range grid.near(a, b) {
		if ci == p || ci == q {
			continue
		}
		if onSegment(verts[ci], a, b, lineTol) {
			hits = append(hits, onPt{ci, ab.Dot(a.VectorTo(verts[ci]))})
		}
	}
	sort.Slice(hits, func(i, j int) bool { return hits[i].t < hits[j].t })
	out := make([]int, len(hits))
	for i, h := range hits {
		out[i] = h.idx
	}
	return out
}

// fanFromHub triangulates a closed boundary polygon as a fan from an interior hub vertex. Because the hub
// is strictly inside, every fan triangle is non-degenerate even where the boundary carries collinear
// on-edge points — a corner fan would emit zero-area slivers along the corner's two edges. Each boundary
// edge appears once and each hub spoke twice, so the patch is 2-manifold and welds to its neighbours along
// the shared (identically subdivided) boundary edges.
func fanFromHub(hub int, poly []int) [][3]int {
	out := make([][3]int, len(poly))
	for i := range poly {
		out[i] = [3]int{hub, poly[i], poly[(i+1)%len(poly)]}
	}
	return out
}

// triangleCentroid returns the average of a triangle's three corners — strictly interior, on the
// triangle's plane, so using it as a fan hub adds no T-junctions and does not change the surface.
func triangleCentroid(verts []math.Point3, f [3]int) math.Point3 {
	a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
	return math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
}

// meanEdgeLength is the average triangle-edge length, used to size the vertexGrid cell so a typical edge
// spans only a handful of cells. Returns 1 for an empty set (an unused grid).
func meanEdgeLength(verts []math.Point3, faces [][3]int) float64 {
	if len(faces) == 0 {
		return 1
	}
	sum := 0.0
	for _, f := range faces {
		sum += verts[f[0]].DistanceTo(verts[f[1]]) +
			verts[f[1]].DistanceTo(verts[f[2]]) +
			verts[f[2]].DistanceTo(verts[f[0]])
	}
	return sum / float64(3*len(faces))
}

// vertexGrid is a uniform spatial hash over vertex positions: it answers "which vertices lie near this
// segment" in time proportional to the segment's cell span, not the vertex count, so T-junction removal is
// near-linear rather than O(faces·verts).
type vertexGrid struct {
	cell float64
	bins map[[3]int64][]int
}

// newVertexGrid hashes every vertex into a cell of side cell (clamped positive).
func newVertexGrid(verts []math.Point3, cell float64) *vertexGrid {
	if cell <= 0 {
		cell = 1
	}
	g := &vertexGrid{cell: cell, bins: make(map[[3]int64][]int, len(verts))}
	for i, p := range verts {
		k := g.cellOf(p)
		g.bins[k] = append(g.bins[k], i)
	}
	return g
}

func (g *vertexGrid) cellOf(p math.Point3) [3]int64 {
	return [3]int64{
		int64(stdmath.Floor(float64(p.X) / g.cell)),
		int64(stdmath.Floor(float64(p.Y) / g.cell)),
		int64(stdmath.Floor(float64(p.Z) / g.cell)),
	}
}

// near returns the deduplicated vertex indices in the cells the segment a→b passes through, each expanded
// by one cell so a vertex just across a cell boundary is not missed.
func (g *vertexGrid) near(a, b math.Point3) []int {
	seen := map[int]bool{}
	var out []int
	ab := a.VectorTo(b)
	steps := int(a.DistanceTo(b)/g.cell) + 1
	for s := 0; s <= steps; s++ {
		c := g.cellOf(a.TranslateBy(ab.Scale(math.Scalar(float64(s) / float64(steps)))))
		for dx := int64(-1); dx <= 1; dx++ {
			for dy := int64(-1); dy <= 1; dy++ {
				for dz := int64(-1); dz <= 1; dz++ {
					for _, vi := range g.bins[[3]int64{c[0] + dx, c[1] + dy, c[2] + dz}] {
						if !seen[vi] {
							seen[vi] = true
							out = append(out, vi)
						}
					}
				}
			}
		}
	}
	return out
}

// onSegment reports whether p lies on the interior of segment a→b: within lineTol of
// the line and strictly between the endpoints (more than lineTol from each). Endpoint
// exclusion is by absolute distance, not a parameter fraction, so a vertex near a long
// edge's end is still recognized as a T-junction. lineTol is the model-relative on-line
// resolution the caller derives (ADR-0042), not a fixed constant.
func onSegment(p, a, b math.Point3, lineTol float64) bool {
	ab := a.VectorTo(b)
	lenSq := ab.LengthSquared()
	if lenSq == 0 {
		return false
	}
	t := a.VectorTo(p).Dot(ab) / lenSq
	if t <= 0 || t >= 1 {
		return false
	}
	foot := a.TranslateBy(ab.Scale(t))
	if foot.DistanceTo(p) >= lineTol {
		return false
	}
	return p.DistanceTo(a) > lineTol && p.DistanceTo(b) > lineTol
}

// dropDegenerate removes triangles with a repeated corner.
func dropDegenerate(faces [][3]int) [][3]int {
	out := faces[:0]
	for _, f := range faces {
		if f[0] != f[1] && f[1] != f[2] && f[0] != f[2] {
			out = append(out, f)
		}
	}
	return out
}

// cageToBody assembles a B-rep from a welded triangle cage: one shared vertex per cage
// vertex, one shared edge per undirected edge (sorted for stable lineage), a planar face
// per triangle. The body is a solid when every edge is shared by exactly two triangles.
func cageToBody(verts []math.Point3, faces [][3]int, feat string) *topo.Body {
	if len(faces) == 0 {
		return nil
	}
	uses := edgeUseCounts(faces)
	bld := topo.NewBuilder(isClosedCage(uses), topo.NewLineage(topo.Tok(feat, "body", 0)))
	tv := make([]*topo.Vertex, len(verts))
	for i, p := range verts {
		tv[i] = bld.AddVertex(p, topo.NewLineage(topo.Tok(feat, "vertex", i)))
	}
	edges := buildCageEdges(bld, verts, tv, uses, feat)
	for fi, f := range faces {
		bld.AddFace(trianglePlane(verts, f), topo.NewLineage(topo.Tok(feat, "face", fi)), triangleLoop(f, edges))
	}
	return bld.Build()
}

func edgeKeyOf(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// edgeUseCounts counts how many triangles use each undirected edge.
func edgeUseCounts(faces [][3]int) map[[2]int]int {
	uses := map[[2]int]int{}
	for _, f := range faces {
		for i := 0; i < 3; i++ {
			uses[edgeKeyOf(f[i], f[(i+1)%3])]++
		}
	}
	return uses
}

func isClosedCage(uses map[[2]int]int) bool {
	for _, c := range uses {
		if c != 2 {
			return false
		}
	}
	return len(uses) > 0
}

// buildCageEdges creates one shared topo edge per undirected edge, in sorted-key order
// so the synthesized lineage (and reference keys) is stable.
func buildCageEdges(bld *topo.Builder, verts []math.Point3, tv []*topo.Vertex, uses map[[2]int]int, feat string) map[[2]int]*topo.Edge {
	keys := make([][2]int, 0, len(uses))
	for k := range uses {
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
		seg := geom.NewLineSegment(verts[k[0]], verts[k[1]])
		edges[k] = bld.AddEdge(seg, tv[k[0]], tv[k[1]], topo.NewLineage(topo.Tok(feat, "edge", i)))
	}
	return edges
}

// triangleLoop builds a triangle's outer loop, marking a use reversed when its directed
// edge runs against the canonical (min,max) stored edge.
func triangleLoop(f [3]int, edges map[[2]int]*topo.Edge) topo.LoopSpec {
	uses := make([]topo.Use, 3)
	for i := 0; i < 3; i++ {
		a, b := f[i], f[(i+1)%3]
		uses[i] = topo.Use{Edge: edges[edgeKeyOf(a, b)], Reversed: a > b}
	}
	return topo.OuterLoop(uses...)
}

// trianglePlane fits the plane through a triangle's centroid with its (winding) normal,
// falling back to +Z for a degenerate triangle.
func trianglePlane(verts []math.Point3, f [3]int) geom.Surface {
	a, b, c := verts[f[0]], verts[f[1]], verts[f[2]]
	centroid := math.P3((a.X+b.X+c.X)/3, (a.Y+b.Y+c.Y)/3, (a.Z+b.Z+c.Z)/3)
	p, err := geom.NewPlane(centroid, a.VectorTo(b).Cross(a.VectorTo(c)))
	if err != nil {
		p, _ = geom.NewPlane(centroid, math.V3(0, 0, 1))
	}
	return p
}
