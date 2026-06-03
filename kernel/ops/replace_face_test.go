// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/kernel/geom"
	"github.com/Oblikovati/oblikovati/kernel/ops"
	"github.com/Oblikovati/oblikovati/math"
)

// TestReplaceFaceParallelGrows replaces the top face of a 2×2×2 box (z=2) with a parallel
// plane at z=3: the box grows to 2×2×3 (vol 12), a valid solid.
func TestReplaceFaceParallelGrows(t *testing.T) {
	box := shellBox(2, 2, 2)
	target, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	res, err := ops.ReplaceFaces(box, [][]byte{topFaceKey(t, box)}, target)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("replaced body not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-12) > 1e-6 {
		t.Errorf("replace-face volume = %g, want 12", got)
	}
}

// TestReplaceFaceTiltedIsValid replaces the top face with a tilted plane and checks the
// result is still a valid solid (a wedge top).
func TestReplaceFaceTiltedIsValid(t *testing.T) {
	box := shellBox(2, 2, 2)
	target, _ := geom.NewPlane(math.P3(0, 0, 2), math.V3(0, 1, 4)) // tilts about X through z=2
	res, err := ops.ReplaceFaces(box, [][]byte{topFaceKey(t, box)}, target)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("tilted replace not a valid solid: %+v", r)
	}
}

// TestReplaceFaceLostKeyErrors reports a vanished face key so the feature can go Sick.
func TestReplaceFaceLostKeyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	target, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	if _, err := ops.ReplaceFaces(box, [][]byte{[]byte("ghost")}, target); err == nil {
		t.Error("replace-face with a lost key should error")
	}
}
