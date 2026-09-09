// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// The body's own tessellation post-condition (ADR-0061 stage 5).
//
// A CLOSED solid's mesh is a closed surface. Every per-face mesher already certifies its own patch,
// but nothing certified the body: two faces can each mesh correctly and still discretise the boundary
// they SHARE differently, and the crack that leaves is invisible until somebody counts free edges.
// That is a degradation like any other — a rendered surface with a hole in it, a mass property
// integrated over a torn shell — and the ground rules do not let it ship silently.
//
// It runs in TessellateBodyFaces, which is the ONE place every route passes through: the whole-body
// mesh (TessellateBody), the facet store (CalculateBodyFacets), and the diagnostics harvest that
// reaches feature health, the API and the UI (query.BodyMeshDiagnostics). Recording it on a face mesh
// rather than on a merged one is what puts it on all three — the merge carries face diagnostics up,
// and the harvest reads the face meshes directly. Putting it only on the merged mesh, which is where
// this first landed, reached nobody: every TessellateBody caller discards the mesh, and both harvests
// take the faces route.
//
// The check is topological on the way in (the B-rep says the body is closed) and metric only on the
// mesh, so it cannot fire on a body that is genuinely open: a sheet, a surface body, an unstitched
// import. It reports; it does not repair. Repairing belongs to the mesher whose premise broke, and
// naming the faces the tear touches is what points at it.

// CodeMeshNotWatertight marks a body mesh that is NOT a closed surface although the B-rep it was built
// from IS a closed solid. Everything read from that mesh downstream — the rendered surface, a mesh
// export, a volume integrated from facets — is wrong there, and the detail says HOW by the edges'
// degrees: an edge used by ONE triangle is a crack (two neighbours discretised their shared boundary
// differently), one used by THREE or more is a doubled surface (a face emitted a triangle that coincides
// with a neighbour's, or two faces meshed one region). The two are different defects in different
// meshers, and one wording for both sent the reader to the wrong one (final fix wave, finding 2).
const CodeMeshNotWatertight diag.Code = "tessellate.mesh-not-watertight"

// recordBodyMeshTear is the post-condition itself: it records the tear on the face mesh of the
// LOWEST-indexed face the tear touches, so a harvest that reads face meshes in order reports it once.
func recordBodyMeshTear(b *topo.Body, faces []*topo.Face, fm []*Mesh) {
	if !bodyIsClosedSolid(b) {
		return
	}
	torn := tornAcrossMeshes(fm)
	on, ok := firstTornMesh(fm, torn)
	if !ok {
		return
	}
	fm[on].Diagnose(diag.Diagnostic{Code: CodeMeshNotWatertight, Severity: diag.Defect,
		Detail: tearDetail(faces, torn)})
}

// tearDetail words the tear by the class its edges' degrees put it in.
func tearDetail(faces []*topo.Face, torn []meshTear) string {
	cracks, doubled := partitionTears(torn)
	names := tornFaceNames(faces, torn)
	switch {
	case len(doubled) == 0:
		return fmt.Sprintf("the B-rep is a closed solid but its mesh has %d free edge(s), each used by one "+
			"triangle, on face(s) %s: a pair of neighbouring faces did not discretise the boundary they "+
			"share the same way, so the meshed surface is torn there", len(cracks), names)
	case len(cracks) == 0:
		return fmt.Sprintf("the B-rep is a closed solid but its mesh has %d over-merged edge(s), each used "+
			"by three or more triangles, on face(s) %s: a face emitted a triangle that coincides with a "+
			"neighbour's, or two faces meshed one region, so the meshed surface is doubled there",
			len(doubled), names)
	}
	return fmt.Sprintf("the B-rep is a closed solid but its mesh has %d free edge(s) (one triangle each) "+
		"and %d over-merged edge(s) (three or more), on face(s) %s: the meshed surface is torn AND "+
		"doubled there", len(cracks), len(doubled), names)
}

// partitionTears splits the tears by degree: one use is a crack, three or more an over-merge. Zero
// cannot occur (an edge is only recorded by a triangle that uses it) and two is not a tear.
func partitionTears(torn []meshTear) (cracks, doubled []meshTear) {
	for _, tear := range torn {
		if len(tear.on) == 1 {
			cracks = append(cracks, tear)
			continue
		}
		doubled = append(doubled, tear)
	}
	return cracks, doubled
}

// firstTornMesh is the lowest index a tear touches that has a mesh to record on. ok=false when nothing
// is torn, or when every mesh a tear touches is one the mesher declined to build.
func firstTornMesh(fm []*Mesh, torn []meshTear) (int, bool) {
	for _, i := range tornMeshIndices(torn) {
		if i < len(fm) && fm[i] != nil {
			return i, true
		}
	}
	return 0, false
}

// bodyIsClosedSolid is the topological closure this package already knows: a solid body with no
// boundary edge. It reads the B-REP, never the mesh, so tessellation stays downstream of the model.
func bodyIsClosedSolid(b *topo.Body) bool {
	return b != nil && b.IsSolid() && len(validate.BoundaryEdges(b)) == 0
}

// meshTear is one welded edge that is not shared by exactly two triangles, with the meshes whose
// triangles use it — one entry per use, so the count is the degree.
type meshTear struct {
	lo, hi int
	on     []int
}

// tornMeshEdges is [tornAcrossMeshes] for a single mesh — the watertightness metric
// [WeldedFreeEdgeCount] reports the size of.
func tornMeshEdges(m *Mesh) []meshTear { return tornAcrossMeshes([]*Mesh{m}) }

// tornAcrossMeshes welds the meshes' vertices together at the model's own resolution (ADR-0042) and
// returns every edge not used by exactly two triangles — a crack (degree 1) and an over-merge (degree
// 3 or more) alike. Welding the group at once is what makes a body's faces meet: two faces' copies of
// one boundary point are separate vertices until they weld.
//
// The edges come back in the order the triangles first name them, never in map order: the map is a
// lookup, the parallel slice is the order (#2192).
func tornAcrossMeshes(fm []*Mesh) []meshTear {
	weld, base := weldAcrossMeshes(fm)
	use, order := map[[2]int][]int{}, [][2]int{}
	for i, m := range fm {
		collectWeldedEdges(m, i, base[i], weld, use, &order)
	}
	return tornOf(use, order)
}

// collectWeldedEdges files every triangle edge of one mesh under its welded endpoints.
func collectWeldedEdges(m *Mesh, at, base int, weld []int, use map[[2]int][]int, order *[][2]int) {
	if m == nil {
		return
	}
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[base+m.Indices[3*t]], weld[base+m.Indices[3*t+1]], weld[base+m.Indices[3*t+2]]}
		for k := range 3 {
			e := sortedPair(v[k], v[(k+1)%3])
			if _, seen := use[e]; !seen {
				*order = append(*order, e)
			}
			use[e] = append(use[e], at)
		}
	}
}

// tornOf collects the edges used by anything but exactly two triangles, walking the first-seen order.
func tornOf(use map[[2]int][]int, order [][2]int) []meshTear {
	var out []meshTear
	for _, e := range order {
		if on := use[e]; len(on) != 2 {
			out = append(out, meshTear{lo: e[0], hi: e[1], on: on})
		}
	}
	return out
}

// weldAcrossMeshes maps every vertex of every mesh to the first vertex it welds onto, on the grid the
// whole GROUP's extent sets, and returns each mesh's offset into that numbering.
func weldAcrossMeshes(fm []*Mesh) ([]int, []int) {
	base := make([]int, len(fm))
	var all []math.Point3
	for i, m := range fm {
		base[i] = len(all)
		if m != nil {
			all = append(all, m.Positions...)
		}
	}
	return weldedVertexIndices(all), base
}

// weldedVertexIndices maps every point to the first point it welds onto, on the model's own grid.
func weldedVertexIndices(pts []math.Point3) []int {
	grid := geom.ResolutionForPoints(pts).Weld()
	canon := map[[3]int64]int{}
	weld := make([]int, len(pts))
	for i, p := range pts {
		k := WeldKey(p, grid)
		if c, ok := canon[k]; ok {
			weld[i] = c
			continue
		}
		canon[k], weld[i] = i, i
	}
	return weld
}

// sortedPair orders an edge's two welded endpoints so the two triangles that share it agree on the key.
func sortedPair(a, b int) [2]int {
	if a > b {
		return [2]int{b, a}
	}
	return [2]int{a, b}
}

// tornFaceNames lists, in one order and without repeats, the reference keys of the faces whose
// triangles touch a tear. Two result faces can carry ONE key — a merged parent's, an unnamed
// fragment's — and naming it twice says nothing the once did not.
func tornFaceNames(faces []*topo.Face, torn []meshTear) string {
	var names []string
	for _, i := range tornMeshIndices(torn) {
		if i < len(faces) {
			names = append(names, string(faces[i].ReferenceKey()))
		}
	}
	sort.Strings(names)
	return strings.Join(slices.Compact(names), ", ")
}

// tornMeshIndices is every mesh a tear touches, ascending and without repeats.
func tornMeshIndices(torn []meshTear) []int {
	seen := map[int]bool{}
	var out []int
	for _, tear := range torn {
		for _, i := range tear.on {
			if !seen[i] {
				seen[i], out = true, append(out, i)
			}
		}
	}
	sort.Ints(out)
	return out
}
