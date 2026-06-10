// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestRedefineThreePointPlaneRepicksPoint is the core redefine regression: re-pointing a
// three-point plane's third point at a new location re-derives the plane from it.
func TestRedefineThreePointPlaneRepicksPoint(t *testing.T) {
	g := NewWorkGeometry()
	a := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 0, 0) })
	b := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(1, 0, 0) })
	c := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 0) })
	wp := g.WorkPlanes().AddByThreePoints(a.Key(), b.Key(), c.Key())
	if !wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Fatalf("initial three-point plane normal = %v, want +Z", wp.Plane().Normal())
	}

	slots := wp.RedefineSlots()
	if len(slots) != 3 {
		t.Fatalf("three-point plane RedefineSlots = %d, want 3", len(slots))
	}
	for i, s := range slots {
		if s.Kind != WorkRefPoint {
			t.Errorf("slot %d (%q) kind = %d, want WorkRefPoint", i, s.Label, s.Kind)
		}
	}

	// Re-point the third corner above the XY plane so the plane tilts off +Z.
	up := g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(0, 1, 1) })
	slots[2].Set(up.Key())
	g.Recompute(nil)
	if wp.Plane().Normal().AsVector().IsEqualTo(math.V3(0, 0, 1), wtol) {
		t.Errorf("after re-picking point 3, plane normal still +Z (%v) — redefine did not take", wp.Plane().Normal())
	}
}

// TestRedefineOffsetPlaneScalar: an offset plane exposes its distance via EditableScalars, and
// editing it moves the plane.
func TestRedefineOffsetPlaneScalar(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })

	sc := wp.EditableScalars()
	if len(sc) != 1 || sc[0].Label != "Offset" {
		t.Fatalf("offset plane EditableScalars = %+v, want one Offset", sc)
	}
	if sc[0].Get() != 2 {
		t.Errorf("offset Get = %v, want 2", sc[0].Get())
	}
	sc[0].Set(5)
	g.Recompute(nil)
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 5), wtol) {
		t.Errorf("after editing offset, origin = %v, want (0,0,5)", wp.Plane().Origin())
	}
	if len(wp.RedefineSlots()) != 1 || wp.RedefineSlots()[0].Kind != WorkRefPlane {
		t.Errorf("offset plane should also expose its base plane as a WorkRefPlane slot")
	}
}

// TestRedefineLineAngleScalar: a line-plane-angle plane exposes its angle, and editing it
// re-angles the plane.
func TestRedefineLineAngleScalar(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByLinePlaneAndAngle(OriginXAxis, OriginXYPlane, func() float64 { return 0 })
	n0 := wp.Plane().Normal().AsVector()

	sc := wp.EditableScalars()
	if len(sc) != 1 || sc[0].Label != "Angle" {
		t.Fatalf("line-angle plane EditableScalars = %+v, want one Angle", sc)
	}
	sc[0].Set(stdmath.Pi / 4) // 45°
	g.Recompute(nil)
	if wp.Plane().Normal().AsVector().IsEqualTo(n0, wtol) {
		t.Errorf("after editing the angle, the plane normal is unchanged (%v) — edit did not take", wp.Plane().Normal())
	}
}

// TestTangentPlaneRedefineSlots: a plane-tangent plane exposes a parallel-plane slot and a
// tangent-face slot of the right kinds (the face slot is what makes a tangent plane redefinable).
func TestTangentPlaneRedefineSlots(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByPlaneAndTangent(OriginXZPlane, FaceRef([]byte("cyl")))
	slots := wp.RedefineSlots()
	if len(slots) != 2 {
		t.Fatalf("plane-tangent RedefineSlots = %d, want 2", len(slots))
	}
	if slots[0].Kind != WorkRefPlane || slots[1].Kind != WorkRefFace {
		t.Errorf("plane-tangent slot kinds = [%d,%d], want [WorkRefPlane, WorkRefFace]", slots[0].Kind, slots[1].Kind)
	}
}

// TestOriginPlaneNotRedefinable: the origin frame's planes have nothing to edit.
func TestOriginPlaneNotRedefinable(t *testing.T) {
	g := NewWorkGeometry()
	origin := g.OriginPlanes()[0]
	if origin.IsRedefinable() || len(origin.RedefineSlots()) != 0 || len(origin.EditableScalars()) != 0 {
		t.Error("an origin plane must not be redefinable")
	}
}

// TestRedefineRejectsSelfReference is the silent-drift regression: re-picking an offset
// plane's base to the plane itself once stayed healthy=true while the plane moved by its
// offset on EVERY recompute. The slot must refuse the reference and leave the plane stable.
func TestRedefineRejectsSelfReference(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
	g.Recompute(nil)

	if err := wp.RedefineSlots()[0].Set(wp.Key()); err == nil {
		t.Fatal("re-picking a plane's base to the plane itself must be rejected")
	}
	for i := 0; i < 3; i++ { // the old bug compounded per recompute (z = 2, 4, 6, …)
		g.Recompute(nil)
	}
	if !wp.Plane().Origin().IsEqualTo(math.P3(0, 0, 2), wtol) {
		t.Errorf("origin after refused self-reference = %v, want stable (0,0,2)", wp.Plane().Origin())
	}
}

// TestRedefineRejectsReferenceCycle: with B offset from A, re-picking A's base to B would make
// A depend on itself through B — the validation walks the reference graph, not just one edge.
func TestRedefineRejectsReferenceCycle(t *testing.T) {
	g := NewWorkGeometry()
	a := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 1 })
	b := g.WorkPlanes().AddByPlaneAndOffset(a.Key(), func() float64 { return 1 })
	g.Recompute(nil)

	if err := a.RedefineSlots()[0].Set(b.Key()); err == nil {
		t.Fatal("a redefine that closes a reference cycle must be rejected")
	}
}

// TestRedefineRejectsDanglingReference: a repick naming a work feature that does not exist
// fails loudly (the wire path once turned this into a silently sick plane).
func TestRedefineRejectsDanglingReference(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
	if err := wp.RedefineSlots()[0].Set(WorkRef("plane/99")); err == nil {
		t.Fatal("a repick to a nonexistent work plane must be rejected")
	}
}

// TestRedefineScalarAndRepickCompose is the lost-update regression: the angle edit and the
// line re-pick of a line-plane-angle plane mutate one shared (pointer-held) definition, so
// applying both keeps both. The definitions were once value types whose slot closures each
// captured their own copy — the re-pick then silently discarded the angle edit.
func TestRedefineScalarAndRepickCompose(t *testing.T) {
	g := NewWorkGeometry()
	wp := g.WorkPlanes().AddByLinePlaneAndAngle(OriginXAxis, OriginXYPlane, func() float64 { return 0 })
	g.Recompute(nil)

	slots := wp.RedefineSlots() // captured before the scalar edit, like the edit tool does
	wp.EditableScalars()[0].Set(stdmath.Pi / 4)
	if err := slots[0].Set(OriginYAxis); err != nil {
		t.Fatalf("re-picking the line: %v", err)
	}
	g.Recompute(nil)
	if got := wp.EditableScalars()[0].Get(); got != stdmath.Pi/4 {
		t.Errorf("angle after the re-pick = %v, want pi/4 — the scalar edit was lost", got)
	}
	if got := wp.def.refs()[0]; got != OriginYAxis {
		t.Errorf("line ref after the scalar edit = %q, want %q", got, OriginYAxis)
	}
}
