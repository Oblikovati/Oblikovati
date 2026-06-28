// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// TestEdgesByKeyDetectsCollision pins ADR-0043's resolution-guard primitive: EdgesByKey returns
// EVERY edge sharing a reference key, so a caller can tell a clean 1:1 bind from a topological-
// naming collision (two distinct edges minted the same lineage) instead of silently taking the
// first. A correct body binds 1; a colliding body binds >1.
func TestEdgesByKeyDetectsCollision(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), NewLineage(Tok("f", "vertex", 2)))

	dup := NewLineage(Tok("f", "edge", 0)) // ab and bc deliberately share one lineage ⇒ one key
	ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, dup)
	bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, dup)
	ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, NewLineage(Tok("f", "edge", 1)))
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(pl, NewLineage(Tok("f", "face", 0)), OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))
	body := bld.Build()

	if got := body.EdgesByKey(ca.ReferenceKey()); len(got) != 1 {
		t.Errorf("unique key bound %d edges, want 1", len(got))
	}
	if got := body.EdgesByKey(ab.ReferenceKey()); len(got) != 2 {
		t.Errorf("colliding key bound %d edges, want 2 (ab and bc share a lineage)", len(got))
	}
}

// TestFacesByKeyDetectsCollision is the face counterpart: FacesByKey returns every face sharing a
// reference key, so a collision (two faces minted the same lineage) is detectable (ADR-0043).
func TestFacesByKeyDetectsCollision(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	mk := func(x, y float64, i int) *Vertex {
		return bld.AddVertex(math.P3(x, y, 0), NewLineage(Tok("f", "vertex", i)))
	}
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	tri := func(a, b, c *Vertex, e int, lin Lineage) *Face {
		ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, NewLineage(Tok("f", "edge", e)))
		bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, NewLineage(Tok("f", "edge", e+1)))
		ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, NewLineage(Tok("f", "edge", e+2)))
		return bld.AddFace(pl, lin, OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca)))
	}
	dup := NewLineage(Tok("f", "face", 0)) // two faces deliberately share one lineage ⇒ one key
	f1 := tri(mk(0, 0, 0), mk(1, 0, 1), mk(0, 1, 2), 0, dup)
	tri(mk(3, 0, 3), mk(4, 0, 4), mk(3, 1, 5), 3, dup)
	uniq := tri(mk(6, 0, 6), mk(7, 0, 7), mk(6, 1, 8), 6, NewLineage(Tok("f", "face", 1)))
	body := bld.Build()

	if got := body.FacesByKey(uniq.ReferenceKey()); len(got) != 1 {
		t.Errorf("unique face key bound %d faces, want 1", len(got))
	}
	if got := body.FacesByKey(f1.ReferenceKey()); len(got) != 2 {
		t.Errorf("colliding face key bound %d faces, want 2", len(got))
	}
}
