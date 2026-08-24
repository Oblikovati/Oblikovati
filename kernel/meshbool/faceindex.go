// SPDX-License-Identifier: GPL-2.0-only

package meshbool

import "math"

// A bounding-box spatial index over a mesh's faces. Co-refinement pairing and
// coplanar-partner classification are otherwise O(facesA x facesB) — every face
// tested against every other, with exact big.Rat predicates in the inner loop.
// Two triangles can only intersect, or be coincident, if their bounding boxes
// overlap, so this index answers "which faces could overlap this box" and the exact
// tests run only on the handful of real candidates, turning the quadratic scans
// near-linear. Boxes are float64 (from rounding the exact coordinates); the index
// only PRUNES — an exact predicate still decides every surviving pair — so the
// rounding cannot change a result, only widen the candidate set.

type aabb struct{ lo, hi [3]float64 }

func faceAABB(t [3]Point) aabb {
	p := [3]math3{m3(t[0]), m3(t[1]), m3(t[2])}
	var b aabb
	for k := 0; k < 3; k++ {
		b.lo[k] = math.Min(p[0][k], math.Min(p[1][k], p[2][k]))
		b.hi[k] = math.Max(p[0][k], math.Max(p[1][k], p[2][k]))
	}
	return b
}

// math3 is a face vertex as float64 coordinates for bounding-box arithmetic.
type math3 [3]float64

func m3(p Point) math3 {
	q := p.Round()
	return math3{q.X, q.Y, q.Z}
}

func (a aabb) overlaps(b aabb) bool {
	for k := 0; k < 3; k++ {
		if a.hi[k] < b.lo[k] || b.hi[k] < a.lo[k] {
			return false
		}
	}
	return true
}

// faceGrid buckets a mesh's face bounding boxes into a uniform grid.
type faceGrid struct {
	cell    float64
	boxes   []aabb
	buckets map[[3]int][]int
	visited []int // per-face generation stamp, so candidate() needs no per-call alloc
	gen     int
}

// newFaceGrid indexes mesh's faces. Cell size targets roughly one face per cell.
func newFaceGrid(mesh [][3]Point) *faceGrid {
	g := &faceGrid{buckets: make(map[[3]int][]int), boxes: make([]aabb, len(mesh)), visited: make([]int, len(mesh))}
	if len(mesh) == 0 {
		g.cell = 1
		return g
	}
	all := faceAABB(mesh[0])
	for _, t := range mesh {
		b := faceAABB(t)
		for k := 0; k < 3; k++ {
			all.lo[k] = math.Min(all.lo[k], b.lo[k])
			all.hi[k] = math.Max(all.hi[k], b.hi[k])
		}
	}
	g.cell = gridCell(all, len(mesh))
	for i, t := range mesh {
		b := faceAABB(t)
		g.boxes[i] = b
		g.forCells(b, func(c [3]int) { g.buckets[c] = append(g.buckets[c], i) })
	}
	return g
}

// candidates returns the indices of faces whose bounding box overlaps query.
func (g *faceGrid) candidates(query aabb) []int {
	g.gen++
	var out []int
	g.forCells(query, func(c [3]int) {
		for _, i := range g.buckets[c] {
			if g.visited[i] != g.gen && g.boxes[i].overlaps(query) {
				g.visited[i] = g.gen
				out = append(out, i)
			}
		}
	})
	return out
}

// forCells calls fn for each grid cell the box spans.
func (g *faceGrid) forCells(b aabb, fn func([3]int)) {
	lo := cellIndex(b.lo, g.cell)
	hi := cellIndex(b.hi, g.cell)
	for x := lo[0]; x <= hi[0]; x++ {
		for y := lo[1]; y <= hi[1]; y++ {
			for z := lo[2]; z <= hi[2]; z++ {
				fn([3]int{x, y, z})
			}
		}
	}
}

func cellIndex(p [3]float64, cell float64) [3]int {
	return [3]int{int(math.Floor(p[0] / cell)), int(math.Floor(p[1] / cell)), int(math.Floor(p[2] / cell))}
}

// gridCell picks a cell size ~ the mesh diagonal divided by the cube root of the
// face count (about one face per cell), with a positive floor for a degenerate box.
func gridCell(all aabb, n int) float64 {
	var d float64
	for k := 0; k < 3; k++ {
		s := all.hi[k] - all.lo[k]
		d += s * s
	}
	diag := math.Sqrt(d)
	if diag == 0 {
		return 1
	}
	return diag / math.Cbrt(float64(n))
}
