// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// ExtremumOnBracket returns the parameter of the extremum of a unimodal function inside [a, b] — its
// maximum when wantMax, its minimum otherwise — refined by golden-section search until the bracket is
// below the parameters' own rounding. It is the SOLVE behind every "where does this curve turn"
// question: a chart reads an imprint's azimuth extent from its turning points, and the ruled∩quadric
// gate reads the closest the two section branches come from the discriminant's minimum, each found
// here to rounding rather than at whichever sample happened to fall nearest.
//
// The caller brackets the extremum from a scan — three consecutive samples with the middle one
// extremal — so that the function is unimodal on [a, b].
//
//	tTip := geom.ExtremumOnBracket(azimuthAlong, tPrev, tNext, rising) // the turning point
func ExtremumOnBracket(f func(float64) float64, a, b float64, wantMax bool) float64 {
	sign := 1.0
	if wantMax {
		sign = -1
	}
	const ratio = 0.6180339887498949 // (√5 − 1)/2, the golden section
	lo, hi := stdmath.Min(a, b), stdmath.Max(a, b)
	x1, x2 := hi-ratio*(hi-lo), lo+ratio*(hi-lo)
	f1, f2 := sign*f(x1), sign*f(x2)
	for hi-lo > extremumBracketFloor*(stdmath.Abs(lo)+stdmath.Abs(hi)+1) {
		if f1 < f2 {
			hi, x2, f2 = x2, x1, f1
			x1 = hi - ratio*(hi-lo)
			f1 = sign * f(x1)
		} else {
			lo, x1, f1 = x1, x2, f2
			x2 = lo + ratio*(hi-lo)
			f2 = sign * f(x2)
		}
	}
	return (lo + hi) / 2
}

// extremumBracketFloor ends the golden-section search once the bracket is a few units of the
// parameters' own rounding: below it the function's values no longer separate the two probes.
const extremumBracketFloor = 4e-16 // tol:numeric — relative bracket width at parameter rounding
