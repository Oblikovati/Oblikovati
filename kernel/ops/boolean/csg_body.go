// SPDX-License-Identifier: GPL-2.0-only

package boolean

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/mesh"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/ops/tessellate"
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
//
// It facets at the ANGLE-BOUNDED DefaultQuality, NOT the BSP's chord-only booleanInputQuality: the
// cage it returns becomes the operand of the EXACT PLANAR boolean, which has none of the BSP's
// hairline-crack fragility, so it must not inherit that trade. Chord-only faceting is radius-blind
// and collapses a small cylinder outright — at the display chord tolerance (0.05) a r=0.15 shaft
// admits 2*acos(1-0.05/0.15) = 1.68 rad per facet, i.e. FOUR of them: a square prism holding 2r²h
// = 64% of the true volume. A screw's Ø3mm shaft silently lost 38.5 of its 106.03 mm³ THAT WAY,
// before any boolean ran, and the cut that followed was then exactly as wrong as its input. The
// angle bound is radius-independent (~32 facets/circle at any size, DefaultQuality), which costs
// the documented ~0.64% inscribed-N-gon bias instead of 36%.
// # Validity is a post-condition (#3329)
//
// The cage this returns becomes an operand the exact planar boolean TRUSTS as valid, so handing
// back an invalid one launders a defect into a body that looks sound — the boolean then fails, or
// worse succeeds, somewhere with no connection to the faceting that caused it. An invalid cage is
// therefore refused (nil) rather than returned, which callers already treat as "keep the analytic
// body" and route down a path that can report its own failure.
//
// This is insurance, not a behaviour change: measured over the whole tier-2 corpus
// (kernel/... + model/feature) on 2026-09-01, Facet produced an invalid cage ZERO times.
//
// # Nothing takes this behind the caller's back any more (#3459)
//
// Facet existed because the planar boolean could not consume a curved operand, and
// model/feature.planarized applied it AUTOMATICALLY to every curved body before a boolean — the
// defect #3459 names, since faceting is permanent and unrecoverable. That automatic path is gone:
// the boolean takes analytic operands now, and the whole tier-2 corpus passes without it.
//
// Getting there was four fixes, none of them in the boolean's own recognizers:
//
//	#3463  the hole feature recorded a 32-gon prism as its REPLAY tool while cutting with the exact
//	       drill, so a pattern replayed a different solid and the pattern faceted everything to cope
//	#1689  a chamfer section is swept along its edge and only the LINEAR sweep existed, so a closed
//	       circular rim — whose two endpoints are the same point — reported "degenerate edge"
//	#3459  the mixed boolean could not carry a HYPERBOLIC imprint: the section a plane cuts from a
//	       cone when it runs parallel to the axis, which is what an emboss pad's side faces do
//
// What remains is an explicit operation. A caller that genuinely wants a triangle cage can still
// ask for one, and one case still needs it: joining a block onto a plate with ANALYTIC bores falls
// to triangle CSG (TestTwoStraddlingHolesStayOnTheExactPath facets first to stay on the exact
// path). Deleting the function outright waits on that.
func Facet(b *topo.Body, feat string) *topo.Body {
	cage := trianglesToBody(bodyTrianglesAt(b, DefaultQuality()), feat)
	if cage == nil {
		return nil
	}
	// A one-face-per-triangle cage is combinatorially valid but shreds every flat region into a
	// diagonal-laced fan; unifying coplanar faces restores each flat region to the single face it
	// is, so a downstream fillet/boolean does not choke on spurious diagonals (Oblikovati#1693).
	out := brep.UnifyCoplanarFaces(cage, feat)
	if !Validate(out).ValidSolid() {
		return nil // refuse rather than launder an invalid cage into a trusted operand (#3329)
	}
	return out
}

// bodyTriangles returns a body's tessellation as CSG triangles, each oriented
// outward. The BSP-tree CSG depends on globally consistent outward winding, but
// TessellateBody does not guarantee it for curved faces (a cylinder's side
// triangles can wind either way — the same reason meshGeometryProperties
// re-orients before its volume sum). Trusting the raw winding silently breaks any
// boolean whose minuend is a curved body: the BSP misclassifies inside/outside and
// subtracts nothing (cylinder − tool returned the uncut cylinder). We fix the
// winding here with the per-vertex shading normals, which point outward.
func bodyTriangles(b *topo.Body) []mesh.Tri {
	return bodyTrianglesAt(b, booleanInputQuality())
}

// bodyTrianglesAt is [bodyTriangles] at an explicit faceting quality. The BSP CSG needs its
// chord-only booleanInputQuality for robustness; Facet, whose cage feeds the exact planar boolean
// instead, needs the angle-bounded one so small curved faces survive (see Facet).
func bodyTrianglesAt(b *topo.Body, q Quality) []mesh.Tri {
	m, _ := tessellate.TessellateBody(b, q)
	var out []mesh.Tri
	for i := 0; i+2 < len(m.Indices); i += 3 {
		ia, ib, ic := m.Indices[i], m.Indices[i+1], m.Indices[i+2]
		a, bb, c := m.Positions[ia], m.Positions[ib], m.Positions[ic]
		if query.OutwardRef(m, ia, ib, ic).Dot(a.VectorTo(bb).Cross(a.VectorTo(c))) < 0 {
			bb, c = c, bb // flip to outward winding
		}
		if t, ok := mesh.NewTri(a, bb, c); ok {
			out = append(out, t)
		}
	}
	return out
}

// trianglesToBody welds CSG output triangles into a watertight B-rep: coincident
// vertices merge, T-junctions are split out so the cage is combinatorially closed, and
// the welded triangle cage becomes a body. Returns nil when the result is empty.
func trianglesToBody(tris []mesh.Tri, feat string) *topo.Body {
	// One model-relative resolution for the whole triangle set (ADR-0042): a tight
	// vertex weld and a wider on-line tolerance, both scaling with the operand size,
	// so a sub-µm part is no longer welded out of existence while a finely-detailed
	// large part is not over-merged.
	res := ResolutionForTris(tris)
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
