// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/math"
)

// The pointWelder.addID id/coordinate reconciliation (corner-blend-weld Piece 2, delete_face.go): an
// ided point ADOPTS a cell claimed only anonymously (id 0) — so a pass-through face welds to a rebuilt
// far-runout neighbour when a bore fillet spreads onto the base solid's own faces (N1) — while two
// DISTINCT non-zero ids at one coordinate stay a pinch (#1600). These pin that safety-critical branch
// directly (it had no unit test), since a regression there silently re-opens a weld or collapses a pinch.

const welderGrid = 1.0

// coincident is the single test coordinate every case welds at (all points quantize to one cell).
var coincident = math.P3(5, 5, 5)

// TestAddIDAdoptsAnonymousCellOrderIndependent pins property (a): an ided point and an anonymous (id 0)
// point at one coordinate weld to ONE vertex REGARDLESS of order — add-then-addID and addID-then-add
// both yield a single shared index. This is the N1 fix (the ided pass-through corner adopts the
// anonymous rebuilt-neighbour corner, closing the shared edge).
func TestAddIDAdoptsAnonymousCellOrderIndependent(t *testing.T) {
	w1 := newPointWelder(welderGrid)
	a := w1.add(coincident)        // anonymous first
	b := w1.addID(coincident, 100) // ided adopts it
	if a != b || len(w1.points) != 1 {
		t.Fatalf("anon→ided: indices a=%d b=%d, points=%d; want one shared vertex", a, b, len(w1.points))
	}
	w2 := newPointWelder(welderGrid)
	c := w2.addID(coincident, 100) // ided first
	d := w2.add(coincident)        // anonymous adopts it
	if c != d || len(w2.points) != 1 {
		t.Fatalf("ided→anon: indices c=%d d=%d, points=%d; want one shared vertex", c, d, len(w2.points))
	}
}

// TestAddIDSecondDistinctIDIsPinchAfterAdoption pins property (b): once an ided point A has adopted an
// anonymous cell, a SECOND distinct id B (B≠A) at the SAME coordinate does NOT merge — it is a pinch and
// gets its own vertex. Adoption must not turn the cell into a merge magnet for every later id.
func TestAddIDSecondDistinctIDIsPinchAfterAdoption(t *testing.T) {
	w := newPointWelder(welderGrid)
	w.add(coincident)             // anonymous claim
	a := w.addID(coincident, 100) // A adopts it
	b := w.addID(coincident, 200) // B ≠ A — a pinch
	if a == b || len(w.points) != 2 {
		t.Fatalf("anon→adopt A→B: a=%d b=%d, points=%d; want two vertices (B is a pinch)", a, b, len(w.points))
	}
}

// TestAddIDTwoDistinctIDsNeverMerge pins property (c) — the #1600 invariant: two DISTINCT non-zero ids
// at one coordinate are two topological vertices (a boolean's kissing-tangency pinch) and must never
// weld into a non-manifold pinch, whichever arrives first.
func TestAddIDTwoDistinctIDsNeverMerge(t *testing.T) {
	w := newPointWelder(welderGrid)
	a := w.addID(coincident, 100)
	b := w.addID(coincident, 200)
	if a == b || len(w.points) != 2 {
		t.Fatalf("distinct ids: a=%d b=%d, points=%d; want two vertices (#1600 pinch preserved)", a, b, len(w.points))
	}
	// Same id resolves back to its own vertex (not the other pinch half).
	if again := w.addID(coincident, 100); again != a {
		t.Fatalf("re-add id 100 = %d, want %d (id must resolve to its first vertex)", again, a)
	}
}
