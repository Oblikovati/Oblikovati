// SPDX-License-Identifier: GPL-2.0-only

package ops

// FoldEdgeCount returns the number of interior mesh edges (each shared by exactly two triangles)
// whose triangles fold back on each other — their 3D geometric normals oppose by more than
// foldDihedralTol. A mesh can be watertight (freeEdgeCount == 0) yet still fold: a (u,v)→3D
// triangulation of a strongly non-conformal surface (the imported-NURBS "staircase") creases over
// itself, which inflates surface area and the enclosed volume — an over-enclosure with the right
// free-edge count but the wrong mass. FoldEdgeCount localizes that, complementing freeEdgeCount.
//
// Example: FoldEdgeCount(mesh) == 0 is required for a tessellation to bound a well-defined volume.
func FoldEdgeCount(m *Mesh) int {
	count := 0
	for _, ts := range edgeTriMap(m) {
		if len(ts) == 2 && trianglesFold(m, ts[0], ts[1]) {
			count++
		}
	}
	return count
}

// FoldEdges returns the folding interior edges as (a, b) vertex-index pairs — the same set
// FoldEdgeCount counts, in a deterministic ascending order so diagnostics (which edge folds) are
// reproducible across runs.
func FoldEdges(m *Mesh) [][2]int {
	adj := edgeTriMap(m)
	var out [][2]int
	for _, e := range sortedEdgeKeys(adj) {
		if ts := adj[e]; len(ts) == 2 && trianglesFold(m, ts[0], ts[1]) {
			out = append(out, [2]int{e.a, e.b})
		}
	}
	return out
}

// MeshArea returns the total surface area of the mesh (the sum of its triangle areas). A folded
// tessellation reports MORE area than the true surface — the overlapping fold is double-counted —
// so comparing MeshArea to an analytic (or oracle) reference area is an over-enclosure signal that
// does not need a volume oracle.
func MeshArea(m *Mesh) float64 {
	var area float64
	for t := 0; 3*t+2 < len(m.Indices); t++ {
		a := m.Positions[m.Indices[3*t]]
		b := m.Positions[m.Indices[3*t+1]]
		c := m.Positions[m.Indices[3*t+2]]
		area += 0.5 * float64(a.VectorTo(b).Cross(a.VectorTo(c)).Length())
	}
	return area
}
