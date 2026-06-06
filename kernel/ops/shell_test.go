// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati/kernel/ops"
	"oblikovati/kernel/subd"
	"oblikovati/kernel/topo"
)

// shellBox builds an axis-aligned box [0,sx]×[0,sy]×[0,sz].
func shellBox(sx, sy, sz float64) *topo.Body {
	return subd.ToBody(subd.Box(sx, sy, sz), "box")
}

// topFaceKey returns the reference key of the +Z (top) face.
func topFaceKey(t *testing.T, b *topo.Body) []byte {
	t.Helper()
	for _, f := range b.Faces() {
		if f.Geometry().NormalAt(0, 0).Z > 0.99 {
			return f.ReferenceKey()
		}
	}
	t.Fatal("no +Z face found")
	return nil
}

// TestShellOpenTopBox shells a 4×4×4 box with the top removed to wall thickness 0.5: a
// 5-wall open cup. Outer 64 minus the open cavity [0.5,3.5]×[0.5,3.5]×[0.5,4] (3·3·3.5 =
// 31.5) = 32.5, and the result must be a valid solid.
func TestShellOpenTopBox(t *testing.T) {
	box := shellBox(4, 4, 4)
	res, err := ops.Shell(box, [][]byte{topFaceKey(t, box)}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("shelled box not a valid solid: %+v", r)
	}
	want := 64.0 - 3*3*3.5
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("shell volume = %g, want %g", got, want)
	}
}

// TestShellLostFaceErrors checks a vanished removed-face key is reported (so the feature
// can go Sick) rather than silently shelling a closed box.
func TestShellLostFaceErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.Shell(box, [][]byte{[]byte("ghost")}, 0.2); err == nil {
		t.Error("shell with a lost face key should error")
	}
}

// TestShellThicknessMustBePositive guards the thickness.
func TestShellThicknessMustBePositive(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.Shell(box, [][]byte{topFaceKey(t, box)}, 0); err == nil {
		t.Error("zero thickness should error")
	}
}
