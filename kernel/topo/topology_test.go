// SPDX-License-Identifier: GPL-2.0-only

package topo

import (
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/math"
)

// buildTetra assembles a tetrahedron A(0,0,0) B(1,0,0) C(0,1,0) D(0,0,1) with
// lineage on every entity, rebuilt identically on each call (to test rebinding).
func buildTetra() *Body {
	feat := "extrude1"
	bld := NewBuilder(true, NewLineage(Tok(feat, "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), NewLineage(Tok(feat, "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok(feat, "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), NewLineage(Tok(feat, "vertex", 2)))
	d := bld.AddVertex(math.P3(0, 0, 1), NewLineage(Tok(feat, "vertex", 3)))

	seg := func(p, q *Vertex) geom.LineSegment { return geom.NewLineSegment(p.Point(), q.Point()) }
	edge := func(p, q *Vertex, i int) *Edge { return bld.AddEdge(seg(p, q), p, q, NewLineage(Tok(feat, "edge", i))) }
	ab, ac, ad := edge(a, b, 0), edge(a, c, 1), edge(a, d, 2)
	bc, bd, cd := edge(b, c, 3), edge(b, d, 4), edge(c, d, 5)

	plane := func(origin, normal math.Vector3) geom.Surface {
		p, _ := geom.NewPlane(origin.AsPoint(), normal)
		return p
	}
	faceLin := func(i int) Lineage { return NewLineage(Tok(feat, "face", i)) }
	bld.AddFace(plane(math.V3(0, 0, 0), math.V3(0, 0, 1)), faceLin(0), OuterLoop(Fwd(ab), Fwd(bc), Rev(ac)))
	bld.AddFace(plane(math.V3(0, 0, 0), math.V3(0, 1, 0)), faceLin(1), OuterLoop(Fwd(ab), Fwd(bd), Rev(ad)))
	bld.AddFace(plane(math.V3(0, 0, 0), math.V3(1, 0, 0)), faceLin(2), OuterLoop(Fwd(ac), Fwd(cd), Rev(ad)))
	bld.AddFace(plane(math.V3(1, 1, 1), math.V3(1, 1, 1)), faceLin(3), OuterLoop(Fwd(bc), Fwd(cd), Rev(bd)))
	return bld.Build()
}

func TestTetraTopologyCounts(t *testing.T) {
	body := buildTetra()
	if !body.IsSolid() {
		t.Error("tetra body should be solid")
	}
	if len(body.Faces()) != 4 || len(body.Edges()) != 6 || len(body.Vertices()) != 4 {
		t.Fatalf("F/E/V = %d/%d/%d, want 4/6/4", len(body.Faces()), len(body.Edges()), len(body.Vertices()))
	}
	if len(body.Shells()) != 1 || !body.Shells()[0].IsClosed() {
		t.Error("solid body should have one closed shell")
	}
}

func TestAdjacencyIsConsistentAndComplete(t *testing.T) {
	body := buildTetra()
	// Every edge of a tetrahedron is shared by exactly two faces.
	for _, e := range body.Edges() {
		if got := len(e.Faces()); got != 2 {
			t.Errorf("edge %d bounds %d faces, want 2", e.ID(), got)
		}
		if len(e.Vertices()) != 2 {
			t.Error("edge should have two vertices")
		}
	}
	// Every face is a triangle: 3 edges, 3 vertices.
	for _, f := range body.Faces() {
		if len(f.Edges()) != 3 || len(f.Vertices()) != 3 {
			t.Errorf("face %d: %d edges / %d vertices, want 3/3", f.ID(), len(f.Edges()), len(f.Vertices()))
		}
		if len(f.Loops()) != 1 || !f.Loops()[0].IsOuter() {
			t.Error("each tetra face has one outer loop")
		}
	}
	// Each vertex meets exactly three edges; each edge-use points back to its loop's face.
	for _, v := range body.Vertices() {
		if got := len(v.Edges()); got != 3 {
			t.Errorf("vertex meets %d edges, want 3", got)
		}
	}
	for _, l := range body.Faces()[0].Loops() {
		for _, u := range l.EdgeUses() {
			if u.Loop() != l || u.Edge() == nil {
				t.Error("edge-use back-pointers inconsistent")
			}
		}
	}
}

func TestFaceAndEdgeGeometry(t *testing.T) {
	body := buildTetra()
	if _, ok := body.Faces()[0].Geometry().(geom.Plane); !ok {
		t.Errorf("planar face geometry = %T, want geom.Plane", body.Faces()[0].Geometry())
	}
	if _, ok := body.Edges()[0].Geometry().(geom.LineSegment); !ok {
		t.Errorf("straight edge geometry = %T, want geom.LineSegment", body.Edges()[0].Geometry())
	}
}

func TestCircularEdgeReturnsCircle(t *testing.T) {
	bld := NewBuilder(false, NewLineage(Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(1, 0, 0), NewLineage(Tok("f", "vertex", 0)))
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
	e := bld.AddEdge(circle, v, v, NewLineage(Tok("f", "edge", 0))) // closed circular edge
	if _, ok := e.Geometry().(geom.Circle); !ok {
		t.Errorf("circular edge geometry = %T, want geom.Circle", e.Geometry())
	}
}

func TestReferenceKeyRebindsPostRecompute(t *testing.T) {
	original := buildTetra()
	face := original.Faces()[0]
	key := face.ReferenceKey()
	if len(key) == 0 {
		t.Fatal("face produced an empty reference key")
	}

	// "Recompute": destroy and rebuild the B-rep. The rebuilt face is a different
	// object but carries the same lineage, so the key rebinds to it.
	rebuilt := buildTetra()
	bound, ok := rebuilt.FindFaceByKey(key)
	if !ok {
		t.Fatal("reference key did not rebind after recompute")
	}
	if bound == face {
		t.Error("rebound to the OLD face object; expected the rebuilt one")
	}
	if string(bound.ReferenceKey()) != string(key) {
		t.Error("rebound face has a different key")
	}

	// An edge key rebinds too; a bogus key does not.
	ekey := original.Edges()[0].ReferenceKey()
	if _, ok := rebuilt.FindEdgeByKey(ekey); !ok {
		t.Error("edge key did not rebind")
	}
	bogus := referenceKey(KindFace, NewLineage(Tok("ghost", "face", 99)))
	if _, ok := rebuilt.FindFaceByKey(bogus); ok {
		t.Error("a key to nonexistent topology should not bind")
	}
}

func TestRangeBoxes(t *testing.T) {
	body := buildTetra()
	box := body.RangeBox()
	if !box.Contains(math.P3(0, 0, 0)) || !box.Contains(math.P3(1, 0, 0)) || !box.Contains(math.P3(0, 0, 1)) {
		t.Error("body range box does not contain its vertices")
	}
	if v := box.Diagonal(); v.X < 1 || v.Y < 1 || v.Z < 1 {
		t.Errorf("range box diagonal too small: %v", v)
	}
	if body.Faces()[0].RangeBox().IsEmpty() || body.Edges()[0].RangeBox().IsEmpty() {
		t.Error("face/edge range boxes should not be empty")
	}
}

func TestSurfaceBodiesCollection(t *testing.T) {
	c := NewSurfaceBodies()
	b1 := c.Add(buildTetra())
	b2 := c.Add(buildTetra())
	if c.Count() != 2 || c.Item(0) != b1 || len(c.All()) != 2 {
		t.Fatal("SurfaceBodies tracking wrong")
	}
	if got, ok := c.ByID(b2.ID()); !ok || got != b2 {
		t.Error("ByID failed")
	}
	if !c.Remove(b1) || c.Count() != 1 || c.Remove(b1) {
		t.Error("Remove behavior wrong")
	}
}

func TestEntityKindStrings(t *testing.T) {
	kinds := map[EntityKind]string{
		KindVertex: "vertex", KindEdge: "edge", KindFace: "face",
		KindLoop: "loop", KindShell: "shell", KindBody: "body", EntityKind(9): "unknown",
	}
	for k, want := range kinds {
		if k.String() != want {
			t.Errorf("EntityKind(%d).String() = %q, want %q", k, k.String(), want)
		}
	}
}
