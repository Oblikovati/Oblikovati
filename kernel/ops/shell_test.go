// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/test-utilities/brepfixture"

	"oblikovati.org/kernel/ops"
	"oblikovati.org/kernel/ops/query"
	"oblikovati.org/kernel/topo"
)

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
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 4, 4, 4)
	res, err := ops.Shell(box, [][]byte{topFaceKey(t, box)}, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("shelled box not a valid solid: %+v", r)
	}
	want := 64.0 - 3*3*3.5
	if got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("shell volume = %g, want %g", got, want)
	}
}

// TestShellOutsideBox shells the same 4×4×4 open-top box OUTWARD by 0.5: the wall grows onto the
// outside, so the outer solid [-0.5,4.5]×[-0.5,4.5]×[-0.5,4] (5·5·4.5 = 112.5) minus the original
// 64 leaves a 48.5 wall, and the result must be a valid solid.
func TestShellOutsideBox(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 4, 4, 4)
	res, err := ops.ShellDirected(box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellOutside)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("outward-shelled box not a valid solid: %+v", r)
	}
	want := 5.0*5.0*4.5 - 64.0
	if got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("outside shell volume = %g, want %g", got, want)
	}
}

// TestShellBothBox centres a 0.5 wall on the faces of the 4×4×4 open-top box: outer offset +0.25
// (4.5·4.5·4.25 = 86.0625) minus inner offset −0.25 (3.5·3.5·3.75 = 45.9375) = 40.125.
func TestShellBothBox(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 4, 4, 4)
	res, err := ops.ShellDirected(box, [][]byte{topFaceKey(t, box)}, 0.5, ops.ShellBoth)
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("both-sides-shelled box not a valid solid: %+v", r)
	}
	want := 4.5*4.5*4.25 - 3.5*3.5*3.75
	if got := query.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-want) > 1e-6 {
		t.Errorf("both-sides shell volume = %g, want %g", got, want)
	}
}

// TestShellDirectedUnknownDirection guards the direction switch.
func TestShellDirectedUnknownDirection(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	if _, err := ops.ShellDirected(box, [][]byte{topFaceKey(t, box)}, 0.2, ops.ShellDirection(9)); err == nil {
		t.Error("unknown shell direction should error")
	}
}

// TestShellLostFaceErrors checks a vanished removed-face key is reported (so the feature
// can go Sick) rather than silently shelling a closed box.
func TestShellLostFaceErrors(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	if _, err := ops.Shell(box, [][]byte{[]byte("ghost")}, 0.2); err == nil {
		t.Error("shell with a lost face key should error")
	}
}

// TestShellThicknessMustBePositive guards the thickness.
func TestShellThicknessMustBePositive(t *testing.T) {
	t.Parallel()
	box := brepfixture.Box(math.P3(0, 0, 0), 2, 2, 2)
	if _, err := ops.Shell(box, [][]byte{topFaceKey(t, box)}, 0); err == nil {
		t.Error("zero thickness should error")
	}
}
