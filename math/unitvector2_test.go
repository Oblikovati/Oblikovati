// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

func TestNewUnitVector2Normalizes(t *testing.T) {
	u, err := NewUnitVector2(0, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.AsVector() != (Vector2{0, 1}) {
		t.Errorf("normalized = %v, want {0 1}", u.AsVector())
	}
}

func TestNewUnitVector2ZeroLengthFails(t *testing.T) {
	if _, err := NewUnitVector2(0, 0); err == nil {
		t.Fatal("expected error normalizing zero vector")
	}
}

func TestUnitVector2DotAndPerp(t *testing.T) {
	x := mustUnit2(t, 1, 0)
	y := mustUnit2(t, 0, 1)
	if got := x.Dot(y); got != 0 {
		t.Errorf("x·y = %v, want 0", got)
	}
	if !x.IsPerpendicularTo(y, 0) {
		t.Error("+X should be perpendicular to +Y")
	}
	if !x.IsParallelTo(x.Negate(), 0) {
		t.Error("unit vector should be parallel to its negation")
	}
}

func mustUnit2(t *testing.T, x, y Scalar) UnitVector2 {
	t.Helper()
	u, err := NewUnitVector2(x, y)
	if err != nil {
		t.Fatalf("NewUnitVector2(%g,%g): %v", x, y, err)
	}
	return u
}
