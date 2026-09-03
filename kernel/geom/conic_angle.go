// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
)

// conicParamOfAngle inverts conicAngleOfParam onto the parameter span (t0, t1) (either order), for an
// angle that lands STRICTLY inside it. A closed conic's angle is periodic, so every branch of θ + 2πk is
// tried; ok=false when no branch falls inside the span.
func conicParamOfAngle(c Curve3, theta, t0, t1 float64) (float64, bool) {
	lo, hi := stdmath.Min(t0, t1), stdmath.Max(t0, t1)
	inside := func(t float64) bool { return t > lo && t < hi }
	switch x := c.(type) {
	case Circle, EllipseFull:
		return periodicParamInside(theta/(2*stdmath.Pi), 1, inside)
	case Arc3d:
		return sweepParamInside(theta, x.StartAngle, x.SweepAngle, inside)
	case EllipticalArc:
		return sweepParamInside(theta, x.StartAngle, x.SweepAngle, inside)
	case Hyperbola:
		if inside(theta) {
			return theta, true
		}
	case HyperbolicArc:
		if x.Theta1 == x.Theta0 {
			return 0, false
		}
		if t := (theta - x.Theta0) / (x.Theta1 - x.Theta0); inside(t) {
			return t, true
		}
	}
	return 0, false
}

// periodicParamInside tries a periodic parameter's branches t + k·period against the span predicate.
func periodicParamInside(t, period float64, inside func(float64) bool) (float64, bool) {
	base := t - period*stdmath.Floor(t/period)
	for _, cand := range []float64{base - period, base, base + period} {
		if inside(cand) {
			return cand, true
		}
	}
	return 0, false
}

// sweepParamInside maps an angle onto an arc's [0,1] parameter (angle = start + t·sweep), trying the
// 2π branches of the angle, and reports the one strictly inside the span.
func sweepParamInside(theta, start, sweep float64, inside func(float64) bool) (float64, bool) {
	if sweep == 0 {
		return 0, false
	}
	base := (theta - start) / sweep
	period := 2 * stdmath.Pi / stdmath.Abs(sweep)
	for k := -2.0; k <= 2; k++ {
		if cand := base + k*period; inside(cand) {
			return cand, true
		}
	}
	return 0, false
}

// arcParamAt maps a point's polar angle onto an arc's [0,1] parameter (angle = start + t·sweep), moved
// to the 2π branch nearest the arc's mid-angle so a point on the arc reads inside [0,1].
func arcParamAt(theta, start, sweep float64) float64 {
	if sweep == 0 {
		return 0
	}
	mid := start + sweep/2
	theta += 2 * stdmath.Pi * stdmath.Round((mid-theta)/(2*stdmath.Pi))
	return (theta - start) / sweep
}
