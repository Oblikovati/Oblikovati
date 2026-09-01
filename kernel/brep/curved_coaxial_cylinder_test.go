// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/kernel/topo"
	"oblikovati.org/math"
)

// Coaxial cylinder union (M2 Phase 3, Oblikovati/Oblikovati#1336). Two coaxial equal-radius cylinders that
// overlap or abut along the axis union to ONE taller cylinder — an exact analytic solid, not the faceted
// CSG the general path produced for this coincident-side-surface (coplanar/tangent) case.

// TestCoaxialCylinderUnionOverlap unions z∈[0,4] with z∈[3,7] (overlapping) and checks the result is one
// watertight cylinder spanning z∈[0,7]: three faces (one cylinder side + two caps).
func TestCoaxialCylinderUnionOverlap(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 2, 4)
	res, ok := CoaxialCylinderUnion(a, b)
	if !ok {
		t.Fatal("coaxial overlap union declined; want one taller cylinder")
	}
	assertWatertight(t, res)
	_, cyls, planes := faceTypeCounts(t, res)
	if cyls != 1 || planes != 2 {
		t.Errorf("coaxial union got %d cyl + %d plane faces, want 1 + 2 (one cylinder)", cyls, planes)
	}
}

// TestCoaxialCylinderUnionAbut unions z∈[0,4] with z∈[4,8] (touching end to end) into one z∈[0,8] cylinder.
func TestCoaxialCylinderUnionAbut(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 2, 4)
	res, ok := CoaxialCylinderUnion(a, b)
	if !ok {
		t.Fatal("coaxial abutting union declined; want one cylinder")
	}
	assertWatertight(t, res)
	if _, cyls, _ := faceTypeCounts(t, res); cyls != 1 {
		t.Errorf("abutting union got %d cylinder faces, want 1", cyls)
	}
}

// TestCoaxialEqualCylinders covers the predicate the model boolean gate uses (#1831): true for a
// coaxial equal-radius bare-cylinder pair (overlap or abut), false for a different radius, an offset
// axis, or a non-cylinder body — so a JOIN is routed to the analytic path only when it applies.
func TestCoaxialEqualCylinders(t *testing.T) {
	t.Parallel()
	base, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	abut, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 2, 4)
	overlap, _ := SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 2, 4)
	if !CoaxialEqualCylinders(base, abut) || !CoaxialEqualCylinders(base, overlap) {
		t.Error("coaxial equal-radius cylinders (abut/overlap) should report true")
	}
	bigR, _ := SolidCylinder(math.P3(0, 0, 4), math.V3(0, 0, 1), 3, 4)
	offset, _ := SolidCylinder(math.P3(5, 0, 4), math.V3(0, 0, 1), 2, 4)
	block, _ := SolidBlock(math.P3(0, 0, 0), math.P3(2, 2, 2), "b")
	for name, other := range map[string]*topo.Body{"bigger radius": bigR, "offset axis": offset, "block": block} {
		if CoaxialEqualCylinders(base, other) {
			t.Errorf("%s should NOT report as a coaxial equal-radius cylinder pair", name)
		}
	}
}

// TestCoaxialCylinderUnionOrderIndependent: union resolves whichever cylinder is passed first.
func TestCoaxialCylinderUnionOrderIndependent(t *testing.T) {
	t.Parallel()
	a, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	b, _ := SolidCylinder(math.P3(0, 0, 3), math.V3(0, 0, 1), 2, 4)
	if _, ok := CoaxialCylinderUnion(b, a); !ok {
		t.Error("coaxial union should resolve with the upper cylinder passed first too")
	}
}

// TestCoaxialCylinderUnionRejectsOffCase: different radius, parallel-but-offset axis, and an axial gap each
// defer (ok=false) so the general boolean keeps the case.
func TestCoaxialCylinderUnionRejectsOffCase(t *testing.T) {
	t.Parallel()
	base, _ := SolidCylinder(math.P3(0, 0, 0), math.V3(0, 0, 1), 2, 4)
	cases := map[string]*topo.Body{
		"different radius": mustCyl(math.P3(0, 0, 3), 3, 4),
		"offset axis":      mustCyl(math.P3(5, 0, 3), 2, 4),
		"axial gap":        mustCyl(math.P3(0, 0, 6), 2, 4), // base z6, gap (0..4) vs (6..10)
	}
	for name, tool := range cases {
		if _, ok := CoaxialCylinderUnion(base, tool); ok {
			t.Errorf("%s: expected the coaxial union to defer (ok=false)", name)
		}
	}
}

// mustCyl builds a validated solid cylinder of radius r, height h, with its base at base, along +Z.
func mustCyl(base math.Point3, r, h float64) *topo.Body {
	c, _ := SolidCylinder(base, math.V3(0, 0, 1), r, h)
	return c
}
