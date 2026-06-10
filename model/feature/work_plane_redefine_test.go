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
