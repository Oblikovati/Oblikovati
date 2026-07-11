// SPDX-License-Identifier: GPL-2.0-only

package ops_test

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/kernel/ops"
	"oblikovati.org/math"
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

// TestReplaceFacesMultiPicksNearest replaces the box's top face against two candidate target
// planes (z=3 near, z=100 far): the top face binds to the nearer plane, growing the box to vol 12
// (#1886).
func TestReplaceFacesMultiPicksNearest(t *testing.T) {
	box := shellBox(2, 2, 2)
	near, _ := geom.NewPlane(math.P3(0, 0, 3), math.V3(0, 0, 1))
	far, _ := geom.NewPlane(math.P3(0, 0, 100), math.V3(0, 0, 1))
	res, err := ops.ReplaceFacesMulti(box, [][]byte{topFaceKey(t, box)}, []geom.Plane{far, near})
	if err != nil {
		t.Fatal(err)
	}
	if r := ops.Validate(res); !r.Valid || !res.IsSolid() {
		t.Fatalf("multi replace not a valid solid: %+v", r)
	}
	if got := ops.BodyGeometryProperties(res, ops.DefaultQuality()).Volume; stdmath.Abs(got-12) > 1e-6 {
		t.Errorf("multi replace volume = %g, want 12 (bound to nearer z=3 plane)", got)
	}
}

// TestReplaceFacesMultiEmptyErrors: no target planes is a caller error (feature goes Sick).
func TestReplaceFacesMultiEmptyErrors(t *testing.T) {
	box := shellBox(2, 2, 2)
	if _, err := ops.ReplaceFacesMulti(box, [][]byte{topFaceKey(t, box)}, nil); err == nil {
		t.Error("ReplaceFacesMulti with no targets should error")
	}
}

// TestPlaneOfFace resolves a planar face's plane by reference key, and reports a miss for an
// unknown key.
func TestPlaneOfFace(t *testing.T) {
	box := shellBox(2, 2, 2)
	pl, ok := ops.PlaneOfFace(box, topFaceKey(t, box))
	if !ok {
		t.Fatal("PlaneOfFace should resolve the +Z face")
	}
	if n := pl.Normal(); stdmath.Abs(n.Z-1) > 1e-9 {
		t.Errorf("+Z face normal = %v, want ≈ (0,0,1)", n)
	}
	if _, ok := ops.PlaneOfFace(box, []byte("ghost")); ok {
		t.Error("PlaneOfFace should miss on an unknown key")
	}
}
