// SPDX-License-Identifier: GPL-2.0-only

package doc

import (
	"math"
	"testing"

	obkmath "oblikovati.org/math"
)

func TestIdentityCubeOrientIsNoOp(t *testing.T) {
	o := IdentityCubeOrient()
	if !o.IsIdentity() {
		t.Fatal("IdentityCubeOrient should report IsIdentity")
	}
	v := obkmath.V3(1, 2, 3)
	if o.ToWorld(v) != v || o.ToLocal(v) != v {
		t.Errorf("identity must map vectors unchanged: world=%v local=%v", o.ToWorld(v), o.ToLocal(v))
	}
}

func TestToWorldToLocalRoundTrip(t *testing.T) {
	// A non-identity front: looking from +X (fwd = −X), up +Z.
	o := FrontFromView(obkmath.V3(-1, 0, 0), obkmath.V3(0, 0, 1))
	if o.IsIdentity() {
		t.Fatal("expected a non-identity orientation")
	}
	v := obkmath.V3(0.3, -0.7, 0.2)
	got := o.ToLocal(o.ToWorld(v))
	for _, d := range []float64{got.X - v.X, got.Y - v.Y, got.Z - v.Z} {
		if math.Abs(d) > 1e-9 {
			t.Fatalf("ToLocal∘ToWorld = %v, want %v (rotation must be orthonormal)", got, v)
		}
	}
}

func TestFrontFromViewMakesViewTheFront(t *testing.T) {
	// Looking from +X at the origin: fwd = −X. After "Set as Front", the FRONT region's
	// world look-from direction must be +X (so the snap reproduces this view).
	o := FrontFromView(obkmath.V3(-1, 0, 0), obkmath.V3(0, 0, 1))
	front := o.ToWorld(obkmath.V3(0, -1, 0)) // FRONT region local dir (look-from)
	if front != obkmath.V3(1, 0, 0) {
		t.Errorf("FRONT look-from = %v, want (1,0,0)", front)
	}
}
