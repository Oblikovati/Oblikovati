// SPDX-License-Identifier: GPL-2.0-only

package geom

import "oblikovati.org/math"

// Circle2d is a full 2D circle (contract: Circle2d), parameterized t∈[0,1]
// mapping to angle 2πt measured counter-clockwise from +X about Center.
type Circle2d struct {
	Center math.Point2
	Radius float64
}

// NewCircle2d builds a 2D circle from center and radius.
func NewCircle2d(center math.Point2, radius float64) Circle2d {
	return Circle2d{Center: center, Radius: radius}
}

// Circle2dByThreePoints builds the 2D circle passing through three points. It
// errors when the points are collinear (no finite circle).
func Circle2dByThreePoints(a, b, c math.Point2) (Circle2d, error) {
	center, ok := circumcenter2d(a, b, c)
	if !ok {
		return Circle2d{}, &CollinearPointsError{A: a, B: b, C: c}
	}
	return Circle2d{Center: center, Radius: center.DistanceTo(a)}, nil
}

// PointAt returns the point at angle 2πt.
func (c Circle2d) PointAt(t float64) math.Point2 {
	cos, sin := cosSin(twoPi * t)
	return math.P2(c.Center.X+c.Radius*cos, c.Center.Y+c.Radius*sin)
}

// TangentAt returns the derivative dP/dt at parameter t.
func (c Circle2d) TangentAt(t float64) math.Vector2 {
	cos, sin := cosSin(twoPi * t)
	return math.V2(-c.Radius*sin*twoPi, c.Radius*cos*twoPi)
}

// Domain returns [0, 1].
func (c Circle2d) Domain() (lo, hi float64) { return 0, 1 }

// Circumference returns 2πr.
func (c Circle2d) Circumference() float64 { return twoPi * c.Radius }
