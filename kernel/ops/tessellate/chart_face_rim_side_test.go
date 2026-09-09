// SPDX-License-Identifier: GPL-2.0-only

package tessellate

import (
	"testing"

	"oblikovati.org/math"
)

// TestASlitIsNoRimSegment: a boundary that walks a segment in BOTH directions — the artificial seam
// the piston head's merged wall carries up and back — bounds material on both sides, so neither
// direction may constrain a triangle. The two traversals are SEPARATE covering vertices at the same
// 3D point, which is why the count is taken on the weld and not on the index pair (#3518).
func TestASlitIsNoRimSegment(t *testing.T) {
	t.Parallel()
	b, chains := slitCover(t)
	got := b.directedRimSegments([]rimSegment{{0, 1, 0}, {1, 2, 0}, {3, 4, 0}, {4, 5, 0}}, chains)
	if len(got) != 0 {
		t.Errorf("directedRimSegments kept %d segment(s) of a boundary that is entirely slit; want 0 — "+
			"the two traversals are different covering vertices at the same 3D points", len(got))
	}
}

// TestEachChainKeepsItsOwnSegments: two chains laid into one covering are told apart, so their
// material sides are voted separately — the fixture wall's window hole is wound the other way from
// its rims and a single per-face bit threw it away (#3518).
func TestEachChainKeepsItsOwnSegments(t *testing.T) {
	t.Parallel()
	b, chains := openChainCover(t)
	got := b.directedRimSegments([]rimSegment{{0, 1, 0}, {1, 2, 0}, {2, 3, 1}}, chains)
	if got[[2]int{0, 1}] != 0 || got[[2]int{1, 2}] != 0 {
		t.Errorf("chain 0's segments are filed under %d/%d; want 0", got[[2]int{0, 1}], got[[2]int{1, 2}])
	}
	if got[[2]int{2, 3}] != 1 {
		t.Errorf("chain 1's segment is filed under %d; want 1", got[[2]int{2, 3}])
	}
}

// slitCover holds one three-point run laid TWICE, forward then backward, at the same 3D points — the
// slit shape, and the chain that says so.
func slitCover(t *testing.T) (*chartCover, []chartChain) {
	t.Helper()
	b := newBareCover(t)
	p := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}
	order := []int{0, 1, 2, 2, 1, 0}
	run := make([]math.Point3, 0, len(order))
	for _, i := range order {
		b.add(p[i], float64(i), 0)
		run = append(run, p[i])
	}
	return b, []chartChain{{p3: run}}
}

// openChainCover holds four distinct 3D points in a row, each segment walked once, split into two
// chains that share the middle point.
func openChainCover(t *testing.T) (*chartCover, []chartChain) {
	t.Helper()
	b := newBareCover(t)
	run := make([]math.Point3, 0, 4)
	for i := range 4 {
		p := math.P3(float64(i), 0, 0)
		b.add(p, float64(i), 0)
		run = append(run, p)
	}
	return b, []chartChain{{p3: run[:3]}, {p3: run[2:]}}
}

// newBareCover is a covering with nothing but the metric and a normal, for the segment-bookkeeping rows.
func newBareCover(t *testing.T) *chartCover {
	t.Helper()
	b := &chartCover{}
	b.normalAt = func(float64, float64) math.Vector3 { return math.V3(0, 0, 1) }
	b.su, b.sv = 1, 1
	return b
}

// TestAChainWithNoDecisiveSegmentConstrainsNothing: the side is READ from the chart, so a chain the
// chart never bounded exactly once has no side and may not drop anything.
func TestAChainWithNoDecisiveSegmentConstrainsNothing(t *testing.T) {
	t.Parallel()
	s := rimSideFrom([]int{3, 0, 2}, []int{1, 0, 2})
	for _, c := range []struct {
		ci            int
		left, decided bool
	}{{0, true, true}, {1, false, false}, {2, false, false}} {
		if s.left[c.ci] != c.left || s.decided[c.ci] != c.decided {
			t.Errorf("chain %d: left=%v decided=%v; want %v/%v", c.ci, s.left[c.ci], s.decided[c.ci], c.left, c.decided)
		}
	}
}

// TestDirectedTriangleEdgesReadCounterClockwise pins the convention the whole rule rests on: a CCW
// triangle lies to the LEFT of each edge it names, so carrying (a→b) is the same statement as
// "material is on the left of a→b".
func TestDirectedTriangleEdgesReadCounterClockwise(t *testing.T) {
	t.Parallel()
	got := directedTriangleEdges([3]int{4, 5, 6})
	want := [3][2]int{{4, 5}, {5, 6}, {6, 4}}
	if got != want {
		t.Errorf("directedTriangleEdges = %v, want %v (the traversal order, closing back to the first)", got, want)
	}
}

// TestOnlyKeptTrianglesVote: the side is read off the chart's own verdict, so a triangle the chart
// dropped may not cast a vote.
func TestOnlyKeptTrianglesVote(t *testing.T) {
	t.Parallel()
	tris := [][3]int{{0, 1, 2}, {1, 0, 3}}
	got := keptDirectedEdgeUse(tris, []bool{true, false})
	if got[[2]int{0, 1}] != 1 || got[[2]int{1, 0}] != 0 {
		t.Errorf("edge use = %d forward / %d back; want 1/0 — the dropped triangle voted", got[[2]int{0, 1}], got[[2]int{1, 0}])
	}
}
