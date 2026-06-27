// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// TestKeptComponentsSplitsDisjointBands: two cells sharing an edge are one band; a disjoint cell is its own
// band — the split that separates a rod's two stubs (Oblikovati#1476).
func TestKeptComponentsSplitsDisjointBands(t *testing.T) {
	cellAt := func(x float64) Face2D {
		return Face2D{Outer: []math.Point2{math.P2(x, 0), math.P2(x+1, 0), math.P2(x+1, 1), math.P2(x, 1)}}
	}
	left, mid, far := cellAt(0), cellAt(1), cellAt(10) // left & mid share the edge x=1; far is disjoint
	comps := keptComponents([]Face2D{left, mid, far}, false)
	if len(comps) != 2 {
		t.Fatalf("keptComponents = %d bands, want 2 (left+mid joined, far separate)", len(comps))
	}
	sizes := map[int]int{}
	for _, c := range comps {
		sizes[len(c)]++
	}
	if sizes[2] != 1 || sizes[1] != 1 {
		t.Errorf("band sizes = %v, want one band of 2 cells and one of 1", sizes)
	}
}

// TestIsTopRim: a level nearer vMax is the top rim (reversed by the convention), nearer vMin the bottom.
func TestIsTopRim(t *testing.T) {
	c := ruledUV{band: coneSideBand_{vMin: 0, vMax: 12}}
	if !c.isTopRim(11) {
		t.Error("mv=11 should be the top rim (nearer vMax=12)")
	}
	if c.isTopRim(1) {
		t.Error("mv=1 should be the bottom rim (nearer vMin=0)")
	}
}

// TestReverseCurvedLoop: reversing a loop flips each edge's direction and the edge order, so it walks the
// opposite way around the same boundary (Oblikovati#1476).
func TestReverseCurvedLoop(t *testing.T) {
	in := curvedLoop{edges: []loopEdge{{t0: 0, t1: 1}, {t0: 2, t1: 3}}}
	out := reverseCurvedLoop(in)
	if len(out.edges) != 2 || out.edges[0].t0 != 3 || out.edges[0].t1 != 2 || out.edges[1].t0 != 1 || out.edges[1].t1 != 0 {
		t.Errorf("reverseCurvedLoop = %+v, want order+direction flipped", out.edges)
	}
}
