// SPDX-License-Identifier: GPL-2.0-only

package renderer

import (
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/ops"
	"oblikovati/kernel/topo"
	"oblikovati/math"
	"oblikovati/scene"
)

// box builds an axis-aligned box solid [off, off+size]³-ish via a tetra-free prism:
// a unit square extruded, translated by off. Reusing a simple closed prism keeps the
// renderer test independent of the feature layer.
func box(side float64, off math.Vector3) *topo.Body {
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("r", "body", 0)))
	p := func(x, y, z float64) math.Point3 { return math.P3(x, y, z).TranslateBy(off) }
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok("r", role, i)) }
	// bottom (z=0) and top (z=side) squares
	b := [4]*topo.Vertex{
		bld.AddVertex(p(0, 0, 0), lin("v", 0)), bld.AddVertex(p(side, 0, 0), lin("v", 1)),
		bld.AddVertex(p(side, side, 0), lin("v", 2)), bld.AddVertex(p(0, side, 0), lin("v", 3)),
	}
	tp := [4]*topo.Vertex{
		bld.AddVertex(p(0, 0, side), lin("v", 4)), bld.AddVertex(p(side, 0, side), lin("v", 5)),
		bld.AddVertex(p(side, side, side), lin("v", 6)), bld.AddVertex(p(0, side, side), lin("v", 7)),
	}
	seg := func(a, c *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(a.Point(), c.Point()), a, c, lin("e", i))
	}
	be := [4]*topo.Edge{seg(b[0], b[1], 0), seg(b[1], b[2], 1), seg(b[2], b[3], 2), seg(b[3], b[0], 3)}
	te := [4]*topo.Edge{seg(tp[0], tp[1], 4), seg(tp[1], tp[2], 5), seg(tp[2], tp[3], 6), seg(tp[3], tp[0], 7)}
	ve := [4]*topo.Edge{seg(b[0], tp[0], 8), seg(b[1], tp[1], 9), seg(b[2], tp[2], 10), seg(b[3], tp[3], 11)}
	pl := func(o, n math.Vector3) geom.Surface { s, _ := geom.NewPlane(o.AsPoint().TranslateBy(off), n); return s }
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, 0, -1)), lin("f", 0), topo.OuterLoop(topo.Rev(be[3]), topo.Rev(be[2]), topo.Rev(be[1]), topo.Rev(be[0])))
	bld.AddFace(pl(math.V3(0, 0, side), math.V3(0, 0, 1)), lin("f", 1), topo.OuterLoop(topo.Fwd(te[0]), topo.Fwd(te[1]), topo.Fwd(te[2]), topo.Fwd(te[3])))
	up := math.V3(0, 0, 1)
	for i := 0; i < 4; i++ {
		j := (i + 1) % 4
		normal := b[i].Point().VectorTo(b[j].Point()).Cross(up) // outward = edge × up
		sp, _ := geom.NewPlane(b[i].Point(), normal)
		bld.AddFace(sp, lin("side", i),
			topo.OuterLoop(topo.Fwd(be[i]), topo.Fwd(ve[j]), topo.Rev(te[i]), topo.Rev(ve[i])))
	}
	return bld.Build()
}

func frontCamera() scene.Camera {
	c := scene.NewCamera(400, 400)
	c.Eye = math.P3(5, 5, 30)
	c.Target = math.P3(5, 5, 0)
	return c
}

func TestBuildDrawListEmitsSurfacesAndWireframe(t *testing.T) {
	list := BuildDrawList([]*topo.Body{box(2, math.V3(0, 0, 0))}, frontCamera(), ops.DefaultQuality(), nil)
	if len(list.Items) != 2 {
		t.Fatalf("draw items = %d, want 2 (surface + wireframe)", len(list.Items))
	}
	// A box has 6 planar faces → 12 triangles, and 12 edges → 12 line segments.
	if list.Triangles() != 12 {
		t.Errorf("triangles = %d, want 12", list.Triangles())
	}
	if list.Lines() != 12 {
		t.Errorf("lines = %d, want 12", list.Lines())
	}
	// The surface item carries normals and the body's object id.
	surf := list.Items[0]
	if surf.Primitive != Triangles || len(surf.Normals) != len(surf.Positions) {
		t.Error("surface item missing normals")
	}
}

func TestVisualStyleSelectsItems(t *testing.T) {
	b := box(2, math.V3(0, 0, 0))
	cam := frontCamera()
	shaded := BuildDrawListStyled([]*topo.Body{b}, cam, ops.DefaultQuality(), nil, Shaded)
	if shaded.Triangles() != 12 || shaded.Lines() != 0 {
		t.Errorf("Shaded = %d tris / %d lines, want 12 / 0", shaded.Triangles(), shaded.Lines())
	}
	withEdges := BuildDrawListStyled([]*topo.Body{b}, cam, ops.DefaultQuality(), nil, ShadedWithEdges)
	if withEdges.Triangles() != 12 || withEdges.Lines() != 12 {
		t.Errorf("ShadedWithEdges = %d tris / %d lines, want 12 / 12", withEdges.Triangles(), withEdges.Lines())
	}
	wire := BuildDrawListStyled([]*topo.Body{b}, cam, ops.DefaultQuality(), nil, Wireframe)
	if wire.Triangles() != 0 || wire.Lines() != 12 {
		t.Errorf("Wireframe = %d tris / %d lines, want 0 / 12", wire.Triangles(), wire.Lines())
	}
}

func TestObjectIDsTagEachBody(t *testing.T) {
	a, b := box(1, math.V3(0, 0, 0)), box(1, math.V3(3, 0, 0))
	list := BuildDrawList([]*topo.Body{a, b}, frontCamera(), ops.DefaultQuality(), nil)
	ids := map[uint64]bool{}
	for _, it := range list.Items {
		ids[it.ObjectID] = true
	}
	if !ids[a.ID()] || !ids[b.ID()] || len(ids) != 2 {
		t.Errorf("object ids = %v, want both body ids distinct", ids)
	}
}

// TestTranslationInvariance is a metamorphic oracle (ADR-0014): translating the
// scene must not change how much is drawn.
func TestTranslationInvariance(t *testing.T) {
	atOrigin := BuildDrawList([]*topo.Body{box(2, math.V3(0, 0, 0))}, frontCamera(), ops.DefaultQuality(), nil)
	// Move both the body and the camera by the same offset → identical draw list size.
	off := math.V3(7, -3, 0)
	cam := frontCamera()
	cam.Eye = cam.Eye.TranslateBy(off)
	cam.Target = cam.Target.TranslateBy(off)
	moved := BuildDrawList([]*topo.Body{box(2, off)}, cam, ops.DefaultQuality(), nil)
	if atOrigin.Triangles() != moved.Triangles() || atOrigin.Lines() != moved.Lines() {
		t.Errorf("translation changed draw counts: %d/%d vs %d/%d",
			atOrigin.Triangles(), atOrigin.Lines(), moved.Triangles(), moved.Lines())
	}
}

func TestBodyBehindCameraIsCulled(t *testing.T) {
	cam := frontCamera()                 // looking from z=30 toward z=0 (forward −Z)
	behind := box(2, math.V3(0, 0, 100)) // box well behind the eye
	list := BuildDrawList([]*topo.Body{behind}, cam, ops.DefaultQuality(), nil)
	if len(list.Items) != 0 {
		t.Errorf("body behind the camera not culled: %d items", len(list.Items))
	}
}

func TestNullBackendRecordsFrames(t *testing.T) {
	var be Backend = &NullBackend{}
	list := BuildDrawList([]*topo.Body{box(1, math.V3(0, 0, 0))}, frontCamera(), ops.DefaultQuality(), nil)
	be.Render(list)
	be.Render(list)
	null := be.(*NullBackend)
	if null.FrameCount() != 2 || null.LastFrame().Triangles() != 12 {
		t.Errorf("null backend recorded %d frames, last triangles=%d", null.FrameCount(), null.LastFrame().Triangles())
	}
	null.Reset()
	if null.FrameCount() != 0 {
		t.Error("Reset did not clear frames")
	}
}

func TestDrawItemPrimitiveCounts(t *testing.T) {
	tri := DrawItem{Primitive: Triangles, Indices: []int{0, 1, 2, 0, 2, 3}}
	line := DrawItem{Primitive: Lines, Indices: []int{0, 1, 1, 2}}
	if tri.TriangleCount() != 2 || tri.LineCount() != 0 {
		t.Error("triangle item counts wrong")
	}
	if line.LineCount() != 2 || line.TriangleCount() != 0 {
		t.Error("line item counts wrong")
	}
}

// A SurfaceLookup must drive the triangle item's PBR fields (albedo + metallic/roughness);
// the wireframe item keeps the edge color.
func TestBuildDrawListAppliesSurfaceLookup(t *testing.T) {
	want := Surface{Albedo: [4]float32{0.2, 0.4, 0.8, 1}, Metallic: 0.9, Roughness: 0.3, Opacity: 1}
	list := BuildDrawList([]*topo.Body{box(2, math.V3(0, 0, 0))}, frontCamera(), ops.DefaultQuality(),
		func(*topo.Body) Surface { return want })
	var tri *DrawItem
	for i := range list.Items {
		if list.Items[i].Primitive == Triangles {
			tri = &list.Items[i]
		}
	}
	if tri == nil {
		t.Fatal("no triangle item produced")
	}
	if tri.Color != want.Albedo || tri.Metallic != want.Metallic || tri.Roughness != want.Roughness {
		t.Errorf("surface not applied: color=%v metallic=%v roughness=%v", tri.Color, tri.Metallic, tri.Roughness)
	}
}
