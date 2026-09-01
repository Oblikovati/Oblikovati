// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestKeptComponentsSplitsDisjointBands: two cells sharing an edge are one band; a disjoint cell is its own
// band — the split that separates a rod's two stubs (Oblikovati#1476).
func TestKeptComponentsSplitsDisjointBands(t *testing.T) {
	t.Parallel()
	cellAt := func(x float64) Face2D {
		return Face2D{Outer: []math.Point2{math.P2(x, 0), math.P2(x+1, 0), math.P2(x+1, 1), math.P2(x, 1)}}
	}
	left, mid, far := cellAt(0), cellAt(1), cellAt(10) // left & mid share the edge x=1; far is disjoint
	comps := keptComponents([]Face2D{left, mid, far}, true, false)
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

// TestSourceRimSense: each loop ON a band rim wants the sense the SOURCE recorded for THAT rim, and a
// full-wrap IMPRINT circle between them is no rim at all — it keeps its arrangement winding (#3460; the
// retired isTopRim halved the band and swallowed that circle).
func TestSourceRimSense(t *testing.T) {
	t.Parallel()
	rim := func(mv float64) emittedLoop {
		return emittedLoop{mv: mv, face: []loopEdge{{curve: geom.Circle{}, t0: 0, t1: 1}}}
	}
	c := ruledUV{band: coneSideBand_{vMin: 0, vMax: 12, topRimReversed: true, botRimReversed: false}}
	if rev, isRim := c.sourceRimSense(rim(12)); !isRim || !rev {
		t.Errorf("vMax rim = (%v,%v), want (true,true) — the source's top-rim sense", rev, isRim)
	}
	if rev, isRim := c.sourceRimSense(rim(0)); !isRim || rev {
		t.Errorf("vMin rim = (%v,%v), want (false,true) — the source's bottom-rim sense", rev, isRim)
	}
	// Each rim reads its OWN recorded sense: an apex-at-top cone band reverses the vMin rim while leaving
	// the vMax one forward, which a "negate the other flag" rule could not express for a synthetic end.
	flipped := ruledUV{band: coneSideBand_{vMin: 0, vMax: 12, topRimReversed: false, botRimReversed: true}}
	if rev, isRim := flipped.sourceRimSense(rim(0)); !isRim || !rev {
		t.Errorf("apex-at-top vMin rim = (%v,%v), want (true,true)", rev, isRim)
	}
	if _, isRim := c.sourceRimSense(rim(11)); isRim {
		t.Error("a full-wrap circle at mv=11 is an imprint, not the vMax rim")
	}
}

// TestChainReversed: an emitted loop is reversed exactly when its first edge runs t1<t0.
func TestChainReversed(t *testing.T) {
	t.Parallel()
	if chainReversed(nil) {
		t.Error("an empty chain is not reversed")
	}
	if !chainReversed([]loopEdge{{t0: 1, t1: 0}}) {
		t.Error("t0=1,t1=0 is a reversed chain")
	}
	if chainReversed([]loopEdge{{t0: 0, t1: 1}}) {
		t.Error("t0=0,t1=1 is a forward chain")
	}
}

// TestReverseCurvedLoop: reversing a loop flips each edge's direction and the edge order, so it walks the
// opposite way around the same boundary (Oblikovati#1476).
func TestReverseCurvedLoop(t *testing.T) {
	t.Parallel()
	in := curvedLoop{edges: []loopEdge{{t0: 0, t1: 1}, {t0: 2, t1: 3}}}
	out := reverseCurvedLoop(in)
	if len(out.edges) != 2 || out.edges[0].t0 != 3 || out.edges[0].t1 != 2 || out.edges[1].t0 != 1 || out.edges[1].t1 != 0 {
		t.Errorf("reverseCurvedLoop = %+v, want order+direction flipped", out.edges)
	}
}
