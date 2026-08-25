// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"

	"oblikovati.org/math"
)

// ArcShaped is the optional capability of a sketch arc: a point on the arc halfway along its ACTUAL
// (CounterClockwise-respecting) sweep. Center + start + end alone define two arcs — the minor and the
// major — so a consumer rebuilding the analytic arc (extrude keeping arc cap edges, revolve) needs
// this interior point to pick the right one without reading the winding flag itself. Only Arc carries
// it; consumers assert the capability, never the concrete type (the sketch-entity seam, #1624).
type ArcShaped interface {
	// ArcMidpoint returns the sketch-space point at the arc's angular midpoint along its sweep.
	ArcMidpoint() math.Point2
}

var _ ArcShaped = (*Arc)(nil)

// ArcMidpoint returns the point halfway along the arc's swept angle, on the side the arc actually
// runs (CounterClockwise ? increasing : decreasing angle), so it lies strictly between Start and End
// on the real arc — the disambiguating third point for [geom.Arc3dByThreePoints].
func (a *Arc) ArcMidpoint() math.Point2 {
	c := a.Center.Position()
	sa := angleFromCenter(c, a.Start.Position())
	ea := angleFromCenter(c, a.End.Position())
	if a.CounterClockwise {
		for ea <= sa {
			ea += 2 * stdmath.Pi
		}
	} else {
		for ea >= sa {
			ea -= 2 * stdmath.Pi
		}
	}
	mid := (sa + ea) / 2
	r := float64(a.Radius())
	return math.P2(c.X+math.Scalar(r*stdmath.Cos(mid)), c.Y+math.Scalar(r*stdmath.Sin(mid)))
}

// angleFromCenter is the polar angle of p about center.
func angleFromCenter(center, p math.Point2) float64 {
	return stdmath.Atan2(float64(p.Y-center.Y), float64(p.X-center.X))
}
