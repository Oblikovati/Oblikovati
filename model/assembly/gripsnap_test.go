// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/api/types"
	"oblikovati.org/math"
	"oblikovati.org/model/occurrence"
)

// snapPair sets up a grounded base + a free moving occurrence and a constraint set over them — the
// fixture the grip-snap tests build their two refs on.
func snapPair(t *testing.T, movingAt math.Matrix4) (*ConstraintSet, *occurrence.Occurrence, *occurrence.Occurrence) {
	t.Helper()
	occs := occurrence.NewOccurrences()
	base := place(occs, "base:1", math.Identity4())
	base.SetGrounded(true)
	moving := place(occs, "moving:1", movingAt)
	return NewConstraintSet(occs, nil), base, moving
}

// TestGripSnapInfers checks the grip-snap inference picks the right constraint from the two inputs'
// primitive kinds (and, for two planes, their current world orientation).
func TestGripSnapInfers(t *testing.T) {
	cases := []struct {
		name string
		a, b Primitive
		want types.AssemblyConstraintType
	}{
		{"planes-opposed→mate", PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)), PlanePrimitive(math.P3(0, 0, 5), unit(t, 0, 0, -1)), types.ConstraintMate},
		{"planes-aligned→flush", PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)), PlanePrimitive(math.P3(0, 0, 5), unit(t, 0, 0, 1)), types.ConstraintFlush},
		{"cylinders→insert", CylinderPrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), 2), CylinderPrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), 2), types.ConstraintInsert},
		{"axes→mate", LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)), LinePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)), types.ConstraintMate},
		{"plane+cylinder→tangent", PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)), CylinderPrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0), 2), types.ConstraintTangent},
		{"cylinder+plane→tangent", CylinderPrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0), 2), PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1)), types.ConstraintTangent},
		{"points→mate", PointPrimitive(math.P3(0, 0, 0)), PointPrimitive(math.P3(0, 0, 5)), types.ConstraintMate},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			set, base, moving := snapPair(t, math.Identity4())
			cst, kind, err := set.InferGripConstraint(ref(base, c.a), ref(moving, c.b), 0)
			if err != nil {
				t.Fatalf("InferGripConstraint: %v", err)
			}
			if kind != c.want || cst.Type() != c.want {
				t.Errorf("inferred %s (constraint %s), want %s", kind, cst.Type(), c.want)
			}
		})
	}
}

// TestGripSnapInferenceErrors checks the inference rejects a pair it cannot snap (a plane + a bare
// axis with no radius) and a prefer that is not a snap constraint (an angle).
func TestGripSnapInferenceErrors(t *testing.T) {
	set, base, moving := snapPair(t, math.Identity4())
	plane := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	if _, _, err := set.InferGripConstraint(ref(base, plane), ref(moving, LinePrimitive(math.P3(0, 0, 0), unit(t, 1, 0, 0))), 0); err == nil {
		t.Error("plane + bare axis (no radius) should not infer a snap constraint")
	}
	if _, _, err := set.InferGripConstraint(ref(base, plane), ref(moving, plane), types.ConstraintAngle); err == nil {
		t.Error("prefer=angle is not a snap constraint and should error")
	}
}

// TestGripSnapPreferOverrides checks an explicit prefer overrides the inference: two opposed planes
// (which would auto-infer a mate) snap as a flush when flush is preferred.
func TestGripSnapPreferOverrides(t *testing.T) {
	set, base, moving := snapPair(t, math.Identity4())
	a := PlanePrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1))
	b := PlanePrimitive(math.P3(0, 0, 5), unit(t, 0, 0, -1))
	_, kind, err := set.InferGripConstraint(ref(base, a), ref(moving, b), types.ConstraintFlush)
	if err != nil {
		t.Fatal(err)
	}
	if kind != types.ConstraintFlush {
		t.Errorf("preferred kind = %s, want flush", kind)
	}
}

// TestGripSnapSnapsIntoPlace checks the grip snap repositions the moving component: snapping its hole
// axis (offset in space) onto the base's axis inserts it so the axes are collinear at the origin.
func TestGripSnapSnapsIntoPlace(t *testing.T) {
	set, base, moving := snapPair(t, math.Translation4(math.V3(3, 4, 10)))
	_, kind, err := set.InferGripConstraint(
		ref(base, CylinderPrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), 2)),
		ref(moving, CylinderPrimitive(math.P3(0, 0, 0), unit(t, 0, 0, 1), 2)), 0)
	if err != nil {
		t.Fatal(err)
	}
	if kind != types.ConstraintInsert {
		t.Fatalf("inferred %s, want insert", kind)
	}
	set.Solve()
	if p := moving.Transform().Translation(); stdmath.Abs(p.X) > 1e-6 || stdmath.Abs(p.Y) > 1e-6 {
		t.Errorf("snapped axis point = (%g,%g,%g), want collinear with the base axis (x=y=0)", p.X, p.Y, p.Z)
	}
}
