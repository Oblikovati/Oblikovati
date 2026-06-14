// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func mustUnit(t *testing.T, x, y, z Scalar) UnitVector3 {
	t.Helper()
	u, err := NewUnitVector3(x, y, z)
	if err != nil {
		t.Fatalf("NewUnitVector3(%v,%v,%v): %v", x, y, z, err)
	}
	return u
}

func TestQuaternionFromAxisAngleIsUnit(t *testing.T) {
	q := QuaternionFromAxisAngle(mustUnit(t, 0, 0, 1), stdmath.Pi/3)
	if !approxEqual(q.Length(), 1, 1e-12) {
		t.Errorf("Length() = %v, want 1", q.Length())
	}
}

// TestQuaternionMatrixRotatesVector checks a 90° rotation about +Z maps +X onto +Y.
func TestQuaternionMatrixRotatesVector(t *testing.T) {
	q := QuaternionFromAxisAngle(mustUnit(t, 0, 0, 1), stdmath.Pi/2)
	got := q.Matrix4().TransformVector(V3(1, 0, 0))
	if !got.IsEqualTo(V3(0, 1, 0), 1e-9) {
		t.Errorf("rotated +X = %+v, want (0,1,0)", got)
	}
}

// TestQuaternionMatrixRoundTrip checks Matrix4 → QuaternionFromMatrix → Matrix4 is the
// identity over a spread of axes and angles (the warm-start path's correctness).
func TestQuaternionMatrixRoundTrip(t *testing.T) {
	axes := []UnitVector3{
		mustUnit(t, 1, 0, 0), mustUnit(t, 0, 1, 0), mustUnit(t, 0, 0, 1),
		mustUnit(t, 1, 1, 0), mustUnit(t, 1, 2, 3), mustUnit(t, -2, 1, -1),
	}
	angles := []Scalar{0.1, stdmath.Pi / 4, stdmath.Pi / 2, 2.5, stdmath.Pi - 0.01}
	for _, axis := range axes {
		for _, angle := range angles {
			q := QuaternionFromAxisAngle(axis, angle)
			back := QuaternionFromMatrix(q.Matrix4())
			if !back.IsEqualTo(q, 1e-9) {
				t.Errorf("round trip axis=%+v angle=%v: got %+v want %+v", axis, angle, back, q)
			}
		}
	}
}

func TestQuaternionNormalizeZeroIsIdentity(t *testing.T) {
	if got := (Quaternion{}).Normalize(); got != QuaternionIdentity() {
		t.Errorf("zero quaternion normalized = %+v, want identity", got)
	}
}

// TestQuaternionMulComposesRotations checks two 90° Z-rotations compose to a 180° one.
func TestQuaternionMulComposesRotations(t *testing.T) {
	q90 := QuaternionFromAxisAngle(mustUnit(t, 0, 0, 1), stdmath.Pi/2)
	q180 := QuaternionFromAxisAngle(mustUnit(t, 0, 0, 1), stdmath.Pi)
	if !q90.Mul(q90).IsEqualTo(q180, 1e-9) {
		t.Errorf("q90·q90 = %+v, want q180 %+v", q90.Mul(q90), q180)
	}
}

func TestQuaternionMatrixIsRigid(t *testing.T) {
	q := QuaternionFromAxisAngle(mustUnit(t, 1, 2, 3), 1.2)
	if !q.Matrix4().IsRigid(1e-9) {
		t.Error("quaternion rotation matrix is not rigid")
	}
}
