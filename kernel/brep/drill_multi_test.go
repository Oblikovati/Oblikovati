// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"strings"
	"testing"

	"oblikovati.org/kernel/topo"
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

// TestDrillRejectsCapsAcrossAVoid pins that the drill declines when its two pierced faces bound a VOID
// rather than material — the "drilling the empty gap" defect (#31).
//
// Being perpendicular pierced faces with the circle strictly inside is NOT enough to mean there is
// anything between them to remove. A slot's two facing walls satisfy every one of those preconditions,
// and the drill used to "bore" the gap between them and stitch a cylinder wall across it, MATERIALIZING
// a plug: the cut ADDED pi*r^2*slotWidth and still reported a valid solid.
//
// The shape is a slotted round-head screw, which is where this was found: a cylinder (axis +X) with a
// rectangular slot tunnelled along its axis, then drilled ACROSS at right angles. It reproduces the real
// case exactly because the ONLY faces perpendicular to the drill are the slot's two walls (z=+/-0.08) —
// the head's outer wall is curved and its caps face +/-X, so neither can be a cap candidate. That is why
// a plain stack of two blocks does NOT reproduce it: there the axis also crosses the outer z-faces, so
// the drill declines on the "exactly 2 caps" count and never reaches this check.
func TestDrillRejectsCapsAcrossAVoid(t *testing.T) {
	slotted := slottedHead(t)
	// The drill crosses the slot: material above it, the void, material below.
	_, err := drillThroughCurved(slotted, math.P3(0.35, 0, -1), math.V3(0, 0, 1), 0.1)
	if err == nil {
		t.Fatal("drilling between two faces that bound a VOID must error, got nil (it would plug the slot)")
	}
	if !strings.Contains(err.Error(), "VOID") {
		t.Errorf("declined for the wrong reason (want the void check, so the fix is what fired): %v", err)
	}
}

// slottedHead builds a cylinder (axis +X, r=0.35, x in [0,0.7]) carrying an axial rectangular slot
// (y in [-0.3,0.3], z in [-0.08,0.08]) — a slotted screw head.
func slottedHead(t *testing.T) *topo.Body {
	t.Helper()
	head, err := SolidCylinder(math.P3(0, 0, 0), math.V3(1, 0, 0), 0.35, 0.7)
	if err != nil {
		t.Fatalf("head cylinder: %v", err)
	}
	// CCW about +X at the base cap; every corner is inside the disk (|(0.3,0.08)| = 0.31 < 0.35).
	slot := []math.Point3{
		math.P3(0, -0.3, -0.08), math.P3(0, 0.3, -0.08), math.P3(0, 0.3, 0.08), math.P3(0, -0.3, 0.08),
	}
	slotted, err := SubtractAxialPrism(head, slot)
	if err != nil {
		t.Fatalf("slot the head: %v", err)
	}
	return slotted
}

// TestDrillStillTakesATrueThroughHole guards the fix's OTHER side: the void check must not reject a
// genuine drill. On a real slab the entry normal opposes the travel and the exit normal runs with it —
// exactly what tells it apart from the facing-walls pair above, which has those signs inverted.
func TestDrillStillTakesATrueThroughHole(t *testing.T) {
	slab, _ := SolidBlock(math.P3(-16, -10, 0), math.P3(16, 10, 6), "slab")
	res, err := drillThroughCurved(slab, math.P3(0, 0, -1), math.V3(0, 0, 1), 1.5)
	if err != nil {
		t.Fatalf("a clean through-hole must still drill: %v", err)
	}
	assertWatertight(t, res)
	if _, cyls, planes := faceTypeCounts(t, res); cyls != 1 || planes != 6 {
		t.Errorf("drilled slab got %d cyl + %d plane faces, want 1 + 6", cyls, planes)
	}
}
