// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestNewUnitVector3Normalizes(t *testing.T) {
	u, err := NewUnitVector3(0, 0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.AsVector() != (Vector3{0, 0, 1}) {
		t.Errorf("normalized = %v, want {0 0 1}", u.AsVector())
	}
	if l := u.AsVector().Length(); !approxEqual(l, 1, 1e-15) {
		t.Errorf("unit length = %v, want 1", l)
	}
}

func TestNewUnitVector3ZeroLengthFails(t *testing.T) {
	_, err := NewUnitVector3(0, 0, 0)
	if err == nil {
		t.Fatal("expected error normalizing zero vector")
	}
}

func TestUnitVector3FromVector(t *testing.T) {
	u, err := UnitVector3FromVector(V3(3, 0, 4))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !u.IsEqualTo(mustUnit3(t, 0.6, 0, 0.8), 1e-15) {
		t.Errorf("normalized = %v, want (0.6,0,0.8)", u.AsVector())
	}
}

func TestUnitVector3AngleAndDot(t *testing.T) {
	x := mustUnit3(t, 1, 0, 0)
	y := mustUnit3(t, 0, 1, 0)
	if got := x.Dot(y); got != 0 {
		t.Errorf("x·y = %v, want 0", got)
	}
	if got := x.AngleTo(y); !approxEqual(got, stdmath.Pi/2, 1e-12) {
		t.Errorf("AngleTo = %v, want π/2", got)
	}
	if got := x.Cross(y); got != (Vector3{0, 0, 1}) {
		t.Errorf("x×y = %v, want {0 0 1}", got)
	}
}

func TestUnitVector3ParallelPerpendicular(t *testing.T) {
	x := mustUnit3(t, 1, 0, 0)
	if !x.IsParallelTo(x.Negate(), 0) {
		t.Error("a unit vector should be parallel to its negation")
	}
	if !x.IsPerpendicularTo(mustUnit3(t, 0, 1, 0), 0) {
		t.Error("+X should be perpendicular to +Y")
	}
}

func mustUnit3(t *testing.T, x, y, z Scalar) UnitVector3 {
	t.Helper()
	u, err := NewUnitVector3(x, y, z)
	if err != nil {
		t.Fatalf("NewUnitVector3(%g,%g,%g): %v", x, y, z, err)
	}
	return u
}
