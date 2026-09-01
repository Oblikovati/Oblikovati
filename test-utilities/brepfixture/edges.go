// SPDX-License-Identifier: GPL-2.0-only

package brepfixture

import (
	"testing"

	"oblikovati.org/kernel/topo"
)

// VerticalEdgeKey returns the reference key of a body's first axis-parallel (constant x,y) edge.
// Edge selection is by reference key, so a fillet, chamfer or draft test needs the key rather
// than the edge — and needs it to still resolve after the operation ran.
//
// Example: k := brepfixture.VerticalEdgeKey(t, box) // the edge a fillet rounds
// verticalEdgeKey returns a vertical edge (start/end share X,Y) of a box.
func VerticalEdgeKey(tb testing.TB, b *topo.Body) []byte {
	tb.Helper()
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			return e.ReferenceKey()
		}
	}
	tb.Fatal("no vertical edge")
	return nil
}

// VerticalEdgeKeys returns the reference keys of every axis-parallel edge, in body order — the
// selection a tangent-chain or all-edges dress-up test starts from.
//
// Example: ks := brepfixture.VerticalEdgeKeys(t, box)
// verticalEdgeKeys returns every vertical edge (start/end share X,Y) of a box.
func VerticalEdgeKeys(tb testing.TB, b *topo.Body) [][]byte {
	tb.Helper()
	var keys [][]byte
	for _, e := range b.Edges() {
		if a, c := e.StartVertex().Point(), e.EndVertex().Point(); a.X == c.X && a.Y == c.Y {
			keys = append(keys, e.ReferenceKey())
		}
	}
	if len(keys) != 4 {
		tb.Fatalf("expected 4 vertical box edges, found %d", len(keys))
	}
	return keys
}
