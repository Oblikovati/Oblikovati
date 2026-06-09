// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
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
