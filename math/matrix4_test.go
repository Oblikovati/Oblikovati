// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestMatrix4Identity(t *testing.T) {
	p := P3(3, -2, 7)
	if got := Identity4().TransformPoint(p); got != p {
		t.Errorf("identity moved point: %v != %v", got, p)
	}
}

func TestMatrix4FromCellsRoundTrip(t *testing.T) {
	m := Rotation4(0.7, V3(0, 0, 1).AsUnit(), P3(1, 2, 3)).Mul(Translation4(V3(4, -5, 6)))
	if got := Matrix4FromCells(m.Cells()); !got.IsEqualTo(m, 1e-12) {
		t.Errorf("Matrix4FromCells(m.Cells()) != m:\n%v\n%v", got.Cells(), m.Cells())
	}
}

func TestMatrix4Translation(t *testing.T) {
	m := Translation4(V3(1, 2, 3))
	if got := m.TransformPoint(P3(0, 0, 0)); got != (Point3{1, 2, 3}) {
		t.Errorf("TransformPoint = %v, want {1 2 3}", got)
	}
	// A vector is unaffected by translation.
	if got := m.TransformVector(V3(5, 5, 5)); got != (Vector3{5, 5, 5}) {
		t.Errorf("translation leaked into vector: %v", got)
	}
}

func TestMatrix4RotationZ90(t *testing.T) {
	z := mustUnit3(t, 0, 0, 1)
	m := Rotation4(stdmath.Pi/2, z, P3(0, 0, 0))
	// +X rotates to +Y about Z.
	got := m.TransformPoint(P3(1, 0, 0))
	if !got.IsEqualTo(P3(0, 1, 0), 1e-12) {
		t.Errorf("rotateZ90(X) = %v, want {0 1 0}", got)
	}
}

func TestMatrix4RotationAboutCenterIsFixed(t *testing.T) {
	z := mustUnit3(t, 0, 0, 1)
	center := P3(5, 5, 0)
	m := Rotation4(stdmath.Pi/3, z, center)
	if got := m.TransformPoint(center); !got.IsEqualTo(center, 1e-12) {
		t.Errorf("rotation moved its own center: %v != %v", got, center)
	}
}

func TestMatrix4Compose(t *testing.T) {
	tr := Translation4(V3(10, 0, 0))
	rot := Rotation4(stdmath.Pi/2, mustUnit3(t, 0, 0, 1), P3(0, 0, 0))
	// Mul(tr, rot) rotates first, then translates.
	got := tr.Mul(rot).TransformPoint(P3(1, 0, 0))
	if !got.IsEqualTo(P3(10, 1, 0), 1e-12) {
		t.Errorf("composed = %v, want {10 1 0}", got)
	}
}

func TestMatrix4InverseRoundTrip(t *testing.T) {
	m := Rotation4(0.7, mustUnit3(t, 1, 2, 3), P3(1, 1, 1)).
		Mul(Translation4(V3(4, -5, 6))).
		Mul(Scale4(2, 3, 0.5))
	inv, ok := m.Inverse()
	if !ok {
		t.Fatal("matrix should be invertible")
	}
	// m∘m⁻¹ = identity within tolerance.
	if got := m.Mul(inv); !got.IsEqualTo(Identity4(), 1e-9) {
		t.Errorf("m·m⁻¹ =\n%v\nwant identity", got.Cells())
	}
	// And inverse actually undoes a point transform.
	p := P3(7, 8, 9)
	round := inv.TransformPoint(m.TransformPoint(p))
	if !round.IsEqualTo(p, 1e-9) {
		t.Errorf("round-trip point = %v, want %v", round, p)
	}
}

func TestMatrix4SingularInverseFails(t *testing.T) {
	if _, ok := Scale4(1, 0, 1).Inverse(); ok {
		t.Error("degenerate scale should not be invertible")
	}
}

func TestMatrix4Determinant(t *testing.T) {
	if got := Scale4(2, 3, 4).Determinant(); !approxEqual(got, 24, 1e-12) {
		t.Errorf("det = %v, want 24", got)
	}
	if got := Rotation4(0.9, mustUnit3(t, 0, 1, 0), P3(0, 0, 0)).Determinant(); !approxEqual(got, 1, 1e-12) {
		t.Errorf("rotation det = %v, want 1", got)
	}
}

func TestMatrix4IsRigid(t *testing.T) {
	rigid := Rotation4(0.5, mustUnit3(t, 0, 0, 1), P3(1, 2, 3)).Mul(Translation4(V3(9, 9, 9)))
	if !rigid.IsRigid(1e-12) {
		t.Error("rotation∘translation should be rigid")
	}
	if Scale4(2, 2, 2).IsRigid(1e-12) {
		t.Error("uniform scale ≠ rigid")
	}
}

func TestMatrix4CoordinateSystem(t *testing.T) {
	// A frame at (1,0,0) with X→+Y, Y→−X, Z→+Z (a 90° turn about Z).
	m := CoordinateSystem4(P3(1, 0, 0), V3(0, 1, 0), V3(-1, 0, 0), V3(0, 0, 1))
	if got := m.TransformPoint(P3(0, 0, 0)); !got.IsEqualTo(P3(1, 0, 0), 1e-12) {
		t.Errorf("origin maps to %v, want {1 0 0}", got)
	}
	if got := m.TransformVector(V3(1, 0, 0)); !got.IsEqualTo(V3(0, 1, 0), 1e-12) {
		t.Errorf("local X maps to %v, want {0 1 0}", got)
	}
}

func TestMatrix4TransformUnitVector(t *testing.T) {
	m := Scale4(10, 10, 10)
	u, err := m.TransformUnitVector(mustUnit3(t, 1, 0, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Result stays unit length despite the scale.
	if l := u.AsVector().Length(); !approxEqual(l, 1, 1e-12) {
		t.Errorf("transformed unit length = %v, want 1", l)
	}
}

func TestRotateBetween(t *testing.T) {
	from := mustUnit3(t, 1, 0, 0)
	to := mustUnit3(t, 0, 1, 0)
	got := RotateBetween(from, to).TransformVector(from.AsVector())
	if !got.IsEqualTo(to.AsVector(), 1e-12) {
		t.Errorf("RotateBetween mapped from to %v, want %v", got, to.AsVector())
	}
}

func TestRotateBetweenAntiparallel(t *testing.T) {
	from := mustUnit3(t, 1, 0, 0)
	to := from.Negate()
	got := RotateBetween(from, to).TransformVector(from.AsVector())
	if !got.IsEqualTo(to.AsVector(), 1e-9) {
		t.Errorf("antiparallel RotateBetween = %v, want %v", got, to.AsVector())
	}
}
