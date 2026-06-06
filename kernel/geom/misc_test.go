// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"strings"
	"testing"

	"oblikovati/math"
)

func TestLineThrough(t *testing.T) {
	l, err := LineThrough(math.P3(0, 0, 0), math.P3(0, 10, 0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := l.PointAt(4); !got.IsEqualTo(math.P3(0, 4, 0), eqScalar) {
		t.Errorf("PointAt(4) = %v, want {0 4 0}", got)
	}
}

func TestCircumferenceAndDomains(t *testing.T) {
	c, _ := NewCircle(math.P3(0, 0, 0), math.V3(0, 0, 1), 2)
	approxScalar(t, c.Circumference(), twoPi*2, "Circumference3d")
	approxScalar(t, NewCircle2d(math.P2(0, 0), 3).Circumference(), twoPi*3, "Circumference2d")

	lo, hi := c.Domain()
	if lo != 0 || hi != 1 {
		t.Errorf("circle Domain = [%v,%v], want [0,1]", lo, hi)
	}
}

func TestArc3dLength(t *testing.T) {
	a, _ := NewArc3d(math.P3(0, 0, 0), math.V3(0, 0, 1), math.V3(1, 0, 0), 2, 0, twoPi/4)
	approxScalar(t, a.Length(), twoPi/4*2, "Arc3d.Length")
}

func TestCollinearErrorMessages(t *testing.T) {
	e2 := &CollinearPointsError{A: math.P2(0, 0), B: math.P2(1, 0), C: math.P2(2, 0)}
	if !strings.Contains(e2.Error(), "collinear") {
		t.Errorf("2D error message lacks 'collinear': %q", e2.Error())
	}
	e3 := &CollinearPoints3dError{A: math.P3(0, 0, 0), B: math.P3(1, 1, 1), C: math.P3(2, 2, 2)}
	if !strings.Contains(e3.Error(), "collinear") {
		t.Errorf("3D error message lacks 'collinear': %q", e3.Error())
	}
}
