// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/topo"
)

// farVerticalEdgeFrom returns a vertical edge of the box whose X,Y is farthest from the
// already-filleted corner at (skipX,skipY), so the second fillet lands on an UNTOUCHED edge.
func farVerticalEdgeFrom(t *testing.T, b *topo.Body, skipX, skipY float64) *topo.Edge {
	t.Helper()
	var best *topo.Edge
	bestD := -1.0
	for _, e := range b.Edges() {
		a, c := e.StartVertex().Point(), e.EndVertex().Point()
		if a.X != c.X || a.Y != c.Y {
			continue
		}
		dx, dy := float64(a.X)-skipX, float64(a.Y)-skipY
		if d := dx*dx + dy*dy; d > bestD {
			bestD, best = d, e
		}
	}
	if best == nil {
		t.Fatal("no vertical edge")
	}
	return best
}

// TestSecondFilletOnUntouchedEdge reproduces #1494: fillet one vertical edge of a box, then fillet
// the diagonally-opposite vertical edge — an edge the first fillet never touched. The reference key
// captured from the filleted body MUST still resolve and produce a second cylinder face. Before the
// fix the assembler re-lineaged every edge to fillet:e#N by construction order, so the untouched
// edge's key was unstable and the second fillet either lost the key or hit a wrong (curved) edge,
// going Sick → no geometry change.
func TestSecondFilletOnUntouchedEdge(t *testing.T) {
	box := shellBox(4, 3, 5)
	corner := verticalEdgeKey(t, box)
	f1, err := ops.FilletEdges(box, [][]byte{corner}, 0.5)
	if err != nil {
		t.Fatalf("first fillet: %v", err)
	}
	if got := hasCylinderFaces(f1); got != 1 {
		t.Fatalf("first fillet cylinder faces = %d, want 1", got)
	}

	// Pick the far corner from the filleted body and capture its key exactly as the UI would.
	far := farVerticalEdgeFrom(t, f1, -2, -1.5) // away from the (0,0)-ish first corner
	key := far.ReferenceKey()

	f2, err := ops.FilletEdges(f1, [][]byte{key}, 0.5)
	if err != nil {
		t.Fatalf("second fillet on untouched edge: %v", err)
	}
	if got := hasCylinderFaces(f2); got != 2 {
		t.Fatalf("after second fillet cylinder faces = %d, want 2 (the second fillet was a no-op)", got)
	}
}
