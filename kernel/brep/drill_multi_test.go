// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati.org/math"
)

// Multi-hole exact drilling (M2 Phase 3, Oblikovati/Oblikovati#1336). drillThroughCurved must drill a
// second (and later) hole into a body that already carries a cylinder wall, where CutCylindricalHole's
// all-planar precondition fails. Each bore stays an exact watertight curved B-rep — one analytic cylinder
// wall per hole — so a drilled plate chains without ever touching CSG.

// TestDrillThroughCurvedSecondHole drills a hole into a slab that already has one, exercising the curved
// path (the target is no longer all-planar). The result is watertight with two cylinder walls and the
// slab's six planar faces.
func TestDrillThroughCurvedSecondHole(t *testing.T) {
	slab, _ := SolidBlock(math.P3(-16, -10, 0), math.P3(16, 10, 6), "slab")
	first, err := CutCylindricalHole(slab, math.P3(-6, 0, -1), math.V3(0, 0, 1), 1.5)
	if err != nil {
		t.Fatalf("first (planar) hole: %v", err)
	}
	second, err := drillThroughCurved(first, math.P3(6, 0, -1), math.V3(0, 0, 1), 1.5)
	if err != nil {
		t.Fatalf("second (curved) hole: %v", err)
	}
	assertWatertight(t, second)
	if n := len(second.Shells()); n != 1 {
		t.Errorf("twice-drilled slab has %d shells, want 1", n)
	}
	_, cyls, planes := faceTypeCounts(t, second)
	if cyls != 2 || planes != 6 {
		t.Errorf("twice-drilled slab got %d cyl + %d plane faces, want 2 + 6", cyls, planes)
	}
}

// TestDrillThroughCurvedFiveHoles drills five holes in sequence and checks each adds exactly one cylinder
// wall while the body stays a single watertight shell — the topology behind the chained-drift guard.
func TestDrillThroughCurvedFiveHoles(t *testing.T) {
	res, _ := SolidBlock(math.P3(-16, -10, 0), math.P3(16, 10, 6), "slab")
	for i, cx := range []float64{-8, -4, 0, 4, 8} {
		out, err := drillThroughCurved(res, math.P3(cx, 0, -1), math.V3(0, 0, 1), 1.5)
		if err != nil {
			t.Fatalf("bore %d: %v", i, err)
		}
		assertWatertight(t, out)
		if _, cyls, _ := faceTypeCounts(t, out); cyls != i+1 {
			t.Errorf("bore %d: %d cylinder walls, want %d", i, cyls, i+1)
		}
		res = out
	}
}

// TestDrillThroughCurvedRejectsClipped: a hole whose circle spills past the slab edge is partial, so the
// curved drill returns an error and the caller keeps its fallback.
func TestDrillThroughCurvedRejectsClipped(t *testing.T) {
	slab, _ := SolidBlock(math.P3(-16, -10, 0), math.P3(16, 10, 6), "slab")
	if _, err := drillThroughCurved(slab, math.P3(15.5, 0, -1), math.V3(0, 0, 1), 1.5); err == nil {
		t.Error("a hole clipping the slab edge should error (partial hole), got nil")
	}
}

// TestDrillThroughCurvedRejectsOverlap: a second hole overlapping the first is not a clean through-hole,
// so the curved drill rejects it.
func TestDrillThroughCurvedRejectsOverlap(t *testing.T) {
	slab, _ := SolidBlock(math.P3(-16, -10, 0), math.P3(16, 10, 6), "slab")
	first, _ := CutCylindricalHole(slab, math.P3(0, 0, -1), math.V3(0, 0, 1), 1.5)
	if _, err := drillThroughCurved(first, math.P3(1.5, 0, -1), math.V3(0, 0, 1), 1.5); err == nil {
		t.Error("a hole overlapping an existing hole should error, got nil")
	}
}
