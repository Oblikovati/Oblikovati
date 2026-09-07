// SPDX-License-Identifier: GPL-2.0-only

// Package meshbrep converts a welded triangle mesh into a faceted B-rep solid.
//
// It is the mesh-solid IMPORT path — an STL or OBJ arriving as vertices and facet loops becomes a
// body the modeller can carry, name and tessellate. It is not a modelling engine and no operation
// falls back to it: the faceted booleans that once shared this welder are gone (ADR-0061 stage 7),
// and the welder stayed because importing a mesh is a capability of its own.
package meshbrep

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/internal/tol"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// MeshToBRep converts a closed welded mesh — shared vertices and facets given as ordered
// vertex-index loops (each typically a triangle) — into a faceted B-rep solid: one planar face
// per facet, with shared edges and vertices. Facets with more than three vertices are
// fan-triangulated. The cage is re-oriented to positive volume so an inward-wound (but
// consistent) input still yields an outward solid. Returns nil for an empty mesh.
//
// Example: tetra := MeshToBRep(verts, [][]int{{0,1,2},{0,2,3},{0,3,1},{1,3,2}}, "mesh") — a
// validated 4-face solid.
func MeshToBRep(verts []math.Point3, facets [][]int, feat string) *topo.Body {
	tris := facetTriangles(verts, facets)
	if len(tris) == 0 {
		return nil
	}
	body := trianglesToBody(tris, feat)
	if body == nil {
		return nil
	}
	// cageToBody trusts the facet winding for face normals; if the mesh was wound inward
	// (negative volume) flip every facet so the result is a proper outward solid.
	if query.BodyGeometryProperties(body, mesh.DefaultQuality()).Volume < 0 {
		body = trianglesToBody(reversedTris(tris), feat)
	}
	return body
}

// facetTriangles fan-triangulates each facet loop into CSG triangles, dropping degenerate ones
// and any facet whose indices are out of range.
func facetTriangles(verts []math.Point3, facets [][]int) []mesh.Tri {
	var tris []mesh.Tri
	for _, f := range facets {
		if len(f) < 3 || !facetInRange(f, len(verts)) {
			continue
		}
		for i := 1; i+1 < len(f); i++ {
			if t, ok := mesh.NewTri(verts[f[0]], verts[f[i]], verts[f[i+1]]); ok {
				tris = append(tris, t)
			}
		}
	}
	return tris
}

func facetInRange(f []int, n int) bool {
	for _, idx := range f {
		if idx < 0 || idx >= n {
			return false
		}
	}
	return true
}

// reversedTris flips each triangle's winding (swapping two corners).
func reversedTris(tris []mesh.Tri) []mesh.Tri {
	out := make([]mesh.Tri, 0, len(tris))
	for _, t := range tris {
		if rt, ok := mesh.NewTri(t.A, t.C, t.B); ok {
			out = append(out, rt)
		}
	}
	return out
}

func trianglesToBody(tris []mesh.Tri, feat string) *topo.Body {
	// One model-relative resolution for the whole triangle set (ADR-0042): a tight
	// vertex weld and a wider on-line tolerance, both scaling with the operand size,
	// so a sub-µm part is no longer welded out of existence while a finely-detailed
	// large part is not over-merged.
	res := tol.ForTris(tris)
	verts, faces := weldTriangles(tris, res.Weld())
	faces = dedupTriangles(faces)
	if len(faces) == 0 {
		return nil
	}
	verts, faces = removeTJunctions(verts, faces, res.Plane())
	faces = dropDegenerate(faces)
	// A sub-resolution tangency (a face designed exactly on another body's wall) welds into a
	// PINCHED vertex — two fans on one vertex, χ off by one per pinch, invisible to the edge
	// checks. Cut such vertices apart into coincident duplicates so the cage is a true closed
	// 2-manifold; the shells then touch at a point instead of sharing an inadmissible vertex (#1693).
	verts = splitPinchedVertices(verts, faces)
	return mesh.CageToBody(verts, faces, feat)
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
func weldTriangles(tris []mesh.Tri, grid float64) ([]math.Point3, [][3]int) {
	index := map[[3]int64]int{}
	var verts []math.Point3
	weld := func(p math.Point3) int {
		k := [3]int64{mesh.Quantize(p.X, grid), mesh.Quantize(p.Y, grid), mesh.Quantize(p.Z, grid)}
		if i, ok := index[k]; ok {
			return i
		}
		index[k] = len(verts)
		verts = append(verts, p)
		return len(verts) - 1
	}
	var faces [][3]int
	for _, t := range tris {
		a, b, c := weld(t.A), weld(t.B), weld(t.C)
		if a != b && b != c && a != c {
			faces = append(faces, [3]int{a, b, c})
		}
	}
	return verts, faces
}

// edgeSubdividedBoundary walks the triangle boundary, emitting each corner followed by the mesh vertices
// lying on the interior of the edge leaving it (ordered along that edge). With no on-edge point it returns
// the three corners unchanged.
func edgeSubdividedBoundary(verts []math.Point3, f [3]int, idx *axisIndex, lineTol float64) []int {
	poly := make([]int, 0, 3)
	for e := range 3 {
		p, q := f[e], f[(e+1)%3]
		poly = append(poly, p)
		poly = append(poly, edgeInteriorPoints(verts, p, q, idx, lineTol)...)
	}
	return poly
}

// edgeInteriorPoints returns the mesh vertices on the interior of segment p→q, ordered from p to q. Only
// the vertices the axis index reports within the segment's coordinate slab are tested, so the cost is the
// slab size, not O(verts).
func edgeInteriorPoints(verts []math.Point3, p, q int, idx *axisIndex, lineTol float64) []int {
	a, b := verts[p], verts[q]
	ab := a.VectorTo(b)
	type onPt struct {
		idx int
		t   float64
	}
	var hits []onPt
	for _, ci := range idx.near(a, b, lineTol) {
		if ci == p || ci == q {
			continue
		}
		if mesh.OnSegment(verts[ci], a, b, lineTol) {
			hits = append(hits, onPt{ci, float64(ab.Dot(a.VectorTo(verts[ci])))})
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

// axisIndex sorts the vertices along their widest-spread axis so a segment's candidate on-edge vertices —
// those whose axis coordinate falls in the segment's coordinate range — are found by binary search, in
// time proportional to that slab, not O(verts). A uniform spatial hash fails here: there is no cell size
// that is small enough to spread a dense vertex cluster (else one cell holds O(cluster²) pairs) yet large
// enough that a long edge does not walk millions of cells — the previewshot 600s hang. The sweep index
// avoids both: a tiny edge queries a tiny coordinate slab; the rare long edge scans more vertices but only
// once. Worst case (many edges spanning the widest axis) degrades to a linear scan, still bounded.
type axisIndex struct {
	axis   int
	order  []int     // vertex indices sorted ascending by their axis coordinate
	coords []float64 // the sorted axis coordinates, parallel to order (for binary search)
}

// newAxisIndex builds the sweep index on the widest-spread axis (the most discriminating).
func newAxisIndex(verts []math.Point3) *axisIndex {
	axis := widestAxis(verts)
	order := make([]int, len(verts))
	for i := range order {
		order[i] = i
	}
	sort.Slice(order, func(a, b int) bool { return coordOf(verts[order[a]], axis) < coordOf(verts[order[b]], axis) })
	coords := make([]float64, len(order))
	for i, vi := range order {
		coords[i] = coordOf(verts[vi], axis)
	}
	return &axisIndex{axis: axis, order: order, coords: coords}
}

// near returns the vertex indices whose axis coordinate lies within the segment a→b's range, expanded by
// tol so a vertex just off either end is still a candidate. Endpoint/off-line rejection is left to
// onSegment; this only narrows the field cheaply.
func (x *axisIndex) near(a, b math.Point3, tol float64) []int {
	lo := stdmath.Min(coordOf(a, x.axis), coordOf(b, x.axis)) - tol
	hi := stdmath.Max(coordOf(a, x.axis), coordOf(b, x.axis)) + tol
	i := sort.SearchFloat64s(x.coords, lo)
	out := make([]int, 0, 8)
	for ; i < len(x.coords) && x.coords[i] <= hi; i++ {
		out = append(out, x.order[i])
	}
	return out
}

// coordOf returns a point's coordinate on axis 0/1/2.
func coordOf(p math.Point3, axis int) float64 {
	switch axis {
	case 0:
		return float64(p.X)
	case 1:
		return float64(p.Y)
	default:
		return float64(p.Z)
	}
}

// widestAxis returns the axis (0/1/2) over which the vertices spread furthest — the one that best
// separates them in the sweep index.
func widestAxis(verts []math.Point3) int {
	if len(verts) == 0 {
		return 0
	}
	lo, hi := verts[0], verts[0]
	for _, p := range verts {
		lo = math.P3(stdmath.Min(float64(lo.X), float64(p.X)), stdmath.Min(float64(lo.Y), float64(p.Y)), stdmath.Min(float64(lo.Z), float64(p.Z)))
		hi = math.P3(stdmath.Max(float64(hi.X), float64(p.X)), stdmath.Max(float64(hi.Y), float64(p.Y)), stdmath.Max(float64(hi.Z), float64(p.Z)))
	}
	dx, dy, dz := float64(hi.X-lo.X), float64(hi.Y-lo.Y), float64(hi.Z-lo.Z)
	if dx >= dy && dx >= dz {
		return 0
	}
	if dy >= dz {
		return 1
	}
	return 2
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

// removeTJunctions eliminates T-junctions so every undirected edge of the cage is shared by exactly two
// triangles (the prerequisite for a closed solid). For each triangle it collects the mesh vertices lying
// on the interior of its three edges and re-fans the triangle through them in ONE pass — no cascade and
// no full-vertex scan: a sorted sweep index (axisIndex) makes the per-edge lookup local, so the pass is
// near-linear in the triangle count and needs NO face budget. The cage always comes back combinatorially
// closed, however many facets a curved wall contributed — this is acceptance #3 of the curved-boolean
// umbrella (Oblikovati/Oblikovati#1336, #1320 #3), replacing the bounded O(faces·verts) cascade of
// M20-F01 #470 that BAILED above tjunctionFaceBudget and left the cage open (the chained-bore drift the
// guard catches). Two triangles sharing an edge collect the same on-edge points off the same segment, so
// their subdivisions match and weld. Returns the (possibly grown) vertex list — a subdivided triangle
// gains an interior centroid hub — alongside the new faces.
func removeTJunctions(verts []math.Point3, faces [][3]int, lineTol float64) ([]math.Point3, [][3]int) {
	idx := newAxisIndex(verts)
	out := make([][3]int, 0, len(faces))
	for _, f := range faces {
		poly := edgeSubdividedBoundary(verts, f, idx, lineTol)
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
