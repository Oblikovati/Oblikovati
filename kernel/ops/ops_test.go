// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// tetra builds a unit tetrahedron scaled by s and translated by off, with lineage.
func tetra(s float64, off math.Vector3) *topo.Body {
	feat := "f"
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok(feat, "body", 0)))
	p := func(x, y, z float64) math.Point3 { return math.P3(x*s, y*s, z*s).TranslateBy(off) }
	lin := func(role string, i int) topo.Lineage { return topo.NewLineage(topo.Tok(feat, role, i)) }
	a := bld.AddVertex(p(0, 0, 0), lin("vertex", 0))
	b := bld.AddVertex(p(1, 0, 0), lin("vertex", 1))
	c := bld.AddVertex(p(0, 1, 0), lin("vertex", 2))
	d := bld.AddVertex(p(0, 0, 1), lin("vertex", 3))
	seg := func(x, y *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(x.Point(), y.Point()), x, y, lin("edge", i))
	}
	ab, ac, ad := seg(a, b, 0), seg(a, c, 1), seg(a, d, 2)
	bc, bd, cd := seg(b, c, 3), seg(b, d, 4), seg(c, d, 5)
	pl := func(o, n math.Vector3) geom.Surface { s, _ := geom.NewPlane(o.AsPoint().TranslateBy(off), n); return s }
	// Consistently-outward loops: each shared edge is traversed in opposite
	// directions by its two faces (one Fwd, one Rev), as a valid manifold requires.
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, 0, -1)), lin("face", 0), topo.OuterLoop(topo.Fwd(ac), topo.Rev(bc), topo.Rev(ab)))
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(0, -1, 0)), lin("face", 1), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(bd), topo.Rev(ad)))
	bld.AddFace(pl(math.V3(0, 0, 0), math.V3(-1, 0, 0)), lin("face", 2), topo.OuterLoop(topo.Fwd(ad), topo.Rev(cd), topo.Rev(ac)))
	// The slant plane must actually contain b,c,d (the plane x+y+z=s): its origin is vertex b,
	// scaled with s. A fixed (1,1,1) origin puts the plane at x+y+z=3 — off the face for any s≠3,
	// which the analytic point classifier (unlike mesh tessellation) correctly rejects.
	bld.AddFace(pl(math.V3(1, 0, 0).Scale(s), math.V3(1, 1, 1)), lin("face", 3), topo.OuterLoop(topo.Fwd(bc), topo.Fwd(cd), topo.Rev(bd)))
	return bld.Build()
}

func TestPlanarFaceTessellationIsWatertight(t *testing.T) {
	t.Parallel()
	body := tetra(1, math.V3(0, 0, 0))
	mesh := TessellateFace(body.Faces()[0], DefaultQuality())
	// A triangle face → exactly one triangle over its 3 boundary vertices.
	if mesh.TriangleCount() != 1 || mesh.VertexCount() != 3 {
		t.Fatalf("triangle face → %d tris / %d verts, want 1/3", mesh.TriangleCount(), mesh.VertexCount())
	}
}

func TestEdgeTessellationHonorsChordTolerance(t *testing.T) {
	t.Parallel()
	circle, _ := geom.NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 5)
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(5, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	e := bld.AddEdge(circle, v, v, topo.NewLineage(topo.Tok("f", "edge", 0)))
	tol := 0.01
	poly := TessellateEdge(e, Quality{ChordTolerance: tol})
	if len(poly) < 4 {
		t.Fatalf("circle tessellated into only %d points", len(poly))
	}
	// Every segment midpoint must lie within tol of the true circle (radius 5).
	for i := 0; i+1 < len(poly); i++ {
		mid := poly[i].Midpoint(poly[i+1])
		dev := stdmath.Abs(mid.DistanceTo(math.P3(0, 0, 0)) - 5)
		if dev > tol {
			t.Errorf("chord deviation %v exceeds tolerance %v", dev, tol)
		}
	}
}

func TestCurvedFaceTessellationOnSphere(t *testing.T) {
	t.Parallel()
	sphere, _ := geom.NewSphere(math.P3(0, 0, 0), 2)
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	v := bld.AddVertex(math.P3(2, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	e := bld.AddEdge(geom.NewLineSegment(math.P3(2, 0, 0), math.P3(2, 0, 0)), v, v, topo.NewLineage(topo.Tok("f", "edge", 0)))
	f := bld.AddFace(sphere, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
	mesh := TessellateFace(f, Quality{ChordTolerance: 0.05})
	if mesh.TriangleCount() == 0 {
		t.Fatal("sphere face produced no triangles")
	}
	// Every sampled vertex lies on the sphere of radius 2.
	for _, p := range mesh.Positions {
		if stdmath.Abs(p.DistanceTo(math.P3(0, 0, 0))-2) > 1e-9 {
			t.Errorf("sphere sample off the surface: |p|=%v", p.DistanceTo(math.P3(0, 0, 0)))
		}
	}
}

func TestValidateManifoldSolid(t *testing.T) {
	t.Parallel()
	r := Validate(tetra(1, math.V3(0, 0, 0)))
	if !r.Valid || !r.Manifold || !r.Closed || !r.OrientationOK {
		t.Errorf("tetra validation = %+v, want fully valid", r)
	}
	if len(BoundaryEdges(tetra(1, math.V3(0, 0, 0)))) != 0 {
		t.Error("closed solid should have no boundary edges")
	}
}

func TestValidateOpenSurfaceReportsBoundary(t *testing.T) {
	t.Parallel()
	// A single triangular face is a surface body with three boundary edges.
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	a := bld.AddVertex(math.P3(0, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
	b := bld.AddVertex(math.P3(1, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 1)))
	c := bld.AddVertex(math.P3(0, 1, 0), topo.NewLineage(topo.Tok("f", "vertex", 2)))
	seg := func(x, y *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(x.Point(), y.Point()), x, y, topo.NewLineage(topo.Tok("f", "edge", i)))
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(plane, topo.NewLineage(topo.Tok("f", "face", 0)),
		topo.OuterLoop(topo.Fwd(seg(a, b, 0)), topo.Fwd(seg(b, c, 1)), topo.Rev(seg(a, c, 2))))
	body := bld.Build()

	r := Validate(body)
	if !r.Valid { // surface body is allowed to be open
		t.Errorf("open surface body should still be valid: %+v", r)
	}
	if r.Closed || len(BoundaryEdges(body)) != 3 {
		t.Errorf("expected 3 boundary edges on the open triangle, got %d", len(BoundaryEdges(body)))
	}
}

func TestPointInsideBody(t *testing.T) {
	t.Parallel()
	body := tetra(10, math.V3(0, 0, 0)) // big tetra: simplex x,y,z>=0, x+y+z<=10
	if !PointInsideBody(body, math.P3(2, 2, 2)) {
		t.Error("interior point reported outside")
	}
	if PointInsideBody(body, math.P3(20, 20, 20)) {
		t.Error("far exterior point reported inside")
	}
	if PointInsideBody(body, math.P3(-1, -1, -1)) {
		t.Error("point behind the body reported inside")
	}
}

func TestBooleanDisjoint(t *testing.T) {
	t.Parallel()
	a := tetra(1, math.V3(0, 0, 0))
	b := tetra(1, math.V3(10, 0, 0)) // far away → disjoint bounding boxes

	join, err := Boolean(Join, a, b)
	if err != nil {
		t.Fatalf("Join: %v", err)
	}
	if len(join.Faces()) != 8 || !Validate(join).Valid {
		t.Errorf("disjoint join → %d faces, valid=%v; want 8 faces / valid", len(join.Faces()), Validate(join).Valid)
	}

	inter, _ := Boolean(Intersect, tetra(1, math.V3(0, 0, 0)), tetra(1, math.V3(10, 0, 0)))
	if len(inter.Faces()) != 0 {
		t.Errorf("disjoint intersect → %d faces, want 0 (empty)", len(inter.Faces()))
	}

	cut, _ := Boolean(Cut, tetra(1, math.V3(0, 0, 0)), tetra(1, math.V3(10, 0, 0)))
	if len(cut.Faces()) != 4 {
		t.Errorf("disjoint cut → %d faces, want 4 (target unchanged)", len(cut.Faces()))
	}
}

func TestBooleanContainment(t *testing.T) {
	t.Parallel()
	big := tetra(10, math.V3(0, 0, 0))
	small := tetra(1, math.V3(2, 2, 2)) // strictly inside big

	inter, err := Boolean(Intersect, big, small)
	if err != nil {
		t.Fatalf("Intersect: %v", err)
	}
	if len(inter.Faces()) != 4 { // intersection is the inner body
		t.Errorf("contained intersect → %d faces, want 4 (the inner)", len(inter.Faces()))
	}
	// Cutting a fully-enclosed tool hollows the target into a cavity (a solid with a
	// void): the BSP CSG keeps the outer shell and the tool's inward-facing walls.
	cav, err := Boolean(Cut, tetra(10, math.V3(0, 0, 0)), tetra(1, math.V3(2, 2, 2)))
	if err != nil {
		t.Fatalf("cavity cut: %v", err)
	}
	if r := Validate(cav); !r.Valid {
		t.Errorf("cavity cut should be a valid (hollow) solid: %+v", r)
	}
}

func TestBooleanNewBodyAndStrings(t *testing.T) {
	t.Parallel()
	tool := tetra(1, math.V3(0, 0, 0))
	got, _ := Boolean(NewBody, tetra(1, math.V3(5, 0, 0)), tool)
	if got != tool {
		t.Error("NewBody should return the tool as a separate body")
	}
	if Join.String() != "join" || Cut.String() != "cut" || Intersect.String() != "intersect" || NewBody.String() != "new-body" {
		t.Error("operation strings wrong")
	}
}

// Sew is implemented as of M07 PBI-084 (#300) — gap behavior is covered by
// sew_test.go; an already-closed body simply promotes to a solid.
func TestSewClosedTetraSucceeds(t *testing.T) {
	t.Parallel()
	solid, err := Sew(tetra(1, math.V3(0, 0, 0)), 1e-6)
	if err != nil {
		t.Fatalf("Sew on a closed body: %v", err)
	}
	if !solid.IsSolid() {
		t.Error("sewing a closed tetra should yield a solid")
	}
}

// quadFace builds a planar square face (4 vertices) to exercise ear-clipping.
func quadFace() *topo.Face {
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	mk := func(x, y float64, i int) *topo.Vertex {
		return bld.AddVertex(math.P3(x, y, 0), topo.NewLineage(topo.Tok("f", "vertex", i)))
	}
	p0, p1, p2, p3 := mk(0, 0, 0), mk(2, 0, 1), mk(2, 2, 2), mk(0, 2, 3)
	seg := func(a, b *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(a.Point(), b.Point()), a, b, topo.NewLineage(topo.Tok("f", "edge", i)))
	}
	plane, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	return bld.AddFace(plane, topo.NewLineage(topo.Tok("f", "face", 0)),
		topo.OuterLoop(topo.Fwd(seg(p0, p1, 0)), topo.Fwd(seg(p1, p2, 1)), topo.Fwd(seg(p2, p3, 2)), topo.Fwd(seg(p3, p0, 3))))
}

func TestQuadFaceEarClipping(t *testing.T) {
	t.Parallel()
	mesh := TessellateFace(quadFace(), DefaultQuality())
	if mesh.TriangleCount() != 2 || mesh.VertexCount() != 4 {
		t.Errorf("quad → %d tris / %d verts, want 2/4", mesh.TriangleCount(), mesh.VertexCount())
	}
}

func TestBooleanIntersectingProducesValidSolids(t *testing.T) {
	t.Parallel()
	// Two tetra that overlap with neither containing the other → the intersecting case,
	// now handled by the BSP CSG (PBI-171): each operation yields a valid solid.
	a := tetra(3, math.V3(0, 0, 0))
	b := tetra(3, math.V3(0.5, 0.5, 0.5)) // overlaps a's volume, neither contains the other
	for _, op := range []PartFeatureOperation{Join, Cut, Intersect} {
		res, err := Boolean(op, a, b)
		if err != nil {
			t.Fatalf("intersecting %v: %v", op, err)
		}
		if res == nil || len(res.Faces()) == 0 {
			t.Errorf("intersecting %v produced an empty body", op)
			continue
		}
		if r := Validate(res); !r.Valid {
			t.Errorf("intersecting %v not a valid solid: %+v", op, r)
		}
	}
}

func TestValidateNonManifold(t *testing.T) {
	t.Parallel()
	// Three faces sharing one edge → that edge is used 3 times (non-manifold).
	bld := topo.NewBuilder(true, topo.NewLineage(topo.Tok("f", "body", 0)))
	mk := func(x, y, z float64, i int) *topo.Vertex {
		return bld.AddVertex(math.P3(x, y, z), topo.NewLineage(topo.Tok("f", "vertex", i)))
	}
	a, b := mk(0, 0, 0, 0), mk(1, 0, 0, 1)
	c, d, e := mk(0, 1, 0, 2), mk(0, 0, 1, 3), mk(0, -1, 0, 4)
	seg := func(x, y *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(x.Point(), y.Point()), x, y, topo.NewLineage(topo.Tok("f", "edge", i)))
	}
	ab := seg(a, b, 0)
	pl := func(n math.Vector3) geom.Surface { s, _ := geom.NewPlane(math.P3(0, 0, 0), n); return s }
	bld.AddFace(pl(math.V3(0, 0, 1)), topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(seg(b, c, 1)), topo.Rev(seg(a, c, 2))))
	bld.AddFace(pl(math.V3(0, 1, 0)), topo.NewLineage(topo.Tok("f", "face", 1)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(seg(b, d, 3)), topo.Rev(seg(a, d, 4))))
	bld.AddFace(pl(math.V3(0, 0, 1)), topo.NewLineage(topo.Tok("f", "face", 2)), topo.OuterLoop(topo.Fwd(ab), topo.Fwd(seg(b, e, 5)), topo.Rev(seg(a, e, 6))))
	r := Validate(bld.Build())
	if r.Manifold || r.Valid {
		t.Errorf("body with a 3-face edge should be non-manifold: %+v", r)
	}
}
