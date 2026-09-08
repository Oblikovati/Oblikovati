// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"fmt"
	"sort"
	"strings"

	"oblikovati.org/kernel/diag"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/validate"
	"oblikovati.org/kernel/topo"
)

// TessellateBody's own post-condition (ADR-0061 stage 5).
//
// A CLOSED solid's mesh is a closed surface. Every per-face mesher already certifies its own patch,
// but nothing certified the body: two faces can each mesh correctly and still discretise the boundary
// they SHARE differently, and the crack that leaves is invisible until somebody counts free edges.
// That is a degradation like any other — a rendered surface with a hole in it, a mass property
// integrated over a torn shell — and the ground rules do not let it ship silently.
//
// The check is topological on the way in (the B-rep says the body is closed) and metric only on the
// mesh, so it cannot fire on a body that is genuinely open: a sheet, a surface body, an unstitched
// import. It reports; it does not repair. Repairing belongs to the mesher whose premise broke, and
// naming the faces the tear touches is what points at it.

// CodeMeshNotWatertight marks a body mesh that is NOT a closed surface although the B-rep it was built
// from IS a closed solid. Everything read from that mesh downstream — the rendered surface, a mesh
// export, a volume integrated from facets — is a torn shell, and the faces named in the detail are the
// ones whose shared boundary did not discretise the same way on both sides.
const CodeMeshNotWatertight diag.Code = "tessellate.mesh-not-watertight"

// recordMeshTear is the post-condition itself. spans[i] is the triangle index face i's own triangles
// begin at, with a final sentinel, so a tear found on the whole-body mesh is named against its faces.
func recordMeshTear(m *Mesh, b *topo.Body, faces []*topo.Face, spans []int) {
	if m == nil || !bodyIsClosedSolid(b) {
		return
	}
	torn := tornMeshEdges(m)
	if len(torn) == 0 {
		return
	}
	m.Diagnose(diag.Diagnostic{Code: CodeMeshNotWatertight, Severity: diag.Defect,
		Detail: fmt.Sprintf("the B-rep is a closed solid but its mesh has %d free edge(s), on face(s) "+
			"%s: a pair of neighbouring faces did not discretise the boundary they share the same way, "+
			"so the meshed surface is torn there", len(torn), tornFaceNames(faces, spans, torn))})
}

// bodyIsClosedSolid is the topological closure this package already knows: a solid body with no
// boundary edge. It reads the B-REP, never the mesh, so tessellation stays downstream of the model.
func bodyIsClosedSolid(b *topo.Body) bool {
	return b != nil && b.IsSolid() && len(validate.BoundaryEdges(b)) == 0
}

// meshTear is one welded mesh edge that is not shared by exactly two triangles, with the triangles
// that do use it.
type meshTear struct {
	lo, hi int
	tris   []int
}

// tornMeshEdges welds coincident vertices at the model's own resolution (ADR-0042) and returns every
// edge not used by exactly two triangles — a crack (degree 1) and an over-merge (degree 3 or more)
// alike, which is the watertightness metric [WeldedFreeEdgeCount] reports the size of.
//
// The edges come back in the order the triangles first name them, never in map order: the map is a
// lookup, the parallel slice is the order (#2192).
func tornMeshEdges(m *Mesh) []meshTear {
	use, order := map[[2]int][]int{}, [][2]int{}
	weld := weldedVertexIndices(m)
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		v := [3]int{weld[m.Indices[3*t]], weld[m.Indices[3*t+1]], weld[m.Indices[3*t+2]]}
		for k := range 3 {
			e := sortedPair(v[k], v[(k+1)%3])
			if _, seen := use[e]; !seen {
				order = append(order, e)
			}
			use[e] = append(use[e], t)
		}
	}
	return tornOf(use, order)
}

// tornOf collects the edges used by anything but exactly two triangles, walking the first-seen order.
func tornOf(use map[[2]int][]int, order [][2]int) []meshTear {
	var out []meshTear
	for _, e := range order {
		if tris := use[e]; len(tris) != 2 {
			out = append(out, meshTear{lo: e[0], hi: e[1], tris: tris})
		}
	}
	return out
}

// weldedVertexIndices maps every vertex to the first vertex it welds onto, on the model's own grid.
func weldedVertexIndices(m *Mesh) []int {
	grid := geom.ResolutionForPoints(m.Positions).Weld()
	canon := map[[3]int64]int{}
	weld := make([]int, len(m.Positions))
	for i, p := range m.Positions {
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

// tornFaceNames lists, in one order, the reference keys of the faces whose triangles touch a tear.
func tornFaceNames(faces []*topo.Face, spans []int, torn []meshTear) string {
	seen := map[int]bool{}
	var names []string
	for _, fi := range tornFaceIndices(spans, torn) {
		if seen[fi] || fi >= len(faces) {
			continue
		}
		seen[fi] = true
		names = append(names, string(faces[fi].ReferenceKey()))
	}
	sort.Strings(names)
	return strings.Join(names, ", ")
}

// tornFaceIndices is the face each torn edge's triangles belong to, by the span they fall in.
func tornFaceIndices(spans []int, torn []meshTear) []int {
	var out []int
	for _, tear := range torn {
		for _, t := range tear.tris {
			out = append(out, sort.SearchInts(spans, t+1)-1)
		}
	}
	sort.Ints(out)
	return out
}
