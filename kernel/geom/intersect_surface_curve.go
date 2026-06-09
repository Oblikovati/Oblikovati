// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"

	"oblikovati.org/math"
)

// curveSurfaceSamples is the initial bracketing resolution: the curve is sampled this many
// times over its domain, and each sign change of the signed distance to the surface is
// refined by bisection. High enough to separate the crossings of a multi-turn helix.
const curveSurfaceSamples = 128

// curveSurfaceBisectIter is the bisection refinement depth (≈ 2^-40 of a sample span).
const curveSurfaceBisectIter = 60

// IntersectCurveSurface returns the points where a bounded 3D curve crosses a surface,
// found by sampling the curve's signed distance to the surface and refining each sign
// change by bisection. It handles any number of crossings (e.g. a helix through a plane).
// An unbounded curve domain yields no points (sample the relevant segment instead).
//
// Example — a segment crossing the XY plane yields the one pierce point:
//
//	pl, _ := geom.NewPlane(math.P3(0,0,0), math.V3(0,0,1))
//	seg := geom.NewLineSegment(math.P3(0,0,-1), math.P3(0,0,1))
//	pts := geom.IntersectCurveSurface(seg, pl) // pts[0] ≈ (0,0,0)
func IntersectCurveSurface(c Curve3, s Surface) []math.Point3 {
	lo, hi := c.Domain()
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || hi <= lo {
		return nil
	}
	var out []math.Point3
	prevT := lo
	prevD := SignedDistanceToSurface(s, c.PointAt(lo))
	out = appendIfOnSurface(out, c, lo, prevD)
	for i := 1; i <= curveSurfaceSamples; i++ {
		t := lo + (hi-lo)*float64(i)/float64(curveSurfaceSamples)
		d := SignedDistanceToSurface(s, c.PointAt(t))
		if isCrossing(prevD, d) {
			tc := bisectCurveSurface(c, s, prevT, t)
			out = append(out, c.PointAt(tc))
		} else if i < curveSurfaceSamples {
			out = appendIfOnSurface(out, c, t, d)
		}
		prevT, prevD = t, d
	}
	return out
}

// isCrossing reports a strict sign change between two signed distances (an exact zero at a
// sample is handled separately by appendIfOnSurface to avoid double-counting).
func isCrossing(a, b float64) bool {
	return (a < 0 && b > 0) || (a > 0 && b < 0)
}

// appendIfOnSurface adds the curve point at t when the sample sits exactly on the surface.
func appendIfOnSurface(out []math.Point3, c Curve3, t, d float64) []math.Point3 {
	if d == 0 {
		return append(out, c.PointAt(t))
	}
	return out
}

// bisectCurveSurface refines a bracketed crossing [a, b] (opposite signed distances) to a
// parameter where the curve meets the surface.
func bisectCurveSurface(c Curve3, s Surface, a, b float64) float64 {
	da := SignedDistanceToSurface(s, c.PointAt(a))
	for i := 0; i < curveSurfaceBisectIter; i++ {
		mid := (a + b) / 2
		dm := SignedDistanceToSurface(s, c.PointAt(mid))
		if dm == 0 {
			return mid
		}
		if isCrossing(da, dm) {
			b = mid
		} else {
			a, da = mid, dm
		}
	}
	return (a + b) / 2
}
