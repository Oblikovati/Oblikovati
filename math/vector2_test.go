// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestVector2Arithmetic(t *testing.T) {
	a := V2(1, 2)
	b := V2(3, -4)
	if got := a.Add(b); got != (Vector2{4, -2}) {
		t.Errorf("Add = %v, want {4 -2}", got)
	}
	if got := a.Sub(b); got != (Vector2{-2, 6}) {
		t.Errorf("Sub = %v, want {-2 6}", got)
	}
	if got := a.Scale(3); got != (Vector2{3, 6}) {
		t.Errorf("Scale = %v, want {3 6}", got)
	}
}

func TestVector2Cross(t *testing.T) {
	// +X cross +Y is +1 (counter-clockwise).
	if got := V2(1, 0).Cross(V2(0, 1)); got != 1 {
		t.Errorf("X×Y = %v, want 1", got)
	}
	if got := V2(0, 1).Cross(V2(1, 0)); got != -1 {
		t.Errorf("Y×X = %v, want -1", got)
	}
}

func TestVector2Length(t *testing.T) {
	v := V2(3, 4)
	if got := v.Length(); got != 5 {
		t.Errorf("Length = %v, want 5", got)
	}
}

func TestVector2AngleParallelPerp(t *testing.T) {
	a := V2(1, 0)
	if got := a.AngleTo(V2(0, 1)); !approxEqual(got, stdmath.Pi/2, 1e-12) {
		t.Errorf("AngleTo = %v, want π/2", got)
	}
	if !a.IsParallelTo(V2(-2, 0), 0) {
		t.Error("(1,0) should be parallel to (-2,0)")
	}
	if !a.IsPerpendicularTo(V2(0, 5), 0) {
		t.Error("(1,0) should be perpendicular to (0,5)")
	}
}
