// SPDX-License-Identifier: GPL-2.0-only

package math

import (
	stdmath "math"
	"testing"
)

func TestVector3Arithmetic(t *testing.T) {
	a := V3(1, 2, 3)
	b := V3(4, -5, 6)
	if got := a.Add(b); got != (Vector3{5, -3, 9}) {
		t.Errorf("Add = %v, want {5 -3 9}", got)
	}
	if got := a.Sub(b); got != (Vector3{-3, 7, -3}) {
		t.Errorf("Sub = %v, want {-3 7 -3}", got)
	}
	if got := a.Scale(2); got != (Vector3{2, 4, 6}) {
		t.Errorf("Scale = %v, want {2 4 6}", got)
	}
	if got := a.Negate(); got != (Vector3{-1, -2, -3}) {
		t.Errorf("Negate = %v, want {-1 -2 -3}", got)
	}
}

func TestVector3DotCross(t *testing.T) {
	x := V3(1, 0, 0)
	y := V3(0, 1, 0)
	if got := x.Dot(y); got != 0 {
		t.Errorf("x·y = %v, want 0", got)
	}
	// Right-handed: x × y = z.
	if got := x.Cross(y); got != (Vector3{0, 0, 1}) {
		t.Errorf("x×y = %v, want {0 0 1}", got)
	}
	if got := y.Cross(x); got != (Vector3{0, 0, -1}) {
		t.Errorf("y×x = %v, want {0 0 -1}", got)
	}
}

func TestVector3Length(t *testing.T) {
	v := V3(3, 4, 0)
	if got := v.LengthSquared(); got != 25 {
		t.Errorf("LengthSquared = %v, want 25", got)
	}
	if got := v.Length(); got != 5 {
		t.Errorf("Length = %v, want 5", got)
	}
}

func TestVector3AngleTo(t *testing.T) {
	cases := []struct {
		a, b Vector3
		want Scalar
	}{
		{V3(1, 0, 0), V3(0, 1, 0), stdmath.Pi / 2},
		{V3(1, 0, 0), V3(1, 0, 0), 0},
		{V3(1, 0, 0), V3(-1, 0, 0), stdmath.Pi},
		{V3(0, 0, 0), V3(1, 0, 0), 0}, // zero-length → 0, no NaN
	}
	for _, c := range cases {
		if got := c.a.AngleTo(c.b); !approxEqual(got, c.want, 1e-12) {
			t.Errorf("AngleTo(%v,%v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestVector3ParallelPerpendicular(t *testing.T) {
	a := V3(1, 1, 0)
	if !a.IsParallelTo(V3(2, 2, 0), 0) {
		t.Error("(1,1,0) should be parallel to (2,2,0)")
	}
	if !a.IsParallelTo(V3(-3, -3, 0), 0) { // opposite direction still parallel
		t.Error("(1,1,0) should be parallel to (-3,-3,0)")
	}
	if a.IsParallelTo(V3(1, 0, 0), 0) {
		t.Error("(1,1,0) should not be parallel to (1,0,0)")
	}
	if !a.IsPerpendicularTo(V3(1, -1, 0), 0) {
		t.Error("(1,1,0) should be perpendicular to (1,-1,0)")
	}
	if a.IsPerpendicularTo(V3(1, 1, 0), 0) {
		t.Error("(1,1,0) should not be perpendicular to itself")
	}
}

func TestVector3PointConversion(t *testing.T) {
	v := V3(1, 2, 3)
	if got := v.AsPoint(); got != (Point3{1, 2, 3}) {
		t.Errorf("AsPoint = %v, want {1 2 3}", got)
	}
}

func TestVector3IsEqualTo(t *testing.T) {
	a := V3(1, 2, 3)
	if !a.IsEqualTo(V3(1, 2, 3+1e-12), 0) {
		t.Error("vectors within default tolerance should be equal")
	}
	if a.IsEqualTo(V3(1, 2, 3.1), 0) {
		t.Error("vectors beyond tolerance should not be equal")
	}
}
