// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"sort"

	"oblikovati.org/kernel/brep"
	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// A planar face whose inner loops OVERLAP each other (e.g. a grill boundary whose rib and spar
// holes cross) is geometrically degenerate for plain earcut: feeding the raw overlapping loops
// over- or under-counts the void and used to collapse a ring (#873). unionTris instead computes
// the true union of the holes via the 2D planar arrangement (brep.Arrange) — which splits the
// loops at their crossings into clean, non-overlapping cells — keeps the material cells (outer
// minus the holes' union), and ear-clips each. The result covers exactly outer − union(holes),
// so mass properties and display are area-exact on overlapping holes too.

// unionHoledMesh tessellates a planar face whose hole loops overlap: it projects the face to its
// plane with an orthonormal basis (an exact inverse, so original vertices round-trip and the
// crossing points the union introduces land back on the plane), unions the holes, and lifts the
// triangulation to 3D.
func unionHoledMesh(outer3D []math.Point3, holes3D [][]math.Point3, normal math.Vector3) *Mesh {
	origin := outer3D[0]
	e1, e2 := planeBasis(normal)
	to2D := func(p math.Point3) math.Point2 {
		d := origin.VectorTo(p)
		return math.P2(d.Dot(e1), d.Dot(e2))
	}
	to3D := func(q math.Point2) math.Point3 {
		return origin.TranslateBy(e1.Scale(q.X).Add(e2.Scale(q.Y)))
	}
	outer2D := make([]math.Point2, len(outer3D))
	for i, p := range outer3D {
		outer2D[i] = to2D(p)
	}
	holes2D := make([][]math.Point2, len(holes3D))
	for i, h := range holes3D {
		holes2D[i] = project2D(h, to2D)
	}
	verts2D, tris := unionTris(outer2D, holes2D)
	m := &Mesh{}
	for _, q := range verts2D {
		m.AddVertex(to3D(q), normal)
	}
	for _, t := range tris {
		m.AddTriangle(t[0], t[1], t[2])
	}
	return m
}

// unionTris triangulates outer minus the union of holes, returning a fresh vertex list (the
// arrangement may introduce crossing points absent from the inputs) and triangles indexing it.
// It is used only when the holes overlap; non-overlapping faces stay on the direct earcut path.
func unionTris(outer []math.Point2, holes [][]math.Point2) ([]math.Point2, [][3]int) {
	segs := loopSegments(outer)
	for _, h := range holes {
		segs = append(segs, loopSegments(h)...)
	}
	var verts []math.Point2
	var tris [][3]int
	for _, cell := range brep.Arrange(segs) {
		if !cellIsMaterial(cell, holes) {
			continue
		}
		base := len(verts)
		cellVerts, cellTris := tessellateCell(cell)
		verts = append(verts, cellVerts...)
		for _, t := range cellTris {
			tris = append(tris, [3]int{t[0] + base, t[1] + base, t[2] + base})
		}
	}
	return verts, tris
}

// loopSegments returns a closed loop's undirected edges as point pairs.
func loopSegments(loop []math.Point2) [][2]math.Point2 {
	if len(loop) < 2 {
		return nil
	}
	segs := make([][2]math.Point2, len(loop))
	for i := range loop {
		segs[i] = [2]math.Point2{loop[i], loop[(i+1)%len(loop)]}
	}
	return segs
}

// tessellateCell ear-clips one arrangement cell. A material cell's holes can be several abutting
// void sub-cells (the arrangement splits an overlapping-hole union into pieces); they are merged
// into their true outline first, since abutting hole loops share edges and break earcut.
func tessellateCell(cell brep.Face2D) ([]math.Point2, [][3]int) {
	holes := mergeAbuttingHoles(cell.Holes)
	verts := append([]math.Point2(nil), cell.Outer...)
	for _, h := range holes {
		verts = append(verts, h...)
	}
	return verts, earcut(cell.Outer, holes)
}

// MergeAbuttingLoops fuses closed loops that SHARE edges into their union-boundary outline(s): a
// directed edge that appears in two loops traversed oppositely is interior and dropped; the surviving
// edges chain into the union's boundary loops. Disjoint loops pass through unchanged. Every input loop
// MUST carry the same winding, so an interior edge cancels; DCEL cells of one sketch (all wound CCW)
// satisfy this. The extrude feature uses it to dissolve a region an upstream arrangement over-split
// into abutting cells — a slot plus its corner-relief discs, sharing arcs — into ONE prism, instead of
// several prisms cut one at a time that leave coincident interior walls (#38). Thin wrapper over the
// hole-merge primitive so its exact behaviour is shared, not duplicated.
func MergeAbuttingLoops(loops [][]math.Point2) [][]math.Point2 {
	return mergeAbuttingHoles(loops)
}

// mergeAbuttingHoles fuses hole loops that share edges into their boundary outline: a directed
// edge shared by two abutting loops (traversed oppositely) is interior and dropped; the surviving
// edges chain into the union's boundary loops. A single (or disjoint) hole set is returned as-is.
func mergeAbuttingHoles(holes [][]math.Point2) [][]math.Point2 {
	if len(holes) <= 1 {
		return holes
	}
	w := newHoleWelder(holeGrid(holes))
	dirCount := map[[2]int]int{}
	for _, h := range holes {
		idx := make([]int, len(h))
		for i, p := range h {
			idx[i] = w.add(p)
		}
		for i := range idx {
			a, b := idx[i], idx[(i+1)%len(idx)]
			if a != b {
				dirCount[[2]int{a, b}]++
			}
		}
	}
	return chainBoundaryEdges(boundaryEdges(dirCount), w.points)
}

// boundaryEdges keeps the directed edges whose reverse is absent — the outline of the union (an
// edge shared by two abutting loops appears in both directions and cancels).
//
// Emitted in sorted order: Go randomizes map iteration, and this slice seeds the adjacency
// chainBoundaryEdges walks, where the append order decides how edges pair at a touching vertex.
func boundaryEdges(dirCount map[[2]int]int) [][2]int {
	var keep [][2]int
	for _, e := range sortedDirEdges(dirCount) {
		if dirCount[[2]int{e[1], e[0]}] == 0 {
			for k := 0; k < dirCount[e]; k++ {
				keep = append(keep, e)
			}
		}
	}
	return keep
}

// sortedDirEdges returns the directed edges in ascending (from, to) order.
func sortedDirEdges(dirCount map[[2]int]int) [][2]int {
	keys := make([][2]int, 0, len(dirCount))
	for e := range dirCount {
		keys = append(keys, e)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	return keys
}

// chainBoundaryEdges links directed edges head-to-tail into closed loops.
//
// Loop starts are taken in sorted order for the same reason boundaryEdges sorts: where two hole
// loops touch at a single welded vertex, the start the walk picks decides which outgoing edge pairs
// with which incoming one, so map order changed the LOOPS, not just their order (#23).
func chainBoundaryEdges(edges [][2]int, pts []math.Point2) [][]math.Point2 {
	next := make(map[int][]int, len(edges))
	for _, e := range edges {
		next[e[0]] = append(next[e[0]], e[1])
	}
	starts := make([]int, 0, len(next))
	for v := range next {
		starts = append(starts, v)
	}
	sort.Ints(starts)
	var loops [][]math.Point2
	for _, start := range starts {
		for len(next[start]) > 0 {
			loop := traceBoundaryLoop(start, next)
			if len(loop) >= 3 {
				ring := make([]math.Point2, len(loop))
				for i, v := range loop {
					ring[i] = pts[v]
				}
				loops = append(loops, ring)
			}
		}
	}
	return loops
}

// traceBoundaryLoop walks next-edges from start until it returns, consuming each edge once.
func traceBoundaryLoop(start int, next map[int][]int) []int {
	loop := []int{start}
	for cur := start; ; {
		outs := next[cur]
		if len(outs) == 0 {
			return nil // open chain (degenerate) — abandon
		}
		nxt := outs[0]
		next[cur] = outs[1:]
		if nxt == start {
			return loop
		}
		loop = append(loop, nxt)
		cur = nxt
	}
}

// holeWelder merges coincident hole vertices onto a shared index list, snapping to a
// model-relative arrangement weld grid (ADR-0042) the caller derives from the holes' size.
type holeWelder struct {
	index  map[[2]int64]int
	points []math.Point2
	grid   float64
}

func newHoleWelder(grid float64) *holeWelder {
	return &holeWelder{index: map[[2]int64]int{}, grid: grid}
}

func (w *holeWelder) add(p math.Point2) int {
	k := [2]int64{int64(stdmath.Round(p.X / w.grid)), int64(stdmath.Round(p.Y / w.grid))}
	if i, ok := w.index[k]; ok {
		return i
	}
	w.index[k] = len(w.points)
	w.points = append(w.points, p)
	return len(w.points) - 1
}

// holeGrid is the model-relative arrangement weld grid for a 2D hole set (ADR-0042):
// the weld resolution of the loops' bounding-box diagonal, floored so a degenerate set
// still welds at a positive grid. At a ~1 cm arrangement this is the historical 1e-9.
func holeGrid(holes [][]math.Point2) float64 {
	var pts []math.Point2
	for _, h := range holes {
		pts = append(pts, h...)
	}
	return geom.ResolutionForPoints2D(pts).Weld()
}

// cellIsMaterial reports whether an arrangement cell is solid material — its interior lies
// outside every original hole. A cell wholly inside the holes' union is a void and is dropped.
// The interior is sampled just inside the cell's (CCW) outer edge, which lies in the cell's
// material region (between the outer boundary and any holes), unlike a triangle centroid that
// can fall in a hole.
func cellIsMaterial(cell brep.Face2D, holes [][]math.Point2) bool {
	if len(cell.Outer) < 3 {
		return false
	}
	p := probeInsideOuter(cell.Outer)
	for _, h := range holes {
		if pointInLoop2D(p, h) {
			return false
		}
	}
	return true
}

// probeInsideOuter returns a point just inside a CCW loop's first edge (the interior side),
// matching the arrangement's own nesting probe.
func probeInsideOuter(loop []math.Point2) math.Point2 {
	a, b := loop[0], loop[1]
	e := a.VectorTo(b)
	mid := math.P2((a.X+b.X)/2, (a.Y+b.Y)/2)
	return mid.TranslateBy(math.V2(-e.Y, e.X).Scale(1e-4)) // left normal = interior of a CCW loop
}

// pointInLoop2D is a ray-cast point-in-polygon test for a closed loop.
func pointInLoop2D(p math.Point2, loop []math.Point2) bool {
	in := false
	for i, j := 0, len(loop)-1; i < len(loop); j, i = i, i+1 {
		yi, yj := loop[i].Y, loop[j].Y
		if (yi > p.Y) != (yj > p.Y) {
			x := loop[i].X + (p.Y-yi)/(yj-yi)*(loop[j].X-loop[i].X)
			if p.X < x {
				in = !in
			}
		}
	}
	return in
}

// holesOverlap reports whether any two hole loops overlap — their edges cross, or a vertex of
// one lies strictly inside another (so their union is smaller than the sum of their areas). Only
// then is the costlier arrangement-union tessellation needed.
func holesOverlap(holes [][]math.Point2) bool {
	for i := range holes {
		for j := i + 1; j < len(holes); j++ {
			if loopsOverlap(holes[i], holes[j]) {
				return true
			}
		}
	}
	return false
}

// loopsOverlap reports whether two closed loops share interior area.
func loopsOverlap(a, b []math.Point2) bool {
	for _, sa := range loopSegments(a) {
		for _, sb := range loopSegments(b) {
			if segmentsCross(xy(sa[0]), xy(sa[1]), xy(sb[0]), xy(sb[1])) {
				return true
			}
		}
	}
	return pointInLoop2D(a[0], b) || pointInLoop2D(b[0], a)
}

// xy converts a Point2 to the [2]float64 form the shared geometry predicates use.
func xy(p math.Point2) [2]float64 { return [2]float64{p.X, p.Y} }
