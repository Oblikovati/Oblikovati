// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/math"
)

// TestMoveFaceGrowsBox moves the top face of a 2×2×2 box up by 1: the solid grows to a
// 2×2×3 box (vol 12) and stays a valid manifold.
func TestMoveFaceGrowsBox(t *testing.T) {
	box := shellBox(2, 2, 2)
	res, err := ops.MoveFaces(box, [][]byte{topFaceKey(t, box)}, math.V3(0, 0, 1))
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("moved-face body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-12) > 1e-6 {
		t.Errorf("move-face volume = %g, want 12", got)
	}
}

// TestOffsetFaceShavesBox offsets the top face of a 2×2×2 box inward (−0.5 along its +Z
// normal): the box shrinks to 2×2×1.5 (vol 6).
func TestOffsetFaceShavesBox(t *testing.T) {
	box := shellBox(2, 2, 2)
	res, err := ops.OffsetFaces(box, [][]byte{topFaceKey(t, box)}, -0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("offset-face body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-6) > 1e-6 {
		t.Errorf("offset-face volume = %g, want 6", got)
	}
}

// TestMoveFaceLostKeyErrors reports a vanished face key so the feature can go Sick.
func TestMoveFaceLostKeyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.MoveFaces(box, [][]byte{[]byte("ghost")}, math.V3(0, 0, 1)); err == nil {
		t.Error("move-face with a lost key should error")
	}
}
