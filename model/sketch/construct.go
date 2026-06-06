// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"fmt"

	"oblikovati/math"
)

// AddByThreePoints creates a circle through three points (the circumcircle). It errors
// if the points are collinear (no finite circumcircle).
//
//	c, err := sk.Circles().AddByThreePoints(math.P2(0,0), math.P2(2,0), math.P2(1,1))
func (c *Circles) AddByThreePoints(a, b, d math.Point2) (*Circle, error) {
	center, radius, err := circumcircle(a, b, d)
	if err != nil {
		return nil, err
	}
	return c.AddByCenterRadius(center, radius), nil
}

// AddByThreePoints creates an arc through three points: it starts at a, ends at d, and
// passes through b. It errors if the points are collinear.
//
//	arc, err := sk.Arcs().AddByThreePoints(math.P2(2,0), math.P2(0,2), math.P2(-2,0))
func (c *Arcs) AddByThreePoints(a, b, d math.Point2) (*Arc, error) {
	center, _, err := circumcircle(a, b, d)
	if err != nil {
		return nil, err
	}
	// The sweep from a to d that passes through b is CCW exactly when a,b,d wind CCW.
	ccw := signedArea(a, b, d) > 0
	return c.AddByCenterStartEnd(center, a, d, ccw), nil
}

// circumcircle returns the center and radius of the circle through three points, or an
// error when they are collinear (the perpendicular bisectors are parallel).
func circumcircle(a, b, d math.Point2) (math.Point2, math.Scalar, error) {
	ax, ay := float64(a.X), float64(a.Y)
	bx, by := float64(b.X), float64(b.Y)
	dx, dy := float64(d.X), float64(d.Y)
	det := 2 * (ax*(by-dy) + bx*(dy-ay) + dx*(ay-by))
	if det == 0 {
		return math.Point2{}, 0, fmt.Errorf("circumcircle: points %v %v %v are collinear", a, b, d)
	}
	a2, b2, d2 := ax*ax+ay*ay, bx*bx+by*by, dx*dx+dy*dy
	ux := (a2*(by-dy) + b2*(dy-ay) + d2*(ay-by)) / det
	uy := (a2*(dx-bx) + b2*(ax-dx) + d2*(bx-ax)) / det
	center := math.P2(math.Scalar(ux), math.Scalar(uy))
	return center, center.DistanceTo(a), nil
}

// signedArea returns twice the signed area of triangle a,b,c (positive ⇒ CCW winding).
func signedArea(a, b, c math.Point2) float64 {
	return float64((b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X))
}
