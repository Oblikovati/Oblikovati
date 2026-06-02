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
