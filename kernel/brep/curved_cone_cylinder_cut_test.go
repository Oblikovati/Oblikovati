// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Cone–cylinder cut assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). Drilling a fat cylinder with a
// crossing cone, and the two cone stubs of cone − fat, must weld into watertight analytic solids — the same
// shapes as the crossing-cylinder drill/stubs, only the rod is a cone. Volumes are checked through ops_test;
// here the concern is the watertight topology and that the surfaces stay analytic.

// TestConeCylinderCutDrillsFat drills a radius-3 cylinder with a crossing frustum and checks the result is a
// watertight four-face solid: two fat caps (planes), the holed fat wall (cylinder), and the cone tunnel band.
func TestConeCylinderCutDrillsFat(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")

	res, ok := ConeCylinderCut(cyl, cone, nil)
	if !ok {
		t.Fatal("cone drill of cylinder declined; want a four-face solid")
	}
	assertWatertight(t, res)
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 1 || cyls != 1 || planes != 2 {
		t.Errorf("drilled fat got %d cone + %d cyl + %d plane faces, want 1 (tunnel) + 1 (holed wall) + 2 (caps)",
			cones, cyls, planes)
	}
}

// TestConeCylinderCutConeMinusFatStubs subtracts the fat cylinder from the crossing cone and checks the
// result is the two disconnected tapered stubs (a two-shell solid).
func TestConeCylinderCutConeMinusFatStubs(t *testing.T) {
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")

	res, ok := ConeCylinderCut(cone, cyl, nil)
	if !ok {
		t.Fatal("cone − fat declined; want two tapered stubs")
	}
	assertWatertight(t, res)
	if n := len(res.Shells()); n != 2 {
		t.Errorf("cone − fat has %d shells, want 2 (a disconnected stub each side)", n)
	}
	cones, cyls, planes := faceTypeCounts(t, res)
	if cones != 2 || cyls != 2 || planes != 2 {
		t.Errorf("cone − fat got %d cone + %d cyl + %d plane faces, want 2 (stub bands) + 2 (lens caps) + 2 (cone end caps)",
			cones, cyls, planes)
	}
}

// TestConeCylinderCutTwoCylindersDefer: two cylinders are the crossing-cylinder drill, not this one.
func TestConeCylinderCutTwoCylindersDefer(t *testing.T) {
	a, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	b, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeCylinderCut(a, b, nil); ok {
		t.Error("two cylinders should defer from the cone–cylinder cut (ok=false)")
	}
}

// assertWatertight fails the test unless the body is a solid whose every edge is shared by exactly two faces.
func assertWatertight(t *testing.T, b *topo.Body) {
	t.Helper()
	if !b.IsSolid() {
		t.Fatalf("body is not a solid: %+v", b)
	}
	for _, e := range b.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
}

// faceTypeCounts tallies a body's faces by analytic surface type, failing the test on any non-analytic face.
func faceTypeCounts(t *testing.T, b *topo.Body) (cones, cyls, planes int) {
	t.Helper()
	for _, f := range b.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cones++
		case geom.Cylinder:
			cyls++
		case geom.Plane:
			planes++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
	return
}
