// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestReterminateBothEnds covers the n==3 corner-host flank grow/recede: a single straight flank whose
// BOTH ends move to new feet on its own supporting line, and the off-line decline (do-no-harm floor).
func TestReterminateBothEnds(t *testing.T) {
	t.Parallel()
	flank := endSeg{from: math.P3(0, 0, 0), to: math.P3(10, 0, 0)} // along +x
	got, ok := reterminateBothEnds(flank, math.P3(2, 0, 0), math.P3(8, 0, 0), 1e-9)
	if !ok {
		t.Fatalf("reterminateBothEnds declined an on-line pair")
	}
	if got.from.DistanceTo(math.P3(2, 0, 0)) > 1e-9 || got.to.DistanceTo(math.P3(8, 0, 0)) > 1e-9 {
		t.Fatalf("reterminated flank = %v->%v, want (2,0,0)->(8,0,0)", got.from, got.to)
	}
	if _, ok := reterminateBothEnds(flank, math.P3(2, 5, 0), math.P3(8, 0, 0), 1e-9); ok {
		t.Fatalf("reterminateBothEnds accepted an OFF-line foot (2,5,0) — must decline")
	}
}

// TestGrowCapArc covers the concave far-cap grow: the far-vertex corner of a square loop is replaced by a
// cross-section chord and the two flanking edges are re-terminated onto its feet, growing the ring by one
// segment into a closed loop. An off-flank arc foot declines (do-no-harm).
func TestGrowCapArc(t *testing.T) {
	t.Parallel()
	// square (0,0)-(10,0)-(10,10)-(0,10); the corner at (0,0) is the "far vertex" the cap grows around.
	segs := []endSeg{
		{from: math.P3(0, 0, 0), to: math.P3(10, 0, 0)},
		{from: math.P3(10, 0, 0), to: math.P3(10, 10, 0)},
		{from: math.P3(10, 10, 0), to: math.P3(0, 10, 0)},
		{from: math.P3(0, 10, 0), to: math.P3(0, 0, 0)},
	}
	bite := endSeg{from: math.P3(0, 2, 0), to: math.P3(2, 0, 0)} // feet on x=0 (prev) and y=0 (next)
	got, ok := growCapArc(segs, bite, math.P3(0, 0, 0), 1e-9)
	if !ok {
		t.Fatalf("growCapArc declined a valid cap grow")
	}
	if len(got) != 5 {
		t.Fatalf("growCapArc produced %d segs, want 5 (the corner replaced by the bite)", len(got))
	}
	assertClosedRing(t, got)
	if _, ok := growCapArc(segs, endSeg{from: math.P3(0, 2, 0), to: math.P3(2, 2, 0)}, math.P3(0, 0, 0), 1e-9); ok {
		t.Fatalf("growCapArc accepted an arc foot (2,2,0) off the flanking edge — must decline")
	}
}

// assertClosedRing checks each seg's to coincides with the next seg's from (a continuous closed loop).
func assertClosedRing(t *testing.T, segs []endSeg) {
	t.Helper()
	for i := range segs {
		next := segs[(i+1)%len(segs)]
		if stdmath.Abs(float64(segs[i].to.DistanceTo(next.from))) > 1e-9 {
			t.Fatalf("ring broken at seg %d: %v -> next.from %v", i, segs[i].to, next.from)
		}
	}
}
