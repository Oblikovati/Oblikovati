// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/math"
)

func plane(t *testing.T, o math.Point3, n math.Vector3) geom.Plane {
	t.Helper()
	p, err := geom.NewPlane(o, n)
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return p
}

func TestMeetOfPlanes(t *testing.T) {
	// The three coordinate planes meet at the origin.
	x := plane(t, math.P3(0, 0, 0), math.V3(1, 0, 0))
	y := plane(t, math.P3(0, 0, 0), math.V3(0, 1, 0))
	z := plane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	p, ok := meetOfPlanes([]geom.Plane{x, y, z})
	if !ok || p.DistanceTo(math.P3(0, 0, 0)) > 1e-9 {
		t.Fatalf("meetOfPlanes = %v,%v; want origin", p, ok)
	}
	// Three parallel planes have no single meet.
	if _, ok := meetOfPlanes([]geom.Plane{z, z, z}); ok {
		t.Error("parallel planes should not meet at a point")
	}
}

func TestTwoPlaneLine(t *testing.T) {
	x := plane(t, math.P3(0, 0, 0), math.V3(1, 0, 0))
	y := plane(t, math.P3(0, 0, 0), math.V3(0, 1, 0))
	if _, dir, ok := twoPlaneLine(x, y); !ok || dir.LengthSquared() == 0 {
		t.Error("two perpendicular planes should meet in a line")
	}
	if _, _, ok := twoPlaneLine(x, x); ok {
		t.Error("two parallel planes should not meet in a line")
	}
}

func TestInTriAny(t *testing.T) {
	a, b, c := math.P2(0, 0), math.P2(4, 0), math.P2(0, 4)
	if !inTriAny(math.P2(1, 1), a, b, c) {
		t.Error("interior point reported outside the triangle")
	}
	if inTriAny(math.P2(5, 5), a, b, c) {
		t.Error("exterior point reported inside the triangle")
	}
}

func TestOrientBothWindings(t *testing.T) {
	ccw2 := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(0, 2)}
	ccw3 := []math.Point3{math.P3(0, 0, 0), math.P3(2, 0, 0), math.P3(0, 2, 0)}
	// Asking for CCW when already CCW leaves it unchanged.
	if r2, _ := orient(ccw2, ccw3, true); r2[0] != ccw2[0] {
		t.Error("an already-CCW loop should be unchanged")
	}
	// Asking for CW (ccw=false) reverses it.
	r2, _ := orient(ccw2, ccw3, false)
	if r2[0] == ccw2[0] && r2[1] == ccw2[1] {
		t.Error("a CW request should reverse a CCW loop")
	}
}

func TestCurveAtBounds(t *testing.T) {
	if curveAt(nil, 0) != nil {
		t.Error("curveAt past the end should be nil")
	}
	c := geom.NewLineSegment(math.P3(0, 0, 0), math.P3(1, 0, 0))
	if curveAt([]geom.Curve3{c}, 0) == nil {
		t.Error("curveAt within range should return the curve")
	}
}

func TestOnSegment(t *testing.T) {
	a, b := math.P3(0, 0, 0), math.P3(2, 0, 0)
	if !onSegment(math.P3(1, 0, 0), a, b) {
		t.Error("midpoint should be on the segment")
	}
	if onSegment(math.P3(5, 0, 0), a, b) {
		t.Error("a point past the end should be off the segment")
	}
	if onSegment(math.P3(0, 0, 0), a, a) {
		t.Error("a zero-length segment contains nothing")
	}
}

func TestIsClosedCage(t *testing.T) {
	if !isClosedCage(map[[2]int]int{{0, 1}: 2, {1, 2}: 2}) {
		t.Error("a fully-paired cage should be closed")
	}
	if isClosedCage(map[[2]int]int{{0, 1}: 1}) {
		t.Error("an unpaired edge means open")
	}
	if isClosedCage(map[[2]int]int{}) {
		t.Error("an empty cage is not closed")
	}
}

func TestDropRepeats(t *testing.T) {
	got := dropRepeats([]int{1, 1, 2, 3, 1})
	if len(got) != 3 { // consecutive dup removed + closing 1 dropped
		t.Fatalf("dropRepeats = %v, want 3 unique", got)
	}
}

func TestNewTriRejectsDegenerate(t *testing.T) {
	if _, ok := newTri(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)); ok {
		t.Error("a collinear (zero-area) triangle should be dropped")
	}
	if _, ok := newTri(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)); !ok {
		t.Error("a valid triangle should be accepted")
	}
}

func TestHolesByRightmost(t *testing.T) {
	h1 := []math.Point2{math.P2(0, 0), math.P2(1, 0), math.P2(0, 1)} // rightmost X = 1
	h2 := []math.Point2{math.P2(3, 0), math.P2(4, 0), math.P2(3, 1)} // rightmost X = 4
	order := holesByRightmost([][]math.Point2{h1, h2})
	if len(order) != 2 || order[0] != 1 {
		t.Fatalf("holesByRightmost = %v, want the X=4 hole (index 1) first", order)
	}
}

func TestFindVisibleVertexHitAndMiss(t *testing.T) {
	outer := []math.Point2{math.P2(0, 0), math.P2(4, 0), math.P2(4, 4), math.P2(0, 4)}
	// A point inside casts a +X ray that crosses the right edge ⇒ a visible vertex.
	if i := findVisibleVertex(outer, math.P2(1, 2)); i < 0 || i >= len(outer) {
		t.Errorf("findVisibleVertex(inside) = %d, want a valid index", i)
	}
	// A point to the right of everything sees no edge ⇒ fallback index 0.
	if i := findVisibleVertex(outer, math.P2(9, 2)); i != 0 {
		t.Errorf("findVisibleVertex(miss) = %d, want 0", i)
	}
}

func TestEarClipConcavePolygon(t *testing.T) {
	// An L-shape (one reflex vertex) forces findEar to skip non-ear vertices.
	l := []math.Point2{
		math.P2(0, 0), math.P2(2, 0), math.P2(2, 1),
		math.P2(1, 1), math.P2(1, 2), math.P2(0, 2),
	}
	if tris := earClip(l); len(tris) != len(l)-2 {
		t.Fatalf("earClip produced %d triangles, want %d", len(tris), len(l)-2)
	}
}

func TestSignedAreaSign(t *testing.T) {
	ccw := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(0, 2)}
	if signedArea(ccw) <= 0 {
		t.Error("CCW loop should have positive signed area")
	}
	cw := []math.Point2{math.P2(0, 0), math.P2(0, 2), math.P2(2, 0)}
	if signedArea(cw) >= 0 {
		t.Error("CW loop should have negative signed area")
	}
}

func TestTrianglePlaneDegenerateFallback(t *testing.T) {
	// Collinear triangle vertices ⇒ zero normal ⇒ fallback to +Z.
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}
	n := trianglePlane(verts, [3]int{0, 1, 2}).NormalAt(0, 0)
	if n.Z == 0 {
		t.Errorf("degenerate trianglePlane normal = %v, want a +Z fallback", n)
	}
}
