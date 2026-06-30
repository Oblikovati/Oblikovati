// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/subd"
	"oblikovati.org/math"
)

// TestHullFeatureWrapsTwoBodies hulls two 2×2×2 boxes separated along X but sharing the same
// Y,Z extent — their convex hull is exactly the enclosing 6×2×2 box (volume 24). The feature
// must leave one valid solid.
func TestHullFeatureWrapsTwoBodies(t *testing.T) {
	a := subd.ToBody(subd.Box(2, 2, 2), "a") // [0,2]^3
	bm := subd.Box(2, 2, 2)
	for i := range bm.Verts {
		bm.Verts[i] = bm.Verts[i].TranslateBy(math.V3(4, 0, 0)) // [4,6]×[0,2]×[0,2]
	}
	b := subd.ToBody(bm, "b")

	fs := NewPartFeatures(nil)
	NewBaseFeatures(fs).AddBase(a, b)
	hull := NewHullFeatures(fs).Add()
	fs.Recompute()

	if !hull.Health().OK() {
		t.Fatalf("hull sick: %+v", hull.Health())
	}
	if got := len(fs.Result()); got != 1 {
		t.Fatalf("hull result = %d bodies, want 1", got)
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("hull body invalid: %+v", r)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-24) > 1e-9 {
		t.Errorf("hull volume = %.6f, want 24 (enclosing 6×2×2 box)", v)
	}
}
