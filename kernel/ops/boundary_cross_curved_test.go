// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"testing"

	"oblikovati.org/kernel/brep"
	m "oblikovati.org/math"
)

// These regressions exercise the ANALYTIC boundary-crossing guard (boundary_cross.go, #3423) on CURVED
// faces. The existing #1315 regression (TestClassifyNonConvexVertexInsideButBoundaryCrosses) only covers
// planar crossings; boundariesCross now intersects true cylinder/plane surface pairs via
// geom.SurfaceIntersect + brep.PointInFaceTrim, so a curved case must prove both verdicts.
//
// A cylinder (not a sphere) is the inner curved solid: brep.SolidSphere is a boundary-less single face
// with NO vertices, so allVerticesInside would be trivially false and the containment premise unprovable.
// brep.SolidCylinder carries two real seam vertices on its rim, so the vertex-inside premise is genuine.

// TestBoundariesCrossCurvedContainedIsFalse: a radius-1 cylinder sits fully inside a 10-cube with
// clearance on every wall. Its two rim vertices are inside the box (premise), yet no cylinder face reaches
// a box wall, so the analytic guard reports NO crossing and classify takes the contains fast-path.
func TestBoundariesCrossCurvedContainedIsFalse(t *testing.T) {
	outer, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 10), "outer")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	// Axis-Z cylinder centred in the cube: x,y within [4,6], z within [3,7] — 3..4 units of wall clearance.
	inner, err := brep.SolidCylinder(m.P3(5, 5, 3), m.V3(0, 0, 1), 1, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	if !allVerticesInside(inner, outer) {
		t.Fatal("test premise broken: not all cylinder vertices are inside the box")
	}
	if boundariesCross(outer, inner) {
		t.Error("boundariesCross = true, want false (the cylinder is fully interior; no face touches a wall)")
	}
	if rel := classify(outer, inner); rel != targetContainsTool {
		t.Errorf("classify = %v, want targetContainsTool (curved contains fast-path)", rel)
	}
}

// TestBoundariesCrossCurvedStraddleIsTrue: a radius-1 cylinder along +x is pushed so it pokes through the
// x=10 wall (it spans x in [8,12]). Its curved side surface genuinely crosses the wall plane, so the
// analytic guard must detect the crossing; the far rim vertex is outside the box, so classify is
// intersecting. This confirms the curved face-face crossing is caught, not stepped over.
func TestBoundariesCrossCurvedStraddleIsTrue(t *testing.T) {
	outer, err := brep.SolidBlock(m.P3(0, 0, 0), m.P3(10, 10, 10), "outer")
	if err != nil {
		t.Fatalf("SolidBlock: %v", err)
	}
	// Base at x=8, axis +x, height 4 → the top cap sits at x=12, outside the wall at x=10.
	inner, err := brep.SolidCylinder(m.P3(8, 5, 5), m.V3(1, 0, 0), 1, 4)
	if err != nil {
		t.Fatalf("SolidCylinder: %v", err)
	}
	// The top rim vertex is at x=12 (outside), so this is not a candidate-containment pair: the guard is
	// asserted directly, and classify falls through to intersecting.
	if !boundariesCross(outer, inner) {
		t.Error("boundariesCross = false, want true (the cylinder side pierces the x=10 wall)")
	}
	if rel := classify(outer, inner); rel != intersecting {
		t.Errorf("classify = %v, want intersecting (the cylinder straddles a wall)", rel)
	}
}
