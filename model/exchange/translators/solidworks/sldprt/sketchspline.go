// SPDX-License-Identifier: GPL-2.0-only

package sldprt

// Spline is a fit-point spline: its interpolation points in order, and whether it is closed.
type Spline struct {
	FitPoints []Point
	Closed    bool
}

// splineFromPoints builds a fit-point spline from a spline sketch's cached points, which SolidWorks
// stores as the ordered interpolation points (verified on a four-point spline). A closed spline
// repeats its first point last; that duplicate is dropped and Closed is set. Returns ok=false for
// fewer than two distinct points (nothing to interpolate).
func splineFromPoints(ordered []Point) (Spline, bool) {
	pts := distinctPoints(ordered)
	if len(pts) < 2 {
		return Spline{}, false
	}
	closed := len(ordered) > len(pts) && ordered[0] == ordered[len(ordered)-1]
	return Spline{FitPoints: pts, Closed: closed}, true
}
