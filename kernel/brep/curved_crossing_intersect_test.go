// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Crossing-cylinder intersection assembly (M2 Phase 2, Oblikovati/Oblikovati#1335). The split/classify/
// stitch slice must weld the traced imprint loops into a watertight analytic solid: the rod's wall band
// inside the fat cylinder plus the two fat-wall lens caps, three analytic faces, every edge shared by
// exactly two faces. Volume is checked through ops_test (brep cannot import ops); here the concern is the
// watertight topology and that the cylinder surfaces are preserved (no triangle soup).

// assertCrossingManifold checks the assembled body is a solid of the expected analytic faces with every
// edge used exactly twice — the closed-manifold invariant the stitch must keep.
func assertCrossingManifold(t *testing.T, body *topo.Body, wantFaces int) {
	t.Helper()
	if body == nil || !body.IsSolid() {
		t.Fatalf("crossing intersection is not a solid: %+v", body)
	}
	if n := len(body.Faces()); n != wantFaces {
		t.Errorf("result has %d faces, want %d (rod band + two lens caps)", n, wantFaces)
	}
	for _, e := range body.Edges() {
		if uses := len(e.Uses()); uses != 2 {
			t.Errorf("edge %v has %d uses, want 2 (a closed manifold)", e.Lineage(), uses)
		}
	}
	for _, f := range body.Faces() {
		if _, ok := f.Geometry().(geom.Cylinder); !ok {
			t.Errorf("face surface %T is not a cylinder (the exact surfaces must be preserved)", f.Geometry())
		}
	}
}

// TestCrossingCylinderIntersectThinThroughFat assembles a radius-1.5 rod (axis x) crossing a radius-3
// cylinder (axis z) through its centre into the watertight three-face intersection solid.
func TestCrossingCylinderIntersectThinThroughFat(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)

	res, ok := CrossingCylinderIntersect(fat, thin, nil)
	if !ok {
		t.Fatal("thin-through-fat intersection declined; want a three-face solid")
	}
	assertCrossingManifold(t, res, 3)
}

// TestCrossingCylinderIntersectOrderIndependent: A ∩ B = B ∩ A — the rod is detected from the imprint, not
// the argument order, so passing the rod first gives the same watertight solid.
func TestCrossingCylinderIntersectOrderIndependent(t *testing.T) {
	fat, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 3, 12)
	thin, _ := SolidCylinder(math.P3(-6, 0, 0), math.V3(1, 0, 0), 1.5, 12)

	res, ok := CrossingCylinderIntersect(thin, fat, nil) // rod first
	if !ok {
		t.Fatal("rod-first intersection declined; the assembly must be order-independent")
	}
	assertCrossingManifold(t, res, 3)
}

// TestCrossingCylinderIntersectNonCylinderDefers: the assembly only handles two bare cylinders.
func TestCrossingCylinderIntersectNonCylinderDefers(t *testing.T) {
	block, _ := SolidBlock(math.P3(-2, -2, -2), math.P3(2, 2, 2), "b")
	cyl, _ := SolidCylinder(math.P3(0, 0, -6), math.V3(0, 0, 1), 1, 12)
	if _, ok := CrossingCylinderIntersect(block, cyl, nil); ok {
		t.Error("intersection of a block and a cylinder should defer (ok=false)")
	}
}
