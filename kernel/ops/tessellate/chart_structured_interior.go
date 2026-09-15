// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"sort"
)

// The covering's STRUCTURED interior (#3549).
//
// A covering is a boundary band around a GRID, and a grid is not a point cloud. Every interior node
// sits at a station pair (i, j) whose four neighbours are known before anything is computed, so the
// quad mesh those cells make is already decided — yet the whole covering used to be handed to the
// incremental constrained triangulator, which rediscovered that lattice one point at a time. Measured
// on the J3 host torus at PropertyQuality: 788 255 covering points, 127 s to insert them and 40.8
// million point-location walk steps to find where, for a mesh whose interior the grid already names.
//
// So the triangulator is given the BAND only — the cells the trim actually cuts, plus a margin — and
// the interior block is emitted directly as quads. The block's outline goes in as a constraint, so no
// band triangle crosses into it; the triangulator fills the emptied block with spanning triangles,
// and those are discarded by their centroid. A triangle cannot cross a recovered constraint, so it
// lies wholly inside the block or wholly outside it, and its centroid says which without ambiguity
// even where the block is non-convex.
//
// THE DIAGONAL IS A RULE, NOT AN ITERATION ORDER. Each cell is split from (i+1, j) to (i, j+1),
// always. That is the diagonal a Delaunay triangulation of the sheared lattice picks anyway: the
// covering triangulates in the frame x = u·su + coverShear·v·sv, y = v·sv (covering_vertices.go), in
// which a grid cell is a PARALLELOGRAM with edge vectors a = (Δu·su, 0) and b = (coverShear·Δv·sv,
// Δv·sv). Its two diagonals are a+b and a−b, so |a+b|² − |a−b|² = 4 a·b = 4·coverShear·Δu·Δv·su·sv,
// which is strictly positive for every cell of an ascending station grid. a−b — the (i+1, j) to
// (i, j+1) diagonal — is therefore the shorter one, uniformly, for any cell sizes. The shear that
// removed the in-circle TIE (#3542) is the same thing that makes the tie's resolution nameable here.

// gridNodeIndex is which covering vertex each interior grid node was laid at, by period shift and
// station pair, or -1 where no node was laid (the station was not material, stood too close to the
// rim, or fell outside the replication pad). Each node also carries whether it stands a WHOLE CELL
// clear of every boundary chain, which is what decides where the block ends (blockSafeMargin).
type gridNodeIndex struct {
	shifts, ni, nj int
	at             []int
	clear          []bool
}

// newGridNodeIndex allocates an empty lattice of shifts × ni × nj nodes.
func newGridNodeIndex(shifts, ni, nj int) *gridNodeIndex {
	at := make([]int, shifts*ni*nj)
	for k := range at {
		at[k] = -1
	}
	return &gridNodeIndex{shifts: shifts, ni: ni, nj: nj, at: at, clear: make([]bool, len(at))}
}

// record stores station (i, j)'s covering vertex at each period shift, and whether the node stands a
// whole cell clear of the rim.
func (g *gridNodeIndex) record(i, j int, at []int, clear bool) {
	for si, v := range at {
		g.at[(si*g.ni+i)*g.nj+j] = v
		g.clear[(si*g.ni+i)*g.nj+j] = clear
	}
}

// vertex is the covering vertex of node (i, j) at shift si, or -1 when there is none.
func (g *gridNodeIndex) vertex(si, i, j int) int {
	if si < 0 || i < 0 || j < 0 || si >= g.shifts || i >= g.ni || j >= g.nj {
		return -1
	}
	return g.at[(si*g.ni+i)*g.nj+j]
}

// blockCorner is the covering vertex of a node that may corner a STRUCTURED cell: one that exists and
// stands a whole cell clear of every boundary chain. -1 otherwise.
func (g *gridNodeIndex) blockCorner(si, i, j int) int {
	v := g.vertex(si, i, j)
	if v < 0 || !g.clear[(si*g.ni+i)*g.nj+j] {
		return -1
	}
	return v
}

// nodeClearance answers both of the covering's clearance questions in one pass: whether the node
// stands a whole cell clear of every boundary chain, so the cells around it may be emitted
// structurally, and whether it stands clear enough to be laid at all. The wider test implies the
// narrower one, so the narrower is only asked where the wider fails — the rim's own band, a small
// fraction of the grid.
func (b *chartCover) nodeClearance(chains []chartChain, i, j int, u, v, margin float64) (clear, keep bool) {
	if b.clearOfChains(chains, u, v, b.blockSafeMargin(i, j, margin)) {
		return true, true
	}
	return false, b.clearOfChains(chains, u, v, margin)
}

// blockSafeMargin is how far a grid node must stand from every boundary chain for the cells around it
// to be emitted structurally rather than triangulated: a WHOLE CELL, as a 3D length, and never less
// than the clearance that decided the node exists at all.
//
// One cell is the smallest margin that makes the block's premise true. The block emits cell (i,j) as
// two triangles without ever asking the triangulator about it, which is sound only if the trim does
// not pass through it; a node clear by one cell in every direction cannot have the boundary between it
// and its neighbour. Less than that and four surviving corners are no longer a statement about the
// cell they bound, because the clearance those corners passed is measured in the BOUNDARY's chords and
// a chord can be far shorter than a cell (the merged cocylindrical wall carries 320 chords of 0.074 mm
// around a face 4 mm tall).
func (b *chartCover) blockSafeMargin(i, j int, nodeMargin float64) float64 {
	return stdmath.Max(nodeMargin, b.localCellDiagonal(i, j))
}

// localCellDiagonal is the diagonal, as a 3D length, of the largest grid cell meeting node (i, j).
func (b *chartCover) localCellDiagonal(i, j int) float64 {
	return stdmath.Hypot(widestAdjacentGap(b.us, i)*b.su, widestAdjacentGap(b.vs, j)*b.sv)
}

// widestAdjacentGap is the larger of the two station gaps either side of index i (0 for an axis with
// no cell at all).
func widestAdjacentGap(ps []float64, i int) float64 {
	gap := 0.0
	if i > 0 {
		gap = ps[i] - ps[i-1]
	}
	if i+1 < len(ps) {
		gap = stdmath.Max(gap, ps[i+1]-ps[i])
	}
	return gap
}

// structuredInterior is the block of grid cells emitted as a quad mesh, and the outline that separates
// it from the band the triangulator gets. Cell (si, i, j) is the one whose corners are stations i and
// i+1 by j and j+1 at period shift si.
type structuredInterior struct {
	g      *gridNodeIndex
	ci, cj int
	block  []bool
}

// newStructuredInterior marks every cell all four of whose corners exist and stand a whole cell clear
// of the face's own boundary. That second condition is the band, and it is a DISTANCE rather than a
// count of lattice steps: the clearance that decides whether a node exists at all is measured in
// CHORDS of the boundary (chartBoundaryClearance), and a chord can be shorter than a cell, so four
// surviving corners do not by themselves say the trim misses the cell between them.
func newStructuredInterior(g *gridNodeIndex) *structuredInterior {
	s := &structuredInterior{g: g, ci: max(g.ni-1, 0), cj: max(g.nj-1, 0)}
	s.block = s.fullCells()
	return s
}

// cellAt is a cell's index into the per-cell flag arrays.
func (s *structuredInterior) cellAt(si, i, j int) int {
	return (si*s.ci+i)*s.cj + j
}

// fullCells marks each cell all four of whose corners may corner a structured cell.
func (s *structuredInterior) fullCells() []bool {
	full := make([]bool, s.g.shifts*s.ci*s.cj)
	for si := range s.g.shifts {
		for i := range s.ci {
			for j := range s.cj {
				full[s.cellAt(si, i, j)] = s.corners(si, i, j) != nil
			}
		}
	}
	return full
}

// corners is a cell's four corner vertices in the order (i,j), (i+1,j), (i+1,j+1), (i,j+1), or nil
// when any of them cannot corner a structured cell.
func (s *structuredInterior) corners(si, i, j int) []int {
	c := []int{
		s.g.blockCorner(si, i, j), s.g.blockCorner(si, i+1, j),
		s.g.blockCorner(si, i+1, j+1), s.g.blockCorner(si, i, j+1),
	}
	for _, v := range c {
		if v < 0 {
			return nil
		}
	}
	return c
}

// isBlock reports whether cell (si, i, j) is emitted structurally.
func (s *structuredInterior) isBlock(si, i, j int) bool {
	if si < 0 || i < 0 || j < 0 || si >= s.g.shifts || i >= s.ci || j >= s.cj {
		return false
	}
	return s.block[s.cellAt(si, i, j)]
}

// triangles is the block as a quad mesh, two CCW triangles per cell on the (i+1,j)–(i,j+1) diagonal.
func (s *structuredInterior) triangles() [][3]int {
	var out [][3]int
	for si := range s.g.shifts {
		for i := range s.ci {
			for j := range s.cj {
				if !s.isBlock(si, i, j) {
					continue
				}
				c := s.corners(si, i, j)
				out = append(out, [3]int{c[0], c[1], c[3]}, [3]int{c[1], c[2], c[3]})
			}
		}
	}
	return out
}

// outline is the block's boundary as constraint segments: every cell edge with a block cell on one
// side and no block cell on the other. Both endpoints of such an edge touch the non-block side, so
// neither is absorbed, and the triangulator always has them.
func (s *structuredInterior) outline() [][]int {
	var segs [][]int
	for si := range s.g.shifts {
		for i := range s.ci {
			for j := range s.cj {
				segs = append(segs, s.cellOutline(si, i, j)...)
			}
		}
	}
	return segs
}

// cellOutline is one cell's edges that face a non-block neighbour.
func (s *structuredInterior) cellOutline(si, i, j int) [][]int {
	if !s.isBlock(si, i, j) {
		return nil
	}
	c := s.corners(si, i, j)
	sides := [4][2]int{{c[0], c[1]}, {c[1], c[2]}, {c[2], c[3]}, {c[3], c[0]}}
	across := [4][2]int{{i, j - 1}, {i + 1, j}, {i, j + 1}, {i - 1, j}}
	var segs [][]int
	for k, n := range across {
		if !s.isBlock(si, n[0], n[1]) {
			segs = append(segs, []int{sides[k][0], sides[k][1]})
		}
	}
	return segs
}

// absorbed marks the covering vertices the block emitted itself: a grid node all four of whose cells
// are block cells carries no band triangle, so the triangulator never sees it. Everything else —
// every rim vertex, every ear centre, every node on the block's outline — stays.
func (s *structuredInterior) absorbed(n int) []bool {
	out := make([]bool, n)
	for si := range s.g.shifts {
		for i := range s.g.ni {
			for j := range s.g.nj {
				v := s.g.vertex(si, i, j)
				if v >= 0 && v < n && s.nodeIsEnclosed(si, i, j) {
					out[v] = true
				}
			}
		}
	}
	return out
}

// nodeIsEnclosed reports whether all four cells meeting node (i, j) are block cells.
func (s *structuredInterior) nodeIsEnclosed(si, i, j int) bool {
	return s.isBlock(si, i-1, j-1) && s.isBlock(si, i, j-1) &&
		s.isBlock(si, i-1, j) && s.isBlock(si, i, j)
}

// coveringTriangles is the covering's triangle set: the boundary BAND from the constrained
// triangulator, plus the structured interior emitted directly as quads. It is what replaces the one
// call that used to hand every covering point to the triangulator.
func (b *chartCover) coveringTriangles(loops [][]int) [][3]int {
	if b.node == nil {
		return constrainedTriangulationAll(b.xy, loops)
	}
	s := newStructuredInterior(b.node)
	quads := s.triangles()
	if len(quads) == 0 {
		return constrainedTriangulationAll(b.xy, loops) // no cell is far enough from the trim
	}
	return append(b.bandTriangulation(loops, s), quads...)
}

// bandTriangulation triangulates the covering MINUS the points the block absorbed, with the block's
// outline constrained alongside the rim, and drops the spanning triangles the emptied block fills with.
func (b *chartCover) bandTriangulation(loops [][]int, s *structuredInterior) [][3]int {
	sub, fwd, back := pointsExcept(b.xy, s.absorbed(len(b.xy)))
	cons := append(remapConstraints(loops, fwd), remapConstraints(s.outline(), fwd)...)
	var out [][3]int
	for _, t := range constrainedTriangulationAll(sub, cons) {
		w := [3]int{back[t[0]], back[t[1]], back[t[2]]}
		if !b.insideBlock(w, s) {
			out = append(out, w)
		}
	}
	return out
}

// insideBlock reports whether a band triangle fell inside the structured block. A triangle cannot
// cross a recovered constraint and the block's outline is one, so it lies wholly inside the block or
// wholly outside it: its centroid decides for the whole triangle even where the block is not convex.
func (b *chartCover) insideBlock(t [3]int, s *structuredInterior) bool {
	u, v := b.centroid(t)
	for si, sh := range b.r.shifts() {
		if s.isBlock(si, stationCell(b.us, u-sh[0]), stationCell(b.vs, v-sh[1])) {
			return true
		}
	}
	return false
}

// stationCell is the index of the axis cell containing p, or -1 when p lies off the axis.
func stationCell(ps []float64, p float64) int {
	i := sort.SearchFloat64s(ps, p) - 1
	if i < 0 || i >= len(ps)-1 {
		return -1
	}
	return i
}

// pointsExcept is the points the block did not absorb, with the forward map from covering index to the
// compacted one (-1 for an absorbed point) and its inverse.
func pointsExcept(xy [][2]float64, drop []bool) (sub [][2]float64, fwd, back []int) {
	fwd = make([]int, len(xy))
	for i, p := range xy {
		if drop[i] {
			fwd[i] = -1
			continue
		}
		fwd[i] = len(sub)
		sub = append(sub, p)
		back = append(back, i)
	}
	return sub, fwd, back
}

// remapConstraints carries constraint index sequences onto the compacted point set. A sequence naming
// an absorbed point is dropped whole rather than shortened — a shortened constraint is a chord the
// face does not have — and none should: an absorbed point is strictly inside the block.
func remapConstraints(loops [][]int, fwd []int) [][]int {
	out := make([][]int, 0, len(loops))
	for _, lp := range loops {
		m := make([]int, 0, len(lp))
		for _, v := range lp {
			if fwd[v] >= 0 {
				m = append(m, fwd[v])
			}
		}
		if len(m) == len(lp) {
			out = append(out, m)
		}
	}
	return out
}
