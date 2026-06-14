// SPDX-License-Identifier: GPL-2.0-only

package assembly

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
)

// TestPrimitiveTransformedBy checks a primitive's point and direction map through a rigid
// transform (the head uses this to convert an assembly-space picked face back into a
// component's definition space via the occurrence's inverse placement).
func TestPrimitiveTransformedBy(t *testing.T) {
	nx, _ := math.NewUnitVector3(1, 0, 0)
	prim := PlanePrimitive(math.P3(1, 0, 0), nx)

	// A pure translation moves the point but not the normal.
	moved := prim.TransformedBy(math.Translation4(math.V3(0, 0, 5)))
	if !moved.point.IsEqualTo(math.P3(1, 0, 5), 1e-9) {
		t.Errorf("translated point = %+v, want (1,0,5)", moved.point)
	}
	if !moved.dir.AsVector().IsEqualTo(math.V3(1, 0, 0), 1e-9) {
		t.Errorf("translated normal = %+v, want +X (unchanged)", moved.dir)
	}

	// A 90° rotation about +Z maps +X onto +Y and the point with it.
	q := math.QuaternionFromAxisAngle(mustZ(t), stdmath.Pi/2)
	rot := prim.TransformedBy(q.Matrix4())
	if !rot.dir.AsVector().IsEqualTo(math.V3(0, 1, 0), 1e-9) {
		t.Errorf("rotated normal = %+v, want +Y", rot.dir)
	}
	if !rot.point.IsEqualTo(math.P3(0, 1, 0), 1e-9) {
		t.Errorf("rotated point = %+v, want (0,1,0)", rot.point)
	}
}

func mustZ(t *testing.T) math.UnitVector3 {
	t.Helper()
	z, err := math.NewUnitVector3(0, 0, 1)
	if err != nil {
		t.Fatal(err)
	}
	return z
}
