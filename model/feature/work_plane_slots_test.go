// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	"testing"

	"oblikovati.org/math"
)

// slotTestPoint adds a user work point at (x, y, z) and returns its reference.
func slotTestPoint(g *WorkGeometry, x, y, z float64) WorkRef {
	return g.WorkPoints().AddByPosition(func() math.Point3 { return math.P3(x, y, z) }).Key()
}

// freshSlotRef returns a valid repick reference of the given kind that none of the test's
// plane builders start from, so a successful Set is observable in the definition's refs().
// Plane/axis/face picks are constants the builders avoid (YZ plane, Z axis, a distinct face
// key); a point pick is a freshly created work point.
func freshSlotRef(g *WorkGeometry, kind WorkRefKind) WorkRef {
	switch kind {
	case WorkRefPlane:
		return OriginYZPlane
	case WorkRefAxis:
		return OriginZAxis
	case WorkRefPoint:
		return slotTestPoint(g, 7, 8, 9)
	case WorkRefFace:
		return FaceRef([]byte("repicked-face-key"))
	default:
		return WorkRef("")
	}
}

// TestWorkPlaneRedefineSlotsPerKind pins the slot→definition-field mapping for EVERY
// redefinable plane kind: each kind exposes the expected slots (count + kinds, in display
// order), and Set on slot i rebinds exactly refs()[i] — slots and the definition's reference
// list share one order, which the wire repick path (SlotRepick.Slot) depends on.
func TestWorkPlaneRedefineSlotsPerKind(t *testing.T) {
	cases := []struct {
		kind  string // the definition's kindName, doubling as the case name
		build func(g *WorkGeometry) *WorkPlane
		kinds []WorkRefKind
	}{
		{
			kind: "plane-offset",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndOffset(OriginXYPlane, func() float64 { return 2 })
			},
			kinds: []WorkRefKind{WorkRefPlane},
		},
		{
			kind: "three-points",
			build: func(g *WorkGeometry) *WorkPlane {
				a := slotTestPoint(g, 0, 0, 0)
				b := slotTestPoint(g, 1, 0, 0)
				c := slotTestPoint(g, 0, 1, 0)
				return g.WorkPlanes().AddByThreePoints(a, b, c)
			},
			kinds: []WorkRefKind{WorkRefPoint, WorkRefPoint, WorkRefPoint},
		},
		{
			kind: "plane-point",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndPoint(OriginXYPlane, slotTestPoint(g, 0, 0, 3))
			},
			kinds: []WorkRefKind{WorkRefPlane, WorkRefPoint},
		},
		{
			kind: "two-planes",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTwoPlanes(OriginXYPlane, OriginXZPlane)
			},
			kinds: []WorkRefKind{WorkRefPlane, WorkRefPlane},
		},
		{
			kind: "line-plane-angle",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByLinePlaneAndAngle(OriginXAxis, OriginXYPlane, func() float64 { return 0 })
			},
			kinds: []WorkRefKind{WorkRefAxis, WorkRefPlane},
		},
		{
			kind: "two-lines",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTwoLines(OriginXAxis, OriginYAxis)
			},
			kinds: []WorkRefKind{WorkRefAxis, WorkRefAxis},
		},
		{
			kind: "normal-to-curve",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByNormalToCurve(OriginXAxis, slotTestPoint(g, 1, 0, 0))
			},
			kinds: []WorkRefKind{WorkRefAxis, WorkRefPoint},
		},
		{
			kind: "torus-midplane",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByTorusMidPlane(FaceRef([]byte("face-key")))
			},
			kinds: []WorkRefKind{WorkRefFace},
		},
		{
			kind: "point-tangent",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPointAndTangent(slotTestPoint(g, 1, 0, 0), FaceRef([]byte("face-key")))
			},
			kinds: []WorkRefKind{WorkRefPoint, WorkRefFace},
		},
		{
			kind: "plane-tangent",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByPlaneAndTangent(OriginXZPlane, FaceRef([]byte("face-key")))
			},
			kinds: []WorkRefKind{WorkRefPlane, WorkRefFace},
		},
		{
			kind: "line-tangent",
			build: func(g *WorkGeometry) *WorkPlane {
				return g.WorkPlanes().AddByLineAndTangent(OriginXAxis, FaceRef([]byte("face-key")))
			},
			kinds: []WorkRefKind{WorkRefAxis, WorkRefFace},
		},
	}
	for _, tc := range cases {
		t.Run(tc.kind, func(t *testing.T) {
			g := NewWorkGeometry()
			wp := tc.build(g)
			if wp.Kind() != tc.kind {
				t.Fatalf("Kind() = %q, want %q", wp.Kind(), tc.kind)
			}
			if !wp.IsRedefinable() {
				t.Errorf("a %s plane must be redefinable (it exposes slots)", tc.kind)
			}
			slots := wp.RedefineSlots()
			if len(slots) != len(tc.kinds) {
				t.Fatalf("RedefineSlots = %d slots, want %d", len(slots), len(tc.kinds))
			}
			for i, s := range slots {
				if s.Kind != tc.kinds[i] {
					t.Errorf("slot %d (%q) kind = %q, want %q", i, s.Label, s.Kind, tc.kinds[i])
				}
				if s.Label == "" {
					t.Errorf("slot %d has an empty label", i)
				}
			}
			assertEachSlotRebindsItsRef(t, g, wp, slots)
		})
	}
}

// assertEachSlotRebindsItsRef sets every slot to a fresh valid reference of its kind and
// asserts Set succeeds, refs()[i] becomes that reference, and no other ref moves.
func assertEachSlotRebindsItsRef(t *testing.T, g *WorkGeometry, wp *WorkPlane, slots []WorkRefSlot) {
	t.Helper()
	for i, s := range slots {
		before := append([]WorkRef(nil), wp.def.refs()...)
		newRef := freshSlotRef(g, s.Kind)
		if newRef == before[i] {
			t.Fatalf("test bug: slot %d repick %q equals the existing reference", i, newRef)
		}
		if err := s.Set(newRef); err != nil {
			t.Fatalf("slot %d (%q) Set(%q): %v", i, s.Label, newRef, err)
		}
		after := wp.def.refs()
		if after[i] != newRef {
			t.Errorf("after Set on slot %d (%q), refs()[%d] = %q, want %q — the slot does not drive its field", i, s.Label, i, after[i], newRef)
		}
		for j := range after {
			if j != i && after[j] != before[j] {
				t.Errorf("Set on slot %d also changed refs()[%d]: %q → %q", i, j, before[j], after[j])
			}
		}
	}
}
