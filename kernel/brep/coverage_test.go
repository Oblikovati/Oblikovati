// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"errors"
	"testing"

	"oblikovati/kernel/geom"
	"oblikovati/kernel/topo"
	"oblikovati/math"
)

func TestBooleanRejectsNonPlanarOperand(t *testing.T) {
	// A body with a cylindrical face cannot be handled by the planar B-rep boolean.
	cyl := oneFaceBody(t, true)
	tri := oneFaceBody(t, false)
	if _, err := Boolean(Union, cyl, tri); !errors.Is(err, ErrNonPlanar) {
		t.Fatalf("Boolean(non-planar) err = %v, want ErrNonPlanar", err)
	}
	// Symmetric: a planar a with a non-planar b also rejects.
	if _, err := Boolean(Union, tri, cyl); !errors.Is(err, ErrNonPlanar) {
		t.Fatalf("Boolean(planar, non-planar) err = %v, want ErrNonPlanar", err)
	}
}

// oneFaceBody builds a one-face body; curved=true gives a cylindrical face, else a planar
// triangle.
func oneFaceBody(t *testing.T, curved bool) *topo.Body {
	t.Helper()
	bld := topo.NewBuilder(false, topo.NewLineage(topo.Tok("f", "body", 0)))
	if curved {
		surf, _ := geom.NewCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 1)
		v := bld.AddVertex(math.P3(1, 0, 0), topo.NewLineage(topo.Tok("f", "vertex", 0)))
		e := bld.AddEdge(geom.NewLineSegment(math.P3(1, 0, 0), math.P3(1, 0, 1)), v, v, topo.NewLineage(topo.Tok("f", "edge", 0)))
		bld.AddFace(surf, topo.NewLineage(topo.Tok("f", "face", 0)), topo.OuterLoop(topo.Fwd(e)))
		return bld.Build()
	}
	mk := func(p math.Point3, i int) *topo.Vertex {
		return bld.AddVertex(p, topo.NewLineage(topo.Tok("f", "vertex", i)))
	}
	a, b, c := mk(math.P3(0, 0, 0), 0), mk(math.P3(1, 0, 0), 1), mk(math.P3(0, 1, 0), 2)
	seg := func(p, q *topo.Vertex, i int) *topo.Edge {
		return bld.AddEdge(geom.NewLineSegment(p.Point(), q.Point()), p, q, topo.NewLineage(topo.Tok("f", "edge", i)))
	}
	pl, _ := geom.NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	bld.AddFace(pl, topo.NewLineage(topo.Tok("f", "face", 0)),
		topo.OuterLoop(topo.Fwd(seg(a, b, 0)), topo.Fwd(seg(b, c, 1)), topo.Rev(seg(a, c, 2))))
	return bld.Build()
}

func TestUnitOfZeroVectorIsZero(t *testing.T) {
	if u := unit(math.V3(0, 0, 0)); u.Length() != 0 {
		t.Fatalf("unit(0) = %v, want zero", u)
	}
}

func TestInHoles2D(t *testing.T) {
	hole := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(2, 2), math.P2(0, 2)}
	if !inHoles2D(math.P2(1, 1), [][]math.Point2{hole}) {
		t.Error("point inside the hole reported outside")
	}
	if inHoles2D(math.P2(5, 5), [][]math.Point2{hole}) {
		t.Error("point outside the hole reported inside")
	}
	if inHoles2D(math.P2(1, 1), nil) {
		t.Error("no holes ⇒ never inside")
	}
}

func TestAllEdgesPaired(t *testing.T) {
	if !allEdgesPaired(map[[2]int]int{{0, 1}: 2, {1, 2}: 2}) {
		t.Error("all-paired edges should report paired")
	}
	if allEdgesPaired(map[[2]int]int{{0, 1}: 1}) {
		t.Error("an unpaired edge should report not paired")
	}
}

func TestOrientRingBothWindings(t *testing.T) {
	// A CCW square on z=0 has a +Z Newell normal.
	ccw := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(2, 2, 0), math.P3(0, 2, 0)}
	z := math.V3(0, 0, 1)
	if got := orientRing(ccw, z, true); !sameStart(got, ccw[0]) {
		t.Error("an already-aligned outer ring should be left as is")
	}
	rev := orientRing(ccw, z, false) // wants the hole winding ⇒ reversed
	if sameRingOrder(rev, ccw) {
		t.Error("a misaligned ring should be reversed")
	}
}

func TestReverseSubFaceFlipsNormalAndRings(t *testing.T) {
	sf := subFace{
		outer:  []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)},
		holes:  [][]math.Point3{{math.P3(0.2, 0.2, 0), math.P3(0.4, 0.2, 0), math.P3(0.2, 0.4, 0)}},
		normal: math.V3(0, 0, 1),
	}
	firstOuter, lastOuter := sf.outer[0], sf.outer[2]
	r := reverseSubFace(sf)
	if r.normal.Z != -1 {
		t.Errorf("reversed normal Z = %v, want -1", r.normal.Z)
	}
	// reverseRing reverses: [a,b,c] → [c,b,a].
	if r.outer[0] != lastOuter || r.outer[2] != firstOuter {
		t.Errorf("outer ring not reversed: %v", r.outer)
	}
	if len(r.holes) != 1 || r.holes[0][0] != math.P3(0.2, 0.4, 0) {
		t.Fatalf("reversed hole not reversed: %v", r.holes)
	}
}

func TestRayHitsFaceBranches(t *testing.T) {
	faces, ok := facesOf(oneFaceBody(t, false)) // a z=0 triangle (normal +Z)
	if !ok || len(faces) != 1 {
		t.Fatalf("facesOf(triangle) = %v, %v", len(faces), ok)
	}
	f := faces[0]
	// Hit: from below the plane, shooting +Z through an interior point.
	if !rayHitsFace(math.P3(0.2, 0.2, -1), math.V3(0, 0, 1), f) {
		t.Error("ray through the interior should hit")
	}
	// Parallel: a ray in the plane never hits.
	if rayHitsFace(math.P3(0.2, 0.2, 1), math.V3(1, 0, 0), f) {
		t.Error("a ray parallel to the face should miss")
	}
	// Behind: the face is behind the ray origin along +Z.
	if rayHitsFace(math.P3(0.2, 0.2, 1), math.V3(0, 0, 1), f) {
		t.Error("a face behind the ray should miss")
	}
}

func TestInsideSolidRejectsNonPlanar(t *testing.T) {
	if insideSolid(oneFaceBody(t, true), math.P3(0, 0, 0)) {
		t.Error("insideSolid of a non-planar body should be false")
	}
}

func TestFaceLineIntervalsPerpendicularLine(t *testing.T) {
	faces, _ := facesOf(oneFaceBody(t, false)) // z=0 triangle
	f := faces[0]
	// A line along +Z is perpendicular to the z=0 plane ⇒ no in-plane extent.
	if got := faceLineIntervals(f, math.P3(0.2, 0.2, 0), math.V3(0, 0, 1)); got != nil {
		t.Errorf("perpendicular line intervals = %v, want nil", got)
	}
	// An in-plane line crosses the triangle, producing one interval.
	if got := faceLineIntervals(f, math.P3(-1, 0.3, 0), math.V3(1, 0, 0)); len(got) != 1 {
		t.Errorf("in-plane line intervals = %v, want 1", got)
	}
}

func TestWelder3RingDedupesAndCloses(t *testing.T) {
	w := newWelder3()
	// A loop with a repeated consecutive vertex and a closing duplicate.
	loop := []math.Point3{math.P3(0, 0, 0), math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0), math.P3(0, 0, 0)}
	r := w.ring(loop)
	if len(r) != 3 {
		t.Fatalf("welded ring = %v, want 3 unique indices", r)
	}
}

func TestInteriorPoint2D(t *testing.T) {
	square := Face2D{Outer: []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(0, 4)}}
	p, ok := interiorPoint2D(square)
	if !ok || !pointInPolygon2D(p, square.Outer) {
		t.Fatalf("interiorPoint2D = %v,%v; want an interior point", p, ok)
	}
	// A degenerate (< 3 vertices) region has no interior.
	if _, ok := interiorPoint2D(Face2D{Outer: []math.Point2{math.P2(0, 0), math.P2(1, 1)}}); ok {
		t.Error("a 2-vertex region should report no interior point")
	}
}

func TestVerticesOnSegment(t *testing.T) {
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(1, 0, 0)}
	// Vertex 2 (1,0,0), which is off the ring, lies on segment 0→1.
	got := verticesOnSegment(0, 1, []int{0, 1}, verts)
	if len(got) != 1 || got[0] != 2 {
		t.Fatalf("verticesOnSegment = %v, want [2]", got)
	}
	// A zero-length segment yields nothing.
	if verticesOnSegment(0, 0, []int{0, 1}, verts) != nil {
		t.Error("zero-length segment should have no interior vertices")
	}
}

func TestArrangeEmptyAndDegenerate(t *testing.T) {
	if Arrange(nil) != nil {
		t.Error("Arrange of no segments should be nil")
	}
	// A single zero-length segment yields no edges ⇒ nil.
	if Arrange([][2]math.Point2{{math.P2(1, 1), math.P2(1, 1)}}) != nil {
		t.Error("Arrange of a degenerate segment should be nil")
	}
}

// sameStart reports whether a ring begins at p.
func sameStart(ring []math.Point3, p math.Point3) bool { return len(ring) > 0 && ring[0] == p }

// sameRingOrder reports whether two rings have identical vertex order.
func sameRingOrder(a, b []math.Point3) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
