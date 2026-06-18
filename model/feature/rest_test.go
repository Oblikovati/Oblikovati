// SPDX-License-Identifier: GPL-2.0-only

package feature

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/ops"
)

// A raised rest pad adds a 4×4×1 landing on the top face → block volume + 16 (#486).
func TestRestRaisesPad(t *testing.T) {
	fs := embossedBlock(t) // a 10×10×2 block (vol 200)
	rs := squareOn(planeAtZ(2), 4, 3)
	pad := NewPlasticFeatures(fs).AddRest(rs, []int{0}, func() float64 { return 1 }, false, 0)
	fs.Recompute()
	if !pad.Health().OK() {
		t.Fatalf("rest pad went sick: %+v", pad.Health())
	}
	body := fs.Result()[0]
	if r := ops.Validate(body); !r.Valid || !body.IsSolid() {
		t.Fatalf("rest body not valid: %+v", r.Issues)
	}
	if v := ops.BodyGeometryProperties(body, ops.DefaultQuality()).Volume; stdmath.Abs(v-216) > 1e-6 {
		t.Errorf("raised rest volume = %g, want 216 (200 + 4×4×1)", v)
	}
}

// A recessed rest cuts a 4×4×1 pocket into the top face → block volume − 16.
func TestRestRecesses(t *testing.T) {
	fs := embossedBlock(t)
	rs := squareOn(planeAtZ(2), 4, 3)
	pad := NewPlasticFeatures(fs).AddRest(rs, []int{0}, func() float64 { return 1 }, true, 0) // recessed
	fs.Recompute()
	if !pad.Health().OK() {
		t.Fatalf("recessed rest went sick: %+v", pad.Health())
	}
	if v := ops.BodyGeometryProperties(fs.Result()[0], ops.DefaultQuality()).Volume; stdmath.Abs(v-184) > 1e-6 {
		t.Errorf("recessed rest volume = %g, want 184 (200 − 4×4×1)", v)
	}
}

// TestRestRejectsBadDepth: a non-positive depth is a clean error.
func TestRestRejectsBadDepth(t *testing.T) {
	fs := embossedBlock(t)
	rs := squareOn(planeAtZ(2), 4, 3)
	pad := NewPlasticFeatures(fs).AddRest(rs, []int{0}, func() float64 { return 0 }, false, 0)
	fs.Recompute()
	if pad.Health().OK() {
		t.Error("a zero-depth rest should be sick")
	}
}
