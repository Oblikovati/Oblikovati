// SPDX-License-Identifier: GPL-2.0-only

package ops

import "testing"

func TestEarcutSegmentPredicates(t *testing.T) {
	tc := &triContext{
		x: []float64{0, 2, 2, 0, 1, 1, 3},
		y: []float64{0, 0, 2, 2, -1, 3, 0},
	}
	n := make([]*triNode, len(tc.x))
	for i := range n {
		n[i] = &triNode{i: i}
	}
	if !tc.intersects(n[0], n[2], n[4], n[5]) {
		t.Fatal("expected diagonal segment to intersect vertical segment")
	}
	if !tc.intersects(n[0], n[1], n[1], n[6]) {
		t.Fatal("expected collinear endpoint segment to intersect")
	}
	if tc.intersects(n[0], n[1], n[2], n[3]) {
		t.Fatal("parallel separated segments should not intersect")
	}
	if sign(-1) != -1 || sign(0) != 0 || sign(1) != 1 {
		t.Fatal("sign returned unexpected values")
	}
}

func TestEarcutPolygonPredicates(t *testing.T) {
	tc, nodes := earcutTestRing([][2]float64{{0, 0}, {4, 0}, {4, 4}, {2, 2}, {0, 4}})
	if !tc.middleInside(nodes[0], nodes[2]) {
		t.Fatal("diagonal midpoint should be inside polygon")
	}
	if !tc.locallyInside(nodes[0], nodes[2]) {
		t.Fatal("diagonal should be locally inside at convex vertex")
	}
	if !tc.sectorContains(nodes[0], nodes[2]) {
		t.Fatal("reflex-sector tie-break should accept this vertex")
	}
	if tc.isValidDiagonal(nodes[0], nodes[2]) {
		t.Fatal("expected full diagonal predicate to reject this candidate")
	}
	if !tc.intersectsPolygon(nodes[1], nodes[4]) {
		t.Fatal("expected candidate diagonal to cross a polygon edge")
	}
}

func TestEarcutSplitFallbackDirect(t *testing.T) {
	tc, nodes := earcutTestRing([][2]float64{{0, 0}, {3, 0}, {3, 1}, {1, 1}, {1, 3}, {0, 3}})
	var tris [][3]int
	tc.splitEarcut(nodes[0], &tris)
	if len(tris) == 0 {
		t.Fatal("splitEarcut did not emit triangles for splittable polygon")
	}
}

func earcutTestRing(points [][2]float64) (*triContext, []*triNode) {
	tc := &triContext{x: make([]float64, len(points)), y: make([]float64, len(points))}
	var last *triNode
	nodes := make([]*triNode, len(points))
	for i, p := range points {
		tc.x[i], tc.y[i] = p[0], p[1]
		last = tc.newNode(i, last)
		nodes[i] = last
	}
	return tc, nodes
}
