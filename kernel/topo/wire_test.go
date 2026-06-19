// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestWireAccessorsAndPlanarity covers the Wire accessors (id/kind/body/lineage/refkey/uses/
// edges) and the planarity queries: a closed planar triangle is closed + planar with a plane
// frame; an open two-edge chain is not closed.
func TestWireAccessorsAndPlanarity(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("w", "body", 0)))
	mk := func(p math.Point3, i int) *Vertex { return bld.AddVertex(p, NewLineage(Tok("w", "vertex", i))) }
	seg := func(p, q *Vertex, i int) *Edge {
		return bld.AddEdge(geom.NewLineSegment(p.Point(), q.Point()), p, q, NewLineage(Tok("w", "edge", i)))
	}
	a, b, c := mk(math.P3(0, 0, 0), 0), mk(math.P3(2, 0, 0), 1), mk(math.P3(0, 2, 0), 2)
	ab, bc, ca := seg(a, b, 0), seg(b, c, 1), seg(c, a, 2)
	body := bld.Build()

	wire := body.AttachWire(NewLineage(Tok("w", "wire", 0)), []Use{Fwd(ab), Fwd(bc), Fwd(ca)})
	if wire.Kind() != KindWire {
		t.Errorf("Kind = %v, want KindWire", wire.Kind())
	}
	if wire.Body() != body {
		t.Error("Body() did not return the owning body")
	}
	if wire.ID() == 0 || len(wire.ReferenceKey()) == 0 {
		t.Error("wire should have a non-zero id and a reference key")
	}
	_ = wire.Lineage() // exercised; equality is covered elsewhere
	if len(wire.Edges()) != 3 || len(wire.Uses()) != 3 {
		t.Errorf("edges=%d uses=%d, want 3 each", len(wire.Edges()), len(wire.Uses()))
	}
	if !wire.IsClosed() {
		t.Error("triangle wire should be closed")
	}
	if !wire.IsPlanar() {
		t.Error("triangle wire should be planar")
	}
	if _, _, ok := wire.PlaneFrame(); !ok {
		t.Error("planar wire should yield a plane frame")
	}
	if len(body.Wires()) != 1 {
		t.Errorf("body.Wires() = %d, want 1", len(body.Wires()))
	}

	open := body.AttachWire(NewLineage(Tok("w", "wire", 1)), []Use{Fwd(ab), Fwd(bc)})
	if open.IsClosed() {
		t.Error("an open two-edge chain should not be closed")
	}
}
