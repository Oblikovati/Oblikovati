// SPDX-License-Identifier: GPL-2.0-only

package ops

// Coverage tests whose subjects stayed in kernel/ops when the tessellation family moved
// out: the CSG cage helpers (onSegment, isClosedCage, dropRepeats), the triangle
// constructor and the plane-meet helpers now in internal/probe.

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops/internal/mesh"
	"oblikovati.org/kernel/ops/internal/probe"
	"oblikovati.org/math"
)

func TestOnSegment(t *testing.T) {
	t.Parallel()
	a, b := math.P3(0, 0, 0), math.P3(2, 0, 0)
	tol := geom.ResolutionForPoints([]math.Point3{a, b}).Plane()
	if !mesh.OnSegment(math.P3(1, 0, 0), a, b, tol) {
		t.Error("midpoint should be on the segment")
	}
	if mesh.OnSegment(math.P3(5, 0, 0), a, b, tol) {
		t.Error("a point past the end should be off the segment")
	}
	if mesh.OnSegment(math.P3(0, 0, 0), a, a, tol) {
		t.Error("a zero-length segment contains nothing")
	}
}

func TestIsClosedCage(t *testing.T) {
	t.Parallel()
	if !mesh.IsClosedCage(map[[2]int]int{{0, 1}: 2, {1, 2}: 2}) {
		t.Error("a fully-paired cage should be closed")
	}
	if mesh.IsClosedCage(map[[2]int]int{{0, 1}: 1}) {
		t.Error("an unpaired edge means open")
	}
	if mesh.IsClosedCage(map[[2]int]int{}) {
		t.Error("an empty cage is not closed")
	}
}

func TestDropRepeats(t *testing.T) {
	t.Parallel()
	got := DropRepeats([]int{1, 1, 2, 3, 1})
	if len(got) != 3 { // consecutive dup removed + closing 1 dropped
		t.Fatalf("dropRepeats = %v, want 3 unique", got)
	}
}

func TestNewTriRejectsDegenerate(t *testing.T) {
	t.Parallel()
	if _, ok := mesh.NewTri(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)); ok {
		t.Error("a collinear (zero-area) triangle should be dropped")
	}
	if _, ok := mesh.NewTri(math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)); !ok {
		t.Error("a valid triangle should be accepted")
	}
}

func TestTrianglePlaneDegenerateFallback(t *testing.T) {
	t.Parallel()
	// Collinear triangle vertices ⇒ zero normal ⇒ fallback to +Z.
	verts := []math.Point3{math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(2, 0, 0)}
	n := mesh.TrianglePlane(verts, [3]int{0, 1, 2}).NormalAt(0, 0)
	if n.Z == 0 {
		t.Errorf("degenerate trianglePlane normal = %v, want a +Z fallback", n)
	}
}

func TestMeetOfPlanes(t *testing.T) {
	t.Parallel()
	// The three coordinate planes meet at the origin.
	x := plane(t, math.P3(0, 0, 0), math.V3(1, 0, 0))
	y := plane(t, math.P3(0, 0, 0), math.V3(0, 1, 0))
	z := plane(t, math.P3(0, 0, 0), math.V3(0, 0, 1))
	p, ok := probe.MeetOfPlanes([]geom.Plane{x, y, z})
	if !ok || p.DistanceTo(math.P3(0, 0, 0)) > 1e-9 {
		t.Fatalf("meetOfPlanes = %v,%v; want origin", p, ok)
	}
	// Three parallel planes have no single meet.
	if _, ok := probe.MeetOfPlanes([]geom.Plane{z, z, z}); ok {
		t.Error("parallel planes should not meet at a point")
	}
}

func TestTwoPlaneLine(t *testing.T) {
	t.Parallel()
	x := plane(t, math.P3(0, 0, 0), math.V3(1, 0, 0))
	y := plane(t, math.P3(0, 0, 0), math.V3(0, 1, 0))
	if _, dir, ok := probe.TwoPlaneLine(x, y); !ok || dir.LengthSquared() == 0 {
		t.Error("two perpendicular planes should meet in a line")
	}
	if _, _, ok := probe.TwoPlaneLine(x, x); ok {
		t.Error("two parallel planes should not meet in a line")
	}
}

func plane(t *testing.T, o math.Point3, n math.Vector3) geom.Plane {
	t.Helper()
	p, err := geom.NewPlane(o, n)
	if err != nil {
		t.Fatalf("NewPlane: %v", err)
	}
	return p
}
