// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	stdmath "math"
	"testing"
)

// The structured interior's three claims (#3549): it emits the SAME triangles the triangulator would
// have, it splits every cell on ONE named diagonal, and the block it emits is never crossed by the
// face's own boundary.

// TestTheStructuredInteriorEmitsWhatTheTriangulatorWould is the byte-identity claim at the mechanism
// rather than at the fingerprint: for the same covering, the kept triangles of the banded path and of
// the whole-covering triangulation are the same SET.
func TestTheStructuredInteriorEmitsWhatTheTriangulatorWould(t *testing.T) {
	b, loops, _ := windowedWallCover(t)
	whole := keyedTriangles(b.keepChartTriangles(constrainedTriangulationAll(b.xy, loops)))
	banded := keyedTriangles(b.keepChartTriangles(b.coveringTriangles(loops)))
	if len(whole) == 0 {
		t.Fatal("the whole-covering triangulation kept nothing; the test asserts nothing")
	}
	for k := range whole {
		if !banded[k] {
			t.Fatalf("the banded path lost a triangle the whole covering kept: %v", k)
		}
	}
	for k := range banded {
		if !whole[k] {
			t.Fatalf("the banded path invented a triangle the whole covering has not: %v", k)
		}
	}
}

// keyedTriangles is a triangle set keyed by its vertex indices, order-independent within a triangle.
func keyedTriangles(tris [][3]int) map[[3]int]bool {
	out := map[[3]int]bool{}
	for _, t := range tris {
		k := t
		for i := range 2 {
			for j := 0; j < 2-i; j++ {
				if k[j] > k[j+1] {
					k[j], k[j+1] = k[j+1], k[j]
				}
			}
		}
		out[k] = true
	}
	return out
}

// TestEveryStructuredCellIsSplitOnTheShorterDiagonal pins the rule the emission is allowed to use:
// (i+1,j) to (i,j+1), which the covering's shear makes the shorter diagonal of every cell and so the
// one a Delaunay triangulation of the lattice picks. Both halves are counter-clockwise.
func TestEveryStructuredCellIsSplitOnTheShorterDiagonal(t *testing.T) {
	b, _, _ := windowedWallCover(t)
	s := newStructuredInterior(b.node)
	cells := 0
	for si := range b.node.shifts {
		for i := range s.ci {
			for j := range s.cj {
				if !s.isBlock(si, i, j) {
					continue
				}
				cells++
				b.assertCellDiagonal(t, s.corners(si, i, j))
			}
		}
	}
	if cells == 0 {
		t.Fatal("the windowed wall's covering emitted no structured cell at all")
	}
}

// assertCellDiagonal checks that the chosen diagonal is the shorter one and that both halves are CCW.
func (b *chartCover) assertCellDiagonal(t *testing.T, c []int) {
	t.Helper()
	chosen := sqDist(b.xy[c[1]], b.xy[c[3]])
	other := sqDist(b.xy[c[0]], b.xy[c[2]])
	if chosen >= other {
		t.Fatalf("the cell was split on the longer diagonal: %g against %g", chosen, other)
	}
	for _, tri := range [][3]int{{c[0], c[1], c[3]}, {c[1], c[2], c[3]}} {
		if orient2d(b.xy[tri[0]], b.xy[tri[1]], b.xy[tri[2]]) <= 0 {
			t.Fatalf("a structured half is not counter-clockwise: %v", tri)
		}
	}
}

// TestTheBoundaryNeverCrossesAStructuredCell asserts the block's own premise. The block emits a cell
// as two triangles without asking the triangulator about it, which is sound only while the trim does
// not pass through that cell — and what guarantees it is a clearance measured in CELLS, not the
// clearance in boundary CHORDS that decided whether the corner nodes exist (blockSafeMargin).
func TestTheBoundaryNeverCrossesAStructuredCell(t *testing.T) {
	b, loops, _ := windowedWallCover(t)
	s := newStructuredInterior(b.node)
	for si := range b.node.shifts {
		for i := range s.ci {
			for j := range s.cj {
				if c := s.corners(si, i, j); s.isBlock(si, i, j) {
					b.assertNoRimInCell(t, c, loops)
				}
			}
		}
	}
}

// assertNoRimInCell fails when any constraint segment meets the cell's quad.
func (b *chartCover) assertNoRimInCell(t *testing.T, c []int, loops [][]int) {
	t.Helper()
	lo, hi := b.xy[c[0]], b.xy[c[0]]
	for _, v := range c {
		lo = [2]float64{stdmath.Min(lo[0], b.xy[v][0]), stdmath.Min(lo[1], b.xy[v][1])}
		hi = [2]float64{stdmath.Max(hi[0], b.xy[v][0]), stdmath.Max(hi[1], b.xy[v][1])}
	}
	for _, lp := range loops {
		for k := range lp {
			p, q := b.xy[lp[k]], b.xy[lp[(k+1)%len(lp)]]
			if segmentMeetsBox(p, q, lo, hi) && cellMeetsSegment(b.xy, c, p, q) {
				t.Fatalf("a rim segment %v→%v crosses the structured cell %v", p, q, c)
			}
		}
	}
}

// segmentMeetsBox is the cheap rejection: the segment's own box against the cell's.
func segmentMeetsBox(p, q, lo, hi [2]float64) bool {
	return stdmath.Min(p[0], q[0]) <= hi[0] && stdmath.Max(p[0], q[0]) >= lo[0] &&
		stdmath.Min(p[1], q[1]) <= hi[1] && stdmath.Max(p[1], q[1]) >= lo[1]
}

// cellMeetsSegment reports whether segment (p,q) crosses one of the cell's four sides or has an
// endpoint inside it.
func cellMeetsSegment(xy [][2]float64, c []int, p, q [2]float64) bool {
	poly := [][2]float64{xy[c[0]], xy[c[1]], xy[c[2]], xy[c[3]]}
	for k := range poly {
		if SegmentsCross(p, q, poly[k], poly[(k+1)%len(poly)]) {
			return true
		}
	}
	return pointInConvexQuad(poly, p) || pointInConvexQuad(poly, q)
}

// pointInConvexQuad reports whether p lies inside the counter-clockwise convex quad.
func pointInConvexQuad(poly [][2]float64, p [2]float64) bool {
	for k := range poly {
		if orient2d(poly[k], poly[(k+1)%len(poly)], p) < 0 {
			return false
		}
	}
	return true
}
