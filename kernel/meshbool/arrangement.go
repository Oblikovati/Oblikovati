// SPDX-License-Identifier: GPL-2.0-only

package meshbool

// Arrangement topology extraction (ADR-0056 Layer 2a). The tagged boolean result is a
// watertight triangle soup where every triangle carries the id of the operand surface
// it descends from. Reconstruction groups those triangles into analytic faces; this
// file recovers the pure TOPOLOGY it needs — which faces there are, and how they
// border one another — with no geometry beyond the exact vertex positions. A face is a
// maximal tag-connected patch; the boundary between two faces is a RUN (a maximal
// chain of facet edges with the same neighbour face), which reconstruction later fits
// with a single analytic curve (two analytic surfaces meet along one smooth curve, so
// a constant-neighbour run is exactly one edge).

// ArrangementTopology is the tag-grouped face/edge topology of a result soup: the
// deduplicated vertex list plus one ArrangementFace per output face.
type ArrangementTopology struct {
	Verts []Point
	Faces []ArrangementFace
}

// ArrangementFace is one output face: the provenance tag naming its originating
// surface, and its boundary loops (outer first when classifiable).
type ArrangementFace struct {
	Tag   int
	Loops []ArrangementLoop
}

// ArrangementLoop is one boundary loop of a face, split into runs. Outer is true for
// the loop reconstruction treats as the outer boundary (largest projected area); it is
// meaningful only for a bounded patch, not a periodic band (an untouched wall is
// reconstructed from its original face, not from these loops).
type ArrangementLoop struct {
	Outer bool
	Runs  []ArrangementRun
}

// ArrangementRun is a maximal boundary chain bordering ONE neighbour face: the
// neighbour's tag (which surface is on the other side, for the intersection curve) and
// the vertex indices along the run, endpoints inclusive. Consecutive runs of a loop
// share an endpoint (a corner where three or more faces meet).
type ArrangementRun struct {
	NeighborTag int
	Verts       []int
}

// ArrangementTopologyOf extracts the tag-grouped face/edge topology of a watertight
// tagged soup. PRECONDITION: soup is closed and 2-manifold (the boolean guarantees it)
// with len(Tags) == len(Tris).
func ArrangementTopologyOf(soup TaggedSoup) ArrangementTopology {
	verts, tris := indexSoup(soup.Tris)
	owner := directedEdgeOwners(tris)
	comp := tagComponents(tris, soup.Tags, owner)
	var faces []ArrangementFace
	for _, g := range groupByComponent(comp) {
		faces = append(faces, arrangementFace(verts, tris, soup.Tags, owner, comp, g))
	}
	return ArrangementTopology{Verts: verts, Faces: faces}
}

// directedEdgeOwners maps every directed triangle edge (a,b) to its owning triangle.
// A watertight 2-manifold soup uses each directed edge exactly once.
func directedEdgeOwners(tris [][3]int) map[[2]int]int {
	owner := make(map[[2]int]int, len(tris)*3)
	for ti, t := range tris {
		for e := 0; e < 3; e++ {
			owner[[2]int{t[e], t[(e+1)%3]}] = ti
		}
	}
	return owner
}

// tagComponents labels each triangle with its output-face id: triangles that share an
// edge AND carry the same tag are one face (one original surface can survive as several
// disjoint patches, each its own face). Returns a dense component id per triangle.
func tagComponents(tris [][3]int, tags []int, owner map[[2]int]int) []int {
	uf := newUnionFind(len(tris))
	for ti, t := range tris {
		for e := 0; e < 3; e++ {
			if oj, ok := owner[[2]int{t[(e+1)%3], t[e]}]; ok && tags[oj] == tags[ti] {
				uf.union(ti, oj)
			}
		}
	}
	comp := make([]int, len(tris))
	ids := make(map[int]int)
	for ti := range tris {
		r := uf.find(ti)
		id, ok := ids[r]
		if !ok {
			id = len(ids)
			ids[r] = id
		}
		comp[ti] = id
	}
	return comp
}

// groupByComponent returns the triangle indices of each component, in component-id
// order.
func groupByComponent(comp []int) [][]int {
	n := 0
	for _, c := range comp {
		if c+1 > n {
			n = c + 1
		}
	}
	groups := make([][]int, n)
	for ti, c := range comp {
		groups[c] = append(groups[c], ti)
	}
	return groups
}
