// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"

	"oblikovati.org/math"
)

// The vertex accumulator every COVERING-SPACE mesher shares (#1510, ADR-0061).
//
// A face on a periodic surface whose region wraps the seam has no simple polygon in one branch of the
// surface's parameters, so it is meshed in the covering space instead: the boundary and the interior
// grid are replicated a whole period either way, ONE constrained triangulation covers the lot, and each
// triangle is kept when its centroid falls in the canonical window and on the material side. Period
// copies of a boundary point are the SAME 3D point, so welding the kept triangles closes every seam.
//
// Two meshers do that — the periodic B-spline band cover and the chart-driven analytic mesher — and
// they differ in exactly three things: how a covering point's parameters fold back onto the surface for
// its normal, whether a replicated point is worth carrying at all, and what "material" means. Those are
// the three functions this struct takes; everything else below is the construction itself, held once.

// coverVertices accumulates a covering's vertices: exact 3D positions with their surface normals, the
// metric-scaled (u,v) the triangulation runs in, and the UNFOLDED (u,v) the canonical selection reads.
type coverVertices struct {
	// normalAt is the surface normal at a covering point. The owner folds the parameters back onto the
	// surface: the branch a replica lives on is bookkeeping, not geometry.
	normalAt func(u, v float64) math.Vector3
	// carry decides whether a replicated point is close enough to the canonical window to be worth
	// triangulating. nil carries every replica, which is what a cover of only three copies wants.
	carry  func(u, v float64) bool
	su, sv float64 // the (u,v) metric, so the triangulation runs in a space isometric to 3D
	pos    []math.Point3
	nrm    []math.Vector3
	xy     [][2]float64
	uu, vv []float64
	// at is the whole-period translate each vertex was laid at, as the INDEX of the shift that laid it.
	// It is what the canonical selection orders on: an integer the owner chose, not a float it derived
	// (cover_replica.go).
	at []int
	// locWeld is the distance below which two covering LOCATIONS are one location, and locIndex is the
	// index of the vertices already laid, bucketed on it. Zero locWeld means the owner has not asked
	// for the merge and every call lays a new vertex (Oblikovati/Oblikovati#3551).
	locWeld  float64
	locIndex map[[2]int64][]int
}

// MergeCoincidentLocations tells the accumulator that two covering locations nearer than weld are ONE
// location, and must therefore be one vertex. The owner sets it before laying anything.
//
// A covering is a 2D point set that a constrained triangulation runs on, and a CDT cannot recover a
// constraint incident to a vertex another vertex sits on top of: the two are indistinguishable to every
// orientation predicate the recovery walks with. Where the face's boundary TOUCHES ITSELF that is not a
// degenerate accident, it is the shape — the torus cut by its own tangent plane hands this mesher a
// single loop that passes the pinch twice, and its uv trace returns to the same (u,v) there — and where
// the boundary also wraps a whole period, the shift that carries the far pass back lands a third copy on
// the same spot. Measured on the R=5 r=2.5 figure eight at PropertyQuality: three rim vertices at
// (2π, π), one constraint segment never recovered, the mesh detouring round it through two interior
// nodes, the rim gate refusing the face, and the body shipping the whole torus with 408 free edges.
//
// The merge is a WELD, not a nudge: nothing moves, two records of one location become one record. The
// weld distance is the model's own (geom.ResolutionForPoints), so the only pairs it joins are pairs the
// kernel already calls coincident.
//
// Example: c.MergeCoincidentLocations(weldGrid([][]math.Point3{loop}))
func (c *coverVertices) MergeCoincidentLocations(weld float64) {
	c.locWeld, c.locIndex = weld, map[[2]int64][]int{}
}

// frameXY maps a covering (u, v) into the frame the triangulation runs in: the trim-local metric, then
// coverShear's tilt. It is a FUNCTION and not two open-coded expressions because two things read this
// frame and they must read the same one — add, which records c.xy, and the coincident-location index,
// which buckets and compares in it (#3551 + #3542). The index built in one frame and queried in another
// finds nothing, or finds the wrong neighbour.
//
// The shear does not move a location: identical (u, v) maps to identical (x, y) under any linear frame,
// so the pairs the merge joins are exactly the pairs it joined before. It changes the metric by the
// shear's own slope, 1/1024, which is four decades above nothing and eleven below the weld distance the
// merge uses.
func (c *coverVertices) frameXY(u, v float64) [2]float64 {
	return [2]float64{u*c.su + coverShear*v*c.sv, v * c.sv}
}

// locationOf is the index of the vertex already laid at (u, v), when the owner asked for the merge and
// one is there. The lookup walks the quantised cell and its eight neighbours, so a pair straddling a
// cell boundary — which two values of one location computed two ways easily do — is still found. The
// neighbours are walked in a fixed order and each cell holds its vertices in insertion order, so the
// vertex a lookup finds is the same on every run and every platform; nothing here iterates a map.
//
// The offsets are walked rather than materialised: this runs once per covering vertex, and a covering
// lays tens of thousands (review 2, M6).
func (c *coverVertices) locationOf(u, v float64) (int, bool) {
	if c.locIndex == nil {
		return 0, false
	}
	xy := c.frameXY(u, v)
	x, y := xy[0], xy[1]
	cx, cy := locationCell(x, y, c.locWeld)
	for dx := int64(-1); dx <= 1; dx++ {
		for dy := int64(-1); dy <= 1; dy++ {
			if i, ok := c.nearestIn([2]int64{cx + dx, cy + dy}, x, y); ok {
				return i, true
			}
		}
	}
	return 0, false
}

// nearestIn is the first vertex of one cell within the weld of (x, y).
func (c *coverVertices) nearestIn(cell [2]int64, x, y float64) (int, bool) {
	for _, i := range c.locIndex[cell] {
		if stdmath.Hypot(c.xy[i][0]-x, c.xy[i][1]-y) <= c.locWeld {
			return i, true
		}
	}
	return 0, false
}

// locationCell is the quantised cell (x, y) falls in.
func locationCell(x, y, weld float64) (int64, int64) {
	return int64(stdmath.Floor(x / weld)), int64(stdmath.Floor(y / weld))
}

// rememberLocation files a newly laid vertex under its own cell.
func (c *coverVertices) rememberLocation(i int) {
	if c.locIndex == nil {
		return
	}
	cx, cy := locationCell(c.xy[i][0], c.xy[i][1], c.locWeld)
	k := [2]int64{cx, cy}
	c.locIndex[k] = append(c.locIndex[k], i)
}

// add records one covering vertex laid at shift index at, and returns its index. When the owner asked
// for coincident locations to be merged and one is already there, the existing vertex is returned
// instead of a second one being laid at the same place (#3551).
// coverShear is the slope of the frame the covering TRIANGULATES in, and it exists because a
// RECTANGULAR lattice is the worst input a Delaunay triangulation can be given (#3549, #3542).
//
// The four corners of an axis-aligned rectangle are EXACTLY concyclic. A covering's interior is a grid,
// so without a shear every single cell asks the in-circle predicate a question whose answer is a tie —
// and an exact predicate answers a tie by running every stage of its adaptive escalation before it can
// say "zero". Measured on a 90 000-point regular grid: 2.144 s to insert. The SAME grid displaced by
// 1e-12 — cavity sizes within 8 %, walk length within 8 %, so neither of those is the cost — takes
// 208 ms. Tenfold, all of it the predicate.
//
// A shear is not a perturbation of geometry. Every covering VERTEX keeps its exact (u, v) and its exact
// 3-D point; what changes is the frame the triangulator reasons in, so a lattice cell is a
// parallelogram rather than a rectangle and its corners are not concyclic. The result is still a valid
// constrained triangulation of the same points with the same constraints. What it gives up is that the
// triangulation is Delaunay in the METRIC frame rather than in the sheared one — which for a tied quad
// means the diagonal is chosen by a cheap deterministic rule instead of by an exact predicate that has
// to run to the end to discover there was nothing to choose.
//
// IT ALSO CLOSES #3542. A covering's premise is that its two branch-window ends are the same
// triangulation, and they were not: near-cocircular quads at the seam flipped their diagonal
// differently at the two ends, leaving 6 of the sixteen near-pinch rows with 38 seam edges between them
// (cover_replica.go carries that measurement). With the ties gone the two ends agree. Swept over the
// near-pinch corpus — the eight JOIN bodies at both gate facetings — by rows carrying seam edges /
// seam edges in all / rows with an unbound rim segment:
//
//	shear    0      1/65536  1/16384  1/4096   1/1024   1/256    1/64
//	rows     6      0        0        0        0        0        0
//	edges    38     0        0        0        0        0        0
//	unbound  0      0        0        0        0        0        0
//
// A plateau four decades wide, and the value sits in the middle of it. Below the plateau the shear
// stops separating the tie; above it the working frame stops being near-isotropic, which is what
// trimMetricScale exists to make it. At 1/1024 the frame is within a tenth of a percent of the metric
// one, so triangle quality is unchanged, and the tessellation budget over the OCC fixtures runs 1.49 s
// against its 2.15 s ceiling with every imported periodic face charted, where an unsheared covering
// takes 3.3 s and more.
const coverShear = 1.0 / 1024 // tol:parametric (a slope of the triangulator's own frame, not a length)

// add records one covering vertex laid at shift index at, and returns its index.
func (c *coverVertices) add(p math.Point3, u, v float64, at int) int {
	if i, ok := c.locationOf(u, v); ok {
		return i
	}
	i := len(c.pos)
	c.pos = append(c.pos, p)
	c.nrm = append(c.nrm, c.normalAt(u, v))
	c.xy = append(c.xy, c.frameXY(u, v))
	c.uu, c.vv = append(c.uu, u), append(c.vv, v)
	c.at = append(c.at, at)
	c.rememberLocation(i)
	return i
}

// place records a vertex the owner may not want at this replica, returning -1 when it declines.
func (c *coverVertices) place(p math.Point3, u, v float64, at int) int {
	if c.carry != nil && !c.carry(u, v) {
		return -1
	}
	return c.add(p, u, v, at)
}

// addChain lays one boundary chain into the covering at a whole-period offset and returns its
// PER-SEGMENT constraints. Per-segment rather than as a closed loop because a boundary that wraps a
// period does not close in the covering space, and a spurious closing edge would constrain a chord the
// face does not have; the owner classifies triangles itself rather than by the loop-parity flood.
func (c *coverVertices) addChain(p3 []math.Point3, uv []math.Point2, du, dv float64, at int) [][]int {
	idx := make([]int, len(p3))
	for i := range p3 {
		idx[i] = c.place(p3[i], float64(uv[i].X)+du, float64(uv[i].Y)+dv, at)
	}
	return chainConstraints(idx)
}

// addRing lays a chain that DOES close in the covering space as one loop constraint (constrain wraps
// its last edge back to its first), returning its vertex index sequence.
//
// It calls add and not place, so the carry gate does not apply to it, and that asymmetry with addChain is
// deliberate rather than an oversight (#3527). A declined vertex comes back as -1, which an open chain
// can drop — chainConstraints skips any segment with one, because that segment's own replica a period
// along carries it — and a CLOSED ring cannot: a loop constraint with a hole in it is not the boundary
// the triangulation was handed. The only owner of addRing is the periodic B-spline cover
// (coverBuilder), which sets no carry at all, so the two are consistent today; a future owner that
// wants both has to answer the -1 question before it can use this.
func (c *coverVertices) addRing(p3 []math.Point3, uv []math.Point2, du, dv float64, at int) []int {
	idx := make([]int, len(p3))
	for i := range p3 {
		idx[i] = c.add(p3[i], float64(uv[i].X)+du, float64(uv[i].Y)+dv, at)
	}
	return idx
}

// chainConstraints is a chain's consecutive index pairs, skipping any segment an end of which the
// carry gate declined — that segment's own replica a period along carries it — and any segment whose
// two ends are ONE vertex, which is what a boundary step shorter than the location weld becomes once
// coincident locations are merged (#3551). A self-loop is not a constraint a triangulation can hold.
func chainConstraints(idx []int) [][]int {
	var segs [][]int
	for i := 0; i+1 < len(idx); i++ {
		if idx[i] >= 0 && idx[i+1] >= 0 && idx[i] != idx[i+1] {
			segs = append(segs, []int{idx[i], idx[i+1]})
		}
	}
	return segs
}

// centroid is a triangle's mean UNFOLDED (u,v) — the point the canonical selection reads.
func (c *coverVertices) centroid(t [3]int) (u, v float64) {
	return (c.uu[t[0]] + c.uu[t[1]] + c.uu[t[2]]) / 3, (c.vv[t[0]] + c.vv[t[1]] + c.vv[t[2]]) / 3
}

// keepCanonical keeps each triangle the predicate calls material, ONE translate of each.
//
// The predicate is the CANDIDATE filter and no longer the de-duplication, which is the correction
// #3518 landed. A half-open window de-duplicates a POINT exactly — of p and its whole-period
// translates exactly one satisfies lo <= x < hi — and the premise this doc used to state is that the
// same holds for a triangle. It does not: a replica's centroid is recomputed from its own shifted
// vertices, so it equals the original's plus a period only to within rounding, and a triangle whose
// centroid lands on the window's edge is taken twice or not at all (measured, see cover_replica.go).
//
// So the predicate is asked with a CLOSED window, which offers at least one translate of every
// triangle, and keepOneReplicaEach picks exactly one of them on the integer shift each vertex was
// laid at.
func (c *coverVertices) keepCanonical(tris [][3]int, material func(u, v float64) bool) [][3]int {
	keep := make([]bool, len(tris))
	for i, t := range tris {
		u, v := c.centroid(t)
		keep[i] = material(u, v)
	}
	c.keepOneReplicaEach(tris, keep)
	return selectTriangles(tris, keep)
}

// selectTriangles is the marked subset, in the order the triangulation produced it.
func selectTriangles(tris [][3]int, keep []bool) [][3]int {
	out := make([][3]int, 0, len(tris))
	for i, t := range tris {
		if keep[i] {
			out = append(out, t)
		}
	}
	return out
}
