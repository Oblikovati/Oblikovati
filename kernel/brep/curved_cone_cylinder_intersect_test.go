// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// Cone–cylinder intersection assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). A cone crossing a cylinder
// must weld into the same three-face shape as the crossing-cylinder intersection — a cone band plus two
// cylinder-wall lens caps — watertight. Volume is checked through ops_test; here the concern is the
// watertight topology and that the surfaces stay analytic (one cone band, two cylinder lens caps).

// TestConeCylinderIntersectThreeFaces crosses a frustum through a radius-3 cylinder and checks the result is
// a watertight three-face solid: one cone band and two cylinder lens caps.
func TestConeCylinderIntersectThreeFaces(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)

	res, ok := ConeCylinderIntersect(cone, cyl)
	if !ok {
		t.Fatal("cone–cylinder intersection declined; want a three-face solid")
	}
	if !res.IsSolid() {
		t.Fatalf("cone–cylinder intersection is not a solid: %+v", res)
	}
	for _, e := range res.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	cones, cyls := 0, 0
	for _, f := range res.Faces() {
		switch f.Geometry().(type) {
		case geom.Cone:
			cones++
		case geom.Cylinder:
			cyls++
		default:
			t.Errorf("face surface %T is not analytic", f.Geometry())
		}
	}
	if cones != 1 || cyls != 2 {
		t.Errorf("got %d cone + %d cylinder faces, want 1 (band) + 2 (lens caps)", cones, cyls)
	}
}

// TestConeCylinderIntersectOrderIndependent: resolving works whichever body is passed first.
func TestConeCylinderIntersectOrderIndependent(t *testing.T) {
	cone, _ := SolidCylinderCone(math.P3(-6, 0, 0), math.P3(6, 0, 0), 1, 2.5, "cone")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeCylinderIntersect(cyl, cone); !ok {
		t.Error("cone–cylinder intersection should resolve with the cylinder passed first too")
	}
}

// TestConeCylinderIntersectTwoCylindersDefer: two cylinders are the crossing-cylinder case, not this one.
func TestConeCylinderIntersectTwoCylindersDefer(t *testing.T) {
	a, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)
	b, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	if _, ok := ConeCylinderIntersect(a, b); ok {
		t.Error("two cylinders should defer from the cone–cylinder assembler (ok=false)")
	}
}
