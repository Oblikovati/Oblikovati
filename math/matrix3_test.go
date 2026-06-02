// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestMatrix3Identity(t *testing.T) {
	p := P2(3, -2)
	if got := Identity3().TransformPoint(p); got != p {
		t.Errorf("identity moved point: %v != %v", got, p)
	}
}

func TestMatrix3Translation(t *testing.T) {
	m := Translation3(V2(1, 2))
	if got := m.TransformPoint(P2(0, 0)); got != (Point2{1, 2}) {
		t.Errorf("TransformPoint = %v, want {1 2}", got)
	}
	if got := m.TransformVector(V2(5, 5)); got != (Vector2{5, 5}) {
		t.Errorf("translation leaked into vector: %v", got)
	}
}

func TestMatrix3Rotation90(t *testing.T) {
	m := Rotation3(stdmath.Pi/2, P2(0, 0))
	got := m.TransformPoint(P2(1, 0))
	if !got.IsEqualTo(P2(0, 1), 1e-12) {
		t.Errorf("rotate90(X) = %v, want {0 1}", got)
	}
}

func TestMatrix3RotationCenterFixed(t *testing.T) {
	center := P2(4, 7)
	m := Rotation3(stdmath.Pi/5, center)
	if got := m.TransformPoint(center); !got.IsEqualTo(center, 1e-12) {
		t.Errorf("rotation moved its own center: %v", got)
	}
}

func TestMatrix3Compose(t *testing.T) {
	tr := Translation3(V2(10, 0))
	rot := Rotation3(stdmath.Pi/2, P2(0, 0))
	got := tr.Mul(rot).TransformPoint(P2(1, 0))
	if !got.IsEqualTo(P2(10, 1), 1e-12) {
		t.Errorf("composed = %v, want {10 1}", got)
	}
}

func TestMatrix3InverseRoundTrip(t *testing.T) {
	m := Rotation3(0.6, P2(2, 2)).Mul(Translation3(V2(3, -4))).Mul(Scale3(2, 5))
	inv, ok := m.Inverse()
	if !ok {
		t.Fatal("matrix should be invertible")
	}
	if got := m.Mul(inv); !got.IsEqualTo(Identity3(), 1e-9) {
		t.Errorf("m·m⁻¹ = %v, want identity", got.Cells())
	}
	p := P2(7, 8)
	if round := inv.TransformPoint(m.TransformPoint(p)); !round.IsEqualTo(p, 1e-9) {
		t.Errorf("round-trip = %v, want %v", round, p)
	}
}

func TestMatrix3SingularFails(t *testing.T) {
	if _, ok := Scale3(0, 1).Inverse(); ok {
		t.Error("degenerate scale should not be invertible")
	}
}

func TestMatrix3IsRigid(t *testing.T) {
	if !Rotation3(0.4, P2(1, 1)).Mul(Translation3(V2(2, 3))).IsRigid(1e-12) {
		t.Error("rotation∘translation should be rigid")
	}
	if Scale3(2, 2).IsRigid(1e-12) {
		t.Error("scale ≠ rigid")
	}
}

func TestMatrix3Determinant(t *testing.T) {
	if got := Scale3(2, 3).Determinant(); !approxEqual(got, 6, 1e-12) {
		t.Errorf("det = %v, want 6", got)
	}
}
