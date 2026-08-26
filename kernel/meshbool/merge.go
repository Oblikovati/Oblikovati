// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math/big"

// Merging the boolean's triangle-soup result back into planar B-rep faces. The
// arrangement emits triangles; a B-rep wants maximal planar faces (with holes) so
// the body carries real face identity, not a facet fan. MergeFaces groups the
// coplanar edge-connected triangles into regions and traces each region's boundary
// loops, exactly (all decisions are exact predicates), collapsing collinear
// boundary vertices so each face edge is a single segment.

// Face is a merged planar region: one outer boundary loop (CCW seen along the
// region's outward normal) and zero or more hole loops, all coplanar.
type Face struct {
	Outer []Point
	Holes [][]Point
}

// MergeFaces groups the coplanar edge-connected triangles of a watertight soup into
// planar faces and returns them with clean (collinear-free) boundary loops.
func MergeFaces(soup [][3]Point) []Face {
	verts, tris := indexSoup(soup)
	var faces []Face
	for _, group := range groupCoplanar(verts, tris) {
		faces = append(faces, mergeGroup(verts, tris, group))
	}
	return faces
}

// indexSoup deduplicates the soup's vertices exactly (rational key) and returns the
// distinct vertex list plus the triangles as index triples.
func indexSoup(soup [][3]Point) ([]Point, [][3]int) {
	index := make(map[string]int)
	var verts []Point
	id := func(p Point) int {
		k := p.X.RatString() + "|" + p.Y.RatString() + "|" + p.Z.RatString()
		if i, ok := index[k]; ok {
			return i
		}
		index[k] = len(verts)
		verts = append(verts, p)
		return len(verts) - 1
	}
	tris := make([][3]int, len(soup))
	for t, s := range soup {
		tris[t] = [3]int{id(s[0]), id(s[1]), id(s[2])}
	}
	return verts, tris
}

// groupCoplanar unions triangles that share an edge and are coplanar, returning the
// triangle-index members of each maximal coplanar region.
func groupCoplanar(verts []Point, tris [][3]int) [][]int {
	uf := newUnionFind(len(tris))
	edgeTri := make(map[[2]int]int) // canonical edge -> a triangle already using it
	for ti, t := range tris {
		for e := range 3 {
			key := edgeKeyOf(t[e], t[(e+1)%3])
			if other, ok := edgeTri[key]; ok {
				if coplanarTris(verts, tris[other], t) {
					uf.union(ti, other)
				}
			} else {
				edgeTri[key] = ti
			}
		}
	}
	return uf.groups()
}

// coplanarTris reports whether two triangles lie in the same plane (all four
// distinct vertices coplanar suffices, tested exactly).
func coplanarTris(verts []Point, a, b [3]int) bool {
	return Orient3D(verts[a[0]], verts[a[1]], verts[a[2]], verts[b[0]]) == 0 &&
		Orient3D(verts[a[0]], verts[a[1]], verts[a[2]], verts[b[1]]) == 0 &&
		Orient3D(verts[a[0]], verts[a[1]], verts[a[2]], verts[b[2]]) == 0
}

// mergeGroup traces the boundary of one coplanar region into an outer loop and hole
// loops, collinear-collapsed and returned as Points.
func mergeGroup(verts []Point, tris [][3]int, group []int) Face {
	boundary := boundaryEdges(tris, group)
	loops := traceLoops(boundary)
	axis := planeAxis(triPoints(verts, tris[group[0]]))
	outer, holes := classifyLoops(loops, verts, axis)
	face := Face{Outer: cleanLoop(verts, outer, axis)}
	for _, h := range holes {
		face.Holes = append(face.Holes, cleanLoop(verts, h, axis))
	}
	return face
}

// boundaryEdges returns the directed edges of a group that have no opposite twin in
// the group — the region's boundary (interior edges appear in both directions).
func boundaryEdges(tris [][3]int, group []int) [][2]int {
	present := make(map[[2]int]bool)
	for _, ti := range group {
		t := tris[ti]
		for e := range 3 {
			present[[2]int{t[e], t[(e+1)%3]}] = true
		}
	}
	var out [][2]int
	for e := range present {
		if !present[[2]int{e[1], e[0]}] {
			out = append(out, e)
		}
	}
	return out
}

// classifyLoops picks the largest-area loop as the outer boundary and the rest as
// holes (a hole is enclosed by the outer, hence smaller).
func classifyLoops(loops [][]int, verts []Point, axis int) (outer []int, holes [][]int) {
	best := -1
	var bestArea *big.Rat
	for i, l := range loops {
		area := new(big.Rat).Abs(loopArea2(l, verts, axis))
		if best < 0 || area.Cmp(bestArea) > 0 {
			best, bestArea = i, area
		}
	}
	for i, l := range loops {
		if i == best {
			outer = l
		} else {
			holes = append(holes, l)
		}
	}
	return outer, holes
}

// cleanLoop drops collinear vertices (v[i-1],v[i],v[i+1] collinear) so each face
// edge is a single segment, and maps the surviving indices to Points.
func cleanLoop(verts []Point, loop []int, axis int) []Point {
	n := len(loop)
	var out []Point
	for i := range n {
		a, b, c := verts[loop[(i-1+n)%n]], verts[loop[i]], verts[loop[(i+1)%n]]
		if orient2(a, b, c, axis) != 0 {
			out = append(out, b)
		}
	}
	return out
}

func triPoints(verts []Point, t [3]int) [3]Point {
	return [3]Point{verts[t[0]], verts[t[1]], verts[t[2]]}
}

func edgeKeyOf(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// loopArea2 returns twice the exact signed area of a loop, projected on axis
// (shoelace via 2D cross products).
func loopArea2(loop []int, verts []Point, axis int) *big.Rat {
	sum := new(big.Rat)
	n := len(loop)
	for i := range n {
		u0, v0 := project(verts[loop[i]], axis)
		u1, v1 := project(verts[loop[(i+1)%n]], axis)
		sum.Add(sum, crossDiff(u0, v1, u1, v0))
	}
	return sum
}

// traceLoops chains a set of directed boundary edges into closed vertex loops. Each
// boundary vertex is balanced (equal in/out degree), so a greedy walk that consumes
// each edge once decomposes the boundary into its loops.
func traceLoops(edges [][2]int) [][]int {
	out := make(map[int][]int)
	for _, e := range edges {
		out[e[0]] = append(out[e[0]], e[1])
	}
	var loops [][]int
	for _, e := range edges {
		if len(out[e[0]]) == 0 {
			continue
		}
		loops = append(loops, walkLoop(e[0], out))
	}
	return loops
}

// walkLoop consumes edges from start until it returns to start, returning the loop's
// vertices (start first, no repeat of start).
func walkLoop(start int, out map[int][]int) []int {
	var loop []int
	for cur := start; len(out[cur]) > 0; {
		loop = append(loop, cur)
		last := len(out[cur]) - 1
		nxt := out[cur][last]
		out[cur] = out[cur][:last]
		cur = nxt
		if cur == start {
			break
		}
	}
	return loop
}

// unionFind is a disjoint-set over triangle indices for coplanar grouping.
type unionFind struct{ parent []int }

func newUnionFind(n int) *unionFind {
	p := make([]int, n)
	for i := range p {
		p[i] = i
	}
	return &unionFind{p}
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) {
	if ra, rb := u.find(a), u.find(b); ra != rb {
		u.parent[ra] = rb
	}
}

func (u *unionFind) groups() [][]int {
	m := make(map[int][]int)
	for i := range u.parent {
		r := u.find(i)
		m[r] = append(m[r], i)
	}
	out := make([][]int, 0, len(m))
	for _, g := range m {
		out = append(out, g)
	}
	return out
}
