// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Constraint-edge insertion: force a segment between two existing vertices to be
// present as triangulation edges. The segment carves a cavity (the triangles its
// interior crosses); removing them exposes a simple polygon that the segment
// splits into two, each re-triangulated by ear clipping with the segment as one
// edge. This is what makes the co-refined face conform to the other operand along
// the intersection curve. PRECONDITION for ForceEdge: no vertex lies strictly
// between ui and vi (the caller inserts every crossing point first), so the
// segment crosses existing edges only transversally.
func (tr *Triangulation) ForceEdge(ui, vi int) {
	if ui == vi || tr.edgeExists(ui, vi) {
		return
	}
	crossed := tr.trianglesCrossedBy(ui, vi)
	if len(crossed) == 0 {
		return
	}
	cycle := tr.cavityBoundary(crossed)
	left, right := splitCycle(cycle, ui, vi)
	tr.removeTriangles(crossed)
	tr.retriangulate(left)
	tr.retriangulate(right)
}

// edgeExists reports whether ui and vi are both vertices of some triangle (hence
// joined by an edge).
func (tr *Triangulation) edgeExists(ui, vi int) bool {
	for _, t := range tr.tris {
		if triHas(t, ui) && triHas(t, vi) {
			return true
		}
	}
	return false
}

// trianglesCrossedBy returns the indices of triangles whose interior the segment
// (ui,vi) crosses — those with an edge the segment properly crosses.
func (tr *Triangulation) trianglesCrossedBy(ui, vi int) []int {
	u, v := tr.verts[ui], tr.verts[vi]
	var out []int
	for i, t := range tr.tris {
		for e := 0; e < 3; e++ {
			if segmentsProperlyCross(u, v, tr.verts[t[e]], tr.verts[t[(e+1)%3]], tr.axis) {
				out = append(out, i)
				break
			}
		}
	}
	return out
}

// cavityBoundary returns the boundary vertex cycle of the crossed-triangle set. A
// directed edge on the boundary has no oppositely-directed twin in the set (an
// interior edge is shared by two triangles, appearing in both directions); the
// boundary edges chain into a single cycle because the cavity is simply connected.
func (tr *Triangulation) cavityBoundary(crossed []int) []int {
	present := make(map[[2]int]bool)
	for _, ti := range crossed {
		t := tr.tris[ti]
		for e := 0; e < 3; e++ {
			present[[2]int{t[e], t[(e+1)%3]}] = true
		}
	}
	nextOf := make(map[int]int)
	for de := range present {
		if !present[[2]int{de[1], de[0]}] {
			nextOf[de[0]] = de[1]
		}
	}
	return traceCycle(nextOf)
}

// traceCycle walks the successor map from an arbitrary start back to it, bounded
// by the map size so a malformed (non-simple) cavity cannot loop forever.
func traceCycle(nextOf map[int]int) []int {
	var start int
	for a := range nextOf {
		start = a
		break
	}
	cycle := []int{start}
	for cur := nextOf[start]; cur != start; cur = nextOf[cur] {
		cycle = append(cycle, cur)
		if len(cycle) > len(nextOf) {
			break // defensive: not a single simple cycle
		}
	}
	return cycle
}

// splitCycle cuts the boundary cycle at ui and vi into the two chains ui..vi and
// vi..ui (each traversed forward, wrapping), the two sides the forced edge splits.
func splitCycle(cycle []int, ui, vi int) (left, right []int) {
	n := len(cycle)
	iu, iv := indexIn(cycle, ui), indexIn(cycle, vi)
	for i := iu; ; i = (i + 1) % n {
		left = append(left, cycle[i])
		if i == iv {
			break
		}
	}
	for i := iv; ; i = (i + 1) % n {
		right = append(right, cycle[i])
		if i == iu {
			break
		}
	}
	return left, right
}

// retriangulate ear-clips one cavity side (a vertex chain) and appends the
// resulting triangles, mapping the returned points back to their vertex indices.
func (tr *Triangulation) retriangulate(chain []int) {
	if len(chain) < 3 {
		return // the forced edge is already this side's boundary; nothing to fill
	}
	pts := make([]Point, len(chain))
	for i, vi := range chain {
		pts[i] = tr.verts[vi]
	}
	for _, t := range triangulateSimplePolygon(pts, tr.axis) {
		tr.tris = append(tr.tris, [3]int{tr.indexOf(t[0]), tr.indexOf(t[1]), tr.indexOf(t[2])})
	}
}

// removeTriangles drops the triangles at the given indices.
func (tr *Triangulation) removeTriangles(idxs []int) {
	drop := make(map[int]bool, len(idxs))
	for _, i := range idxs {
		drop[i] = true
	}
	kept := make([][3]int, 0, len(tr.tris)-len(idxs))
	for i, t := range tr.tris {
		if !drop[i] {
			kept = append(kept, t)
		}
	}
	tr.tris = kept
}

func triHas(t [3]int, v int) bool {
	return t[0] == v || t[1] == v || t[2] == v
}

func indexIn(cycle []int, v int) int {
	for i, x := range cycle {
		if x == v {
			return i
		}
	}
	return -1
}
