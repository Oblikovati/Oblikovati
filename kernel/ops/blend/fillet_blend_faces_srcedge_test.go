// SPDX-License-Identifier: GPL-2.0-only

package blend

import (
	"testing"

	"oblikovati.org/math"
)

// Regression test for a wave-G bug (fillet_blend_faces.go's reversedEndSeg): reversing an
// endSeg dropped its srcEdge identity, which farPathSegs' "other way" branch runs on every
// kept segment when the far path is found by walking a ring backward. A bigon host face whose
// only two boundary edges are the picked edge and a straight closing chord (simple/G5, G9 —
// both ends of an open runout landing on the SAME planar cap) needs that identity to survive
// the reversal: without it, the retrim's copy of the shared chord collapses into the same
// generic (op-generated) class as an unrelated new rail sharing its endpoints, and the
// assembler's edge catalog silently fuses two DISTINCT edges into one — a non-manifold
// multi-face edge (proven live: G5/G9 built "valid=false ... non-manifold edge used by 4
// faces" before this fix, "valid=true ... solid=true" after). See fillet_blend_faces.go's
// reversedEndSeg comment for the full mechanism.

// TestReversedEndSegPreservesSrcEdge is the direct unit proof: reversing a segment with a
// nonzero srcEdge must return that SAME srcEdge, not the zero (op-generated) default.
func TestReversedEndSegPreservesSrcEdge(t *testing.T) {
	t.Parallel()
	s := endSeg{from: math.P3(0, 0, 0), to: math.P3(10, 0, 0), srcEdge: 42}
	r := reversedEndSeg(s)
	if r.srcEdge != 42 {
		t.Fatalf("reversedEndSeg dropped srcEdge: got %d, want 42 (from=%v to=%v)", r.srcEdge, r.from, r.to)
	}
	if !r.from.IsEqualTo(s.to, 1e-12) || !r.to.IsEqualTo(s.from, 1e-12) {
		t.Fatalf("reversedEndSeg did not swap from/to: got from=%v to=%v", r.from, r.to)
	}
}

// TestReversedEndSegsPreservesSrcEdgeThroughChain is the chain-level proof farPathSegs actually
// depends on: reverseEndSegs (the "other way" branch every double-bite retrim can take) must
// carry EVERY segment's srcEdge through, in reversed order.
func TestReversedEndSegsPreservesSrcEdgeThroughChain(t *testing.T) {
	t.Parallel()
	chain := []endSeg{
		{from: math.P3(0, 0, 0), to: math.P3(1, 0, 0), srcEdge: 7},
		{from: math.P3(1, 0, 0), to: math.P3(2, 0, 0), srcEdge: 0}, // op-generated bite: stays 0
		{from: math.P3(2, 0, 0), to: math.P3(3, 0, 0), srcEdge: 9},
	}
	rev := reverseEndSegs(chain)
	want := []uint64{9, 0, 7} // reversed order, each segment's OWN identity preserved
	if len(rev) != len(want) {
		t.Fatalf("reverseEndSegs changed segment count: got %d, want %d", len(rev), len(want))
	}
	for i, s := range rev {
		if s.srcEdge != want[i] {
			t.Errorf("reverseEndSegs[%d].srcEdge = %d, want %d", i, s.srcEdge, want[i])
		}
	}
}
