// SPDX-License-Identifier: GPL-2.0-only

package math

import "testing"

func TestPoint2TranslateAndVectorTo(t *testing.T) {
	p := P2(1, 1)
	q := p.TranslateBy(V2(2, 3))
	if q != (Point2{3, 4}) {
		t.Errorf("TranslateBy = %v, want {3 4}", q)
	}
	if got := p.VectorTo(q); got != (Vector2{2, 3}) {
		t.Errorf("VectorTo = %v, want {2 3}", got)
	}
}

func TestPoint2Distance(t *testing.T) {
	if got := P2(0, 0).DistanceTo(P2(3, 4)); got != 5 {
		t.Errorf("DistanceTo = %v, want 5", got)
	}
}

func TestPoint2Midpoint(t *testing.T) {
	if got := P2(0, 0).Midpoint(P2(2, 4)); got != (Point2{1, 2}) {
		t.Errorf("Midpoint = %v, want {1 2}", got)
	}
}
