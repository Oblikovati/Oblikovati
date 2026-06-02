// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

func TestPoint3TranslateAndVectorTo(t *testing.T) {
	p := P3(1, 1, 1)
	q := p.TranslateBy(V3(2, 3, 4))
	if q != (Point3{3, 4, 5}) {
		t.Errorf("TranslateBy = %v, want {3 4 5}", q)
	}
	// VectorTo is the inverse of TranslateBy.
	if got := p.VectorTo(q); got != (Vector3{2, 3, 4}) {
		t.Errorf("VectorTo = %v, want {2 3 4}", got)
	}
}

func TestPoint3Distance(t *testing.T) {
	a := P3(0, 0, 0)
	b := P3(3, 4, 0)
	if got := a.DistanceTo(b); got != 5 {
		t.Errorf("DistanceTo = %v, want 5", got)
	}
	if got := a.DistanceSquaredTo(b); got != 25 {
		t.Errorf("DistanceSquaredTo = %v, want 25", got)
	}
}

func TestPoint3Midpoint(t *testing.T) {
	if got := P3(0, 0, 0).Midpoint(P3(2, 4, 6)); got != (Point3{1, 2, 3}) {
		t.Errorf("Midpoint = %v, want {1 2 3}", got)
	}
}

func TestPoint3IsEqualTo(t *testing.T) {
	a := P3(1, 2, 3)
	if !a.IsEqualTo(P3(1, 2, 3+1e-12), 0) {
		t.Error("points within default tolerance should be equal")
	}
	if a.IsEqualTo(P3(1, 2, 3.5), 0) {
		t.Error("points beyond tolerance should not be equal")
	}
}

func TestPoint3VectorRoundTrip(t *testing.T) {
	p := P3(7, 8, 9)
	if got := p.AsVector().AsPoint(); got != p {
		t.Errorf("AsVector∘AsPoint = %v, want %v", got, p)
	}
}
