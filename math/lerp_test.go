// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

// The endpoint-exactness property is the point of the t==1 pin: with the raw
// fused form, Lerp(0.1, 0.3, 1) = 0.30000000000000004 ≠ 0.3.
func TestLerpEndpointsExact(t *testing.T) {
	pairs := []struct{ a, b Scalar }{
		{0, 1}, {0.1, 0.3}, {-2.5, 7.25}, {1e-9, 1e9}, {3, 3},
	}
	for _, c := range pairs {
		if got := Lerp(c.a, c.b, 0); got != c.a {
			t.Errorf("Lerp(%v, %v, 0) = %v, want exactly a", c.a, c.b, got)
		}
		if got := Lerp(c.a, c.b, 1); got != c.b {
			t.Errorf("Lerp(%v, %v, 1) = %v, want exactly b", c.a, c.b, got)
		}
	}
}

func TestLerpInteriorAndExtrapolation(t *testing.T) {
	if got := Lerp(2, 6, 0.5); got != 4 {
		t.Errorf("Lerp(2, 6, 0.5) = %v, want 4", got)
	}
	if got := Lerp(2, 6, 0.25); got != 3 {
		t.Errorf("Lerp(2, 6, 0.25) = %v, want 3", got)
	}
	// t outside [0,1] extrapolates along the same line — no clamping.
	if got := Lerp(2, 6, 2); got != 10 {
		t.Errorf("Lerp(2, 6, 2) = %v, want 10", got)
	}
	if got := Lerp(2, 6, -0.5); got != 0 {
		t.Errorf("Lerp(2, 6, -0.5) = %v, want 0", got)
	}
}

func TestPoint2LerpEndpointsAndMidpoint(t *testing.T) {
	a, b := P2(0.1, -1), P2(0.3, 3)
	if got := a.Lerp(b, 0); got != a {
		t.Errorf("a.Lerp(b, 0) = %v, want exactly %v", got, a)
	}
	if got := a.Lerp(b, 1); got != b {
		t.Errorf("a.Lerp(b, 1) = %v, want exactly %v", got, b)
	}
	if got := P2(0, 2).Lerp(P2(4, 6), 0.5); got != P2(2, 4) {
		t.Errorf("midpoint = %v, want (2, 4)", got)
	}
}

func TestPoint3LerpEndpointsAndMidpoint(t *testing.T) {
	a, b := P3(0.1, -1, 5), P3(0.3, 3, -5)
	if got := a.Lerp(b, 0); got != a {
		t.Errorf("a.Lerp(b, 0) = %v, want exactly %v", got, a)
	}
	if got := a.Lerp(b, 1); got != b {
		t.Errorf("a.Lerp(b, 1) = %v, want exactly %v", got, b)
	}
	if got := P3(0, 2, -2).Lerp(P3(4, 6, 2), 0.5); got != P3(2, 4, 0) {
		t.Errorf("midpoint = %v, want (2, 4, 0)", got)
	}
}
