// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

func TestNewBoxNormalizesCorners(t *testing.T) {
	b := NewBox(P3(3, 0, 5), P3(-1, 2, 1)) // deliberately unordered
	if b.Min != (Point3{-1, 0, 1}) || b.Max != (Point3{3, 2, 5}) {
		t.Errorf("box = %v..%v, want {-1 0 1}..{3 2 5}", b.Min, b.Max)
	}
}

func TestBoxContainsAndIntersect(t *testing.T) {
	b := NewBox(P3(0, 0, 0), P3(2, 2, 2))
	if !b.Contains(P3(1, 1, 1)) {
		t.Error("interior point should be contained")
	}
	if b.Contains(P3(3, 1, 1)) {
		t.Error("exterior point should not be contained")
	}
	if !b.Intersects(NewBox(P3(1, 1, 1), P3(3, 3, 3))) {
		t.Error("overlapping boxes should intersect")
	}
	if b.Intersects(NewBox(P3(5, 5, 5), P3(6, 6, 6))) {
		t.Error("disjoint boxes should not intersect")
	}
	if !b.ContainsBox(NewBox(P3(0.5, 0.5, 0.5), P3(1, 1, 1))) {
		t.Error("inner box should be contained")
	}
}

func TestBoxFromPointsAndVolume(t *testing.T) {
	b := BoxFromPoints(P3(0, 0, 0), P3(2, 3, 4), P3(-1, 1, 1))
	if b.Min != (Point3{-1, 0, 0}) || b.Max != (Point3{2, 3, 4}) {
		t.Errorf("bounds = %v..%v", b.Min, b.Max)
	}
	if got := b.Volume(); got != 3*3*4 {
		t.Errorf("Volume = %v, want 36", got)
	}
	if got := b.Center(); got != (Point3{0.5, 1.5, 2}) {
		t.Errorf("Center = %v, want {0.5 1.5 2}", got)
	}
}

func TestEmptyBox(t *testing.T) {
	e := EmptyBox()
	if !e.IsEmpty() {
		t.Error("EmptyBox should be empty")
	}
	if e.Volume() != 0 {
		t.Error("empty box volume should be 0")
	}
	// Extending the empty box with one point yields a degenerate box there.
	g := e.ExtendPoint(P3(1, 2, 3))
	if g.Min != (Point3{1, 2, 3}) || g.Max != (Point3{1, 2, 3}) {
		t.Errorf("extended empty box = %v..%v, want point {1 2 3}", g.Min, g.Max)
	}
}

func TestBoxUnionAndCorners(t *testing.T) {
	u := NewBox(P3(0, 0, 0), P3(1, 1, 1)).Union(NewBox(P3(2, 2, 2), P3(3, 3, 3)))
	if u.Min != (Point3{0, 0, 0}) || u.Max != (Point3{3, 3, 3}) {
		t.Errorf("union = %v..%v, want {0 0 0}..{3 3 3}", u.Min, u.Max)
	}
	corners := NewBox(P3(0, 0, 0), P3(1, 1, 1)).Corners()
	if corners[0] != (Point3{0, 0, 0}) || corners[7] != (Point3{1, 1, 1}) {
		t.Errorf("corners[0]=%v corners[7]=%v", corners[0], corners[7])
	}
}

func TestBoxTransform(t *testing.T) {
	// Pure translation shifts both corners by the same vector.
	moved := NewBox(P3(0, 0, 0), P3(1, 1, 1)).Transform(Translation4(V3(10, 20, 30)))
	if moved.Min != (Point3{10, 20, 30}) || moved.Max != (Point3{11, 21, 31}) {
		t.Errorf("translated = %v..%v, want {10 20 30}..{11 21 31}", moved.Min, moved.Max)
	}
	// A 90° rotation about Z maps (x,y)→(-y,x): the AABB of [0,2]×[0,1]×[0,1]
	// grows to [-1,0]×[0,2]×[0,1] (a rotated box's tight AABB is larger).
	rotZ := Matrix4FromCells([16]Scalar{0, -1, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1})
	rotated := NewBox(P3(0, 0, 0), P3(2, 1, 1)).Transform(rotZ)
	if rotated.Min != (Point3{-1, 0, 0}) || rotated.Max != (Point3{0, 2, 1}) {
		t.Errorf("rotated = %v..%v, want {-1 0 0}..{0 2 1}", rotated.Min, rotated.Max)
	}
	// The empty box has no corners to place: it stays empty.
	if got := EmptyBox().Transform(Translation4(V3(5, 5, 5))); !got.IsEmpty() {
		t.Errorf("empty.Transform = %v..%v, want empty", got.Min, got.Max)
	}
}

func TestBoxFarthestPoint(t *testing.T) {
	b := NewBox(P3(-1, -2, -3), P3(4, 5, 6))
	cases := []struct {
		dir  Vector3
		want Point3
	}{
		{V3(1, 1, 1), P3(4, 5, 6)},       // toward Max on every axis
		{V3(-1, -1, -1), P3(-1, -2, -3)}, // toward Min on every axis
		{V3(1, -1, 0), P3(4, -2, 6)},     // mixed; the zero component picks Max
	}
	for _, c := range cases {
		if got := b.FarthestPoint(c.dir); got != c.want {
			t.Errorf("FarthestPoint(%v) = %v, want %v", c.dir, got, c.want)
		}
	}
}

func TestBox2d(t *testing.T) {
	b := Box2dFromPoints(P2(0, 0), P2(4, 2))
	if got := b.Area(); got != 8 {
		t.Errorf("Area = %v, want 8", got)
	}
	if !b.Contains(P2(2, 1)) {
		t.Error("interior point should be contained")
	}
	if !b.Intersects(NewBox2d(P2(3, 1), P2(5, 5))) {
		t.Error("overlapping 2d boxes should intersect")
	}
	if !EmptyBox2d().IsEmpty() {
		t.Error("EmptyBox2d should be empty")
	}
	if got := b.Center(); got != (Point2{2, 1}) {
		t.Errorf("Center = %v, want {2 1}", got)
	}
	u := NewBox2d(P2(0, 0), P2(1, 1)).Union(NewBox2d(P2(2, 3), P2(4, 5)))
	if u.Min != (Point2{0, 0}) || u.Max != (Point2{4, 5}) {
		t.Errorf("Union = %v..%v, want {0 0}..{4 5}", u.Min, u.Max)
	}
}

func TestBoxIntersectsRay(t *testing.T) {
	b := NewBox(P3(0, 0, 0), P3(2, 2, 2))
	cases := []struct {
		name      string
		origin    Point3
		dir       Vector3
		want      bool
		wantEnter Scalar
	}{
		{"hits from outside", P3(1, 1, -5), V3(0, 0, 1), true, 5},
		{"origin inside", P3(1, 1, 1), V3(0, 0, 1), true, 0},
		{"misses beside the box", P3(5, 5, -5), V3(0, 0, 1), false, 0},
		{"points away (box behind)", P3(1, 1, 5), V3(0, 0, 1), false, 0},
		{"axis-parallel grazes through", P3(-5, 1, 1), V3(1, 0, 0), true, 5},
		{"axis-parallel outside slab", P3(-5, 9, 1), V3(1, 0, 0), false, 0},
		{"diagonal corner hit", P3(-1, -1, -1), V3(1, 1, 1), true, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			enter, ok := b.IntersectsRay(c.origin, c.dir)
			if ok != c.want {
				t.Fatalf("ok = %v, want %v", ok, c.want)
			}
			if ok && !IsNearZero(enter-c.wantEnter, 1e-9) {
				t.Errorf("tEnter = %v, want %v", enter, c.wantEnter)
			}
		})
	}
	if _, ok := EmptyBox().IntersectsRay(P3(0, 0, 0), V3(0, 0, 1)); ok {
		t.Error("empty box should never be hit")
	}
}

func TestIsNearZero(t *testing.T) {
	if !IsNearZero(1e-12, 0) {
		t.Error("1e-12 should be near zero at default tolerance")
	}
	if IsNearZero(0.1, 0) {
		t.Error("0.1 should not be near zero")
	}
	if !IsNearZero(0.05, 0.1) {
		t.Error("0.05 should be near zero at explicit tolerance 0.1")
	}
}
