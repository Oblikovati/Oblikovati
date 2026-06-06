// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	"testing"

	"oblikovati/math"
)

// These ear-clip predicates back interiorPoint2D's fallback for sliver regions where no edge
// probe lands clear of the boundary. They are exercised directly here because constructing a
// boolean whose 2D arrangement yields such a sliver is fragile; the predicates themselves are
// pure and small.

func TestTurn2DSign(t *testing.T) {
	// CCW left turn: heading east (0,0)→(1,0) then north (1,0)→(1,1).
	if turn2D(math.P2(0, 0), math.P2(1, 0), math.P2(1, 1)) <= 0 {
		t.Error("a left (CCW) turn should be positive")
	}
	// The mirror is a right turn.
	if turn2D(math.P2(1, 1), math.P2(1, 0), math.P2(0, 0)) >= 0 {
		t.Error("a right (CW) turn should be negative")
	}
	// Collinear points have ~zero turn.
	if v := turn2D(math.P2(0, 0), math.P2(1, 0), math.P2(2, 0)); v > arrTol || v < -arrTol {
		t.Errorf("collinear turn = %g, want ≈ 0", v)
	}
}

func TestPointInTriangle2D(t *testing.T) {
	a, b, c := math.P2(0, 0), math.P2(4, 0), math.P2(0, 4)
	if !pointInTriangle2D(math.P2(1, 1), a, b, c) {
		t.Error("interior point reported outside")
	}
	if pointInTriangle2D(math.P2(3, 3), a, b, c) {
		t.Error("exterior point reported inside")
	}
	if !pointInTriangle2D(math.P2(2, 0), a, b, c) {
		t.Error("edge point should be inclusive")
	}
}

func TestEarEmpty(t *testing.T) {
	// Convex quad: the ear at any vertex contains no other vertex.
	square := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(2, 2), math.P2(0, 2)}
	if !earEmpty(square, 1) {
		t.Error("convex-quad ear should be empty")
	}
	// A reflex "arrowhead": the ear at the spike tip swallows the reflex vertex.
	arrow := []math.Point2{math.P2(0, 0), math.P2(4, 2), math.P2(0, 4), math.P2(1, 2)}
	if earEmpty(arrow, 1) {
		t.Error("ear that contains the reflex vertex should not be empty")
	}
}

func TestCentroid2D(t *testing.T) {
	c := centroid2D([]math.Point2{math.P2(0, 0), math.P2(3, 0), math.P2(0, 3)})
	if c.X != 1 || c.Y != 1 {
		t.Errorf("centroid = %v, want (1,1)", c)
	}
}
