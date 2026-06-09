// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func TestEntityAccessors(t *testing.T) {
	body := buildTetra()
	if body.ID() == 0 || body.Kind() != KindBody || len(body.Lineage().Tokens()) != 1 || len(body.ReferenceKey()) == 0 {
		t.Error("body accessors wrong")
	}
	if body.Lineage().String() == "" {
		t.Error("lineage string empty")
	}

	v := body.Vertices()[0]
	if v.ID() == 0 || v.Kind() != KindVertex || len(v.Lineage().Tokens()) != 1 || v.RangeBox().IsEmpty() {
		t.Error("vertex accessors wrong")
	}

	e := body.Edges()[0]
	if e.ID() == 0 || e.Kind() != KindEdge || e.StartVertex() == nil || e.EndVertex() == nil {
		t.Error("edge accessors wrong")
	}

	f := body.Faces()[0]
	l := f.Loops()[0]
	if l.ID() == 0 || l.Kind() != KindLoop || l.Face() != f {
		t.Error("loop accessors wrong")
	}
	// The Rev(ac) use in face 0 is a reversed edge use.
	reversedSeen := false
	for _, u := range l.EdgeUses() {
		if u.Reversed() {
			reversedSeen = true
		}
	}
	if !reversedSeen {
		t.Error("expected a reversed edge use in the loop")
	}

	sh := body.Shells()[0]
	if sh.ID() == 0 || sh.Kind() != KindShell || len(sh.Faces()) != 4 || sh.Body() != body {
		t.Error("shell accessors wrong")
	}
}

func TestInnerLoopFace(t *testing.T) {
	// A planar washer-like face: an outer square loop and an inner hole loop.
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	mk := func(p math.Point3, i int) *Vertex { return bld.AddVertex(p, NewLineage(Tok("f", "vertex", i))) }
	o0, o1 := mk(math.P3(0, 0, 0), 0), mk(math.P3(4, 0, 0), 1)
	o2, o3 := mk(math.P3(4, 4, 0), 2), mk(math.P3(0, 4, 0), 3)
	seg := func(a, b *Vertex, i int) *Edge {
		return bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, NewLineage(Tok("f", "edge", i)))
	}
	e0, e1, e2, e3 := seg(o0, o1, 0), seg(o1, o2, 1), seg(o2, o3, 2), seg(o3, o0, 3)

	hub := mk(math.P3(2, 2, 0), 4)
	circle, _ := geom.NewCircle(math.P3(2, 2, 0), math.V3(0, 0, 1), 1)
	hole := bld.AddEdge(circle, hub, hub, NewLineage(Tok("f", "edge", 4)))

	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	face := bld.AddFace(
		plane, NewLineage(Tok("f", "face", 0)),
		OuterLoop(Fwd(e0), Fwd(e1), Fwd(e2), Fwd(e3)),
		InnerLoop(Fwd(hole)),
	)
	if len(face.Loops()) != 2 {
		t.Fatalf("face has %d loops, want 2", len(face.Loops()))
	}
	if !face.Loops()[0].IsOuter() || face.Loops()[1].IsOuter() {
		t.Error("loop roles wrong: want outer then inner")
	}
}

// AddFace yields a face whose sense agrees with its surface; AddReversedFace flags the sense
// as opposite (a Difference cut wall), keeping every other property identical.
func TestReversedFaceSense(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	lin := NewLineage(Tok("f", "x", 0))
	a := bld.AddVertex(math.P3(0, 0, 0), lin)
	b := bld.AddVertex(math.P3(1, 0, 0), lin)
	c := bld.AddVertex(math.P3(0, 1, 0), lin)
	tri := func() LoopSpec {
		ab := bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, lin)
		bc := bld.AddEdge(geom.NewLineSegment(b.Point(), c.Point()), b, c, lin)
		ca := bld.AddEdge(geom.NewLineSegment(c.Point(), a.Point()), c, a, lin)
		return OuterLoop(Fwd(ab), Fwd(bc), Fwd(ca))
	}
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	if bld.AddFace(pl, lin, tri()).Reversed() {
		t.Error("AddFace produced a reversed face")
	}
	if !bld.AddReversedFace(pl, lin, tri()).Reversed() {
		t.Error("AddReversedFace did not set the reversed sense")
	}
}

func TestFindEdgeByBogusKey(t *testing.T) {
	body := buildTetra()
	bogus := referenceKey(KindEdge, NewLineage(Tok("ghost", "edge", 42)))
	if _, ok := body.FindEdgeByKey(bogus); ok {
		t.Error("bogus edge key should not bind")
	}
}
