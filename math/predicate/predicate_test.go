// SPDX-License-Identifier: GPL-2.0-only

package predicate

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func sign(v float64) int {
	switch {
	case v > 0:
		return 1
	case v < 0:
		return -1
	default:
		return 0
	}
}

func TestOrient2DSigns(t *testing.T) {
	a, b := math.P2(0, 0), math.P2(1, 0)
	if sign(Orient2D(a, b, math.P2(0, 1))) != 1 {
		t.Error("CCW triangle should be positive")
	}
	if sign(Orient2D(a, b, math.P2(0, -1))) != -1 {
		t.Error("CW triangle should be negative")
	}
	if sign(Orient2D(a, b, math.P2(2, 0))) != 0 {
		t.Error("collinear points should be exactly zero")
	}
}

// A classic near-degenerate case: points so close to collinear that naive float64
// gives the wrong sign. The exact fallback must return the correct one.
func TestOrient2DRobustNearCollinear(t *testing.T) {
	a := math.P2(0.5, 0.5)
	b := math.P2(12, 12) // exactly on the line y=x through a
	// c just above the line y=x by a tiny amount → must be on the positive side.
	c := math.P2(0.5, 0.5+1e-15)
	if sign(Orient2D(a, b, c)) != 1 {
		t.Errorf("point just above the line misclassified: %v", Orient2D(a, b, c))
	}
	// Exactly on the line → exactly zero, however tiny the coordinates.
	if sign(Orient2D(a, b, math.P2(3, 3))) != 0 {
		t.Error("exactly-collinear point not reported zero")
	}
}

func TestOrient3DSigns(t *testing.T) {
	a, b, c := math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(0, 1, 0)
	below := math.P3(0, 0, -1)
	above := math.P3(0, 0, 1)
	if sign(Orient3D(a, b, c, below)) == sign(Orient3D(a, b, c, above)) {
		t.Error("points on opposite sides of a plane must have opposite signs")
	}
	if sign(Orient3D(a, b, c, math.P3(1, 1, 0))) != 0 {
		t.Error("coplanar point should be exactly zero")
	}
}

func TestInCircleSigns(t *testing.T) {
	// CCW triangle on the unit circle.
	a, b, c := math.P2(1, 0), math.P2(0, 1), math.P2(-1, 0)
	if sign(InCircle(a, b, c, math.P2(0, 0))) != 1 {
		t.Error("origin should be inside the unit circle")
	}
	if sign(InCircle(a, b, c, math.P2(2, 2))) != -1 {
		t.Error("far point should be outside")
	}
	if sign(InCircle(a, b, c, math.P2(0, -1))) != 0 {
		t.Error("cocircular point should be exactly zero")
	}
}

func TestExactFallbackMatchesFloatWhenClear(t *testing.T) {
	// When well away from degeneracy, the fast path and exact path agree.
	a, b, c, d := math.P2(0, 0), math.P2(4, 0), math.P2(2, 3), math.P2(2, 1)
	if sign(InCircle(a, b, c, d)) != orientSignViaExact(a, b, c, d) {
		t.Error("fast and exact in-circle disagree on a clear case")
	}
}

func orientSignViaExact(a, b, c, d math.Point2) int { return inCircleExact(a, b, c, d) }
