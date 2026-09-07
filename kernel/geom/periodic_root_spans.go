// SPDX-License-Identifier: GPL-2.0-only

package geom

import stdmath "math"

// The TOPOLOGY half of a two-root section (ADR-0061 stages 4 and 5), shared by every closed form that
// solves one surface against another station by station.
//
// Both of the intersector's parametric×implicit forms have the same shape. A ruled surface substituted
// into a quadric leaves a quadratic in the ruling parameter, one per azimuth; a torus against an
// axis-invariant quadric leaves a single harmonic in the azimuth, one per tube angle. Either way the
// station parameter runs a full period, at each station there are two ordered roots or none, and which
// it is is the sign of a discriminant. So the QUESTION — which connected pieces do those roots form —
// is one question, and it is answered here once: the maximal spans where the discriminant is positive,
// with their ends refined to the folds where the two roots meet.

// periodicRootWindows returns the maximal spans of [0, 2π) over which disc is positive, each bounded by
// the exact station where it crosses zero. A span that wraps the seam comes back as [S0, S0+width] with
// the end past 2π, which a periodic chart evaluates unchanged.
//
// ok=false is "not a window at all": the discriminant never changes sign while being positive somewhere,
// which is the FULL-PERIOD case a wrapping form owns and which must not be dressed up as a window whose
// two folds are the same station. An empty list with ok=true is the honest "they do not meet".
//
//	spans, ok := periodicRootWindows(func(u float64) float64 { return co(u).discriminant() }, 720)
func periodicRootWindows(disc func(float64) float64, probes int) ([][2]float64, bool) {
	step := twoPi / float64(probes)
	samples := make([]float64, probes)
	for i := range samples {
		samples[i] = disc(float64(i) * step)
	}
	var rises, falls []float64
	for i, d := range samples {
		prev := samples[(i+probes-1)%probes]
		switch {
		case prev <= 0 && d > 0:
			rises = append(rises, foldStation(disc, float64(i-1)*step, float64(i)*step))
		case prev > 0 && d <= 0:
			falls = append(falls, foldStation(disc, float64(i-1)*step, float64(i)*step))
		}
	}
	if len(rises) == 0 && len(falls) == 0 {
		return nil, allNonPositive(samples)
	}
	if len(rises) != len(falls) {
		return nil, false // sign changes alternate on a circle; a count that disagrees is numerical noise
	}
	out := make([][2]float64, 0, len(rises))
	for _, s0 := range rises {
		out = append(out, [2]float64{s0, nextStationAbove(falls, s0)})
	}
	return out, true
}

// allNonPositive reports that the two roots meet or miss at every probe.
func allNonPositive(samples []float64) bool {
	for _, d := range samples {
		if d > 0 {
			return false
		}
	}
	return true
}

// nextStationAbove returns the first fall station strictly after s0, wrapped by a period when the window
// straddles the seam — so a window is always [S0, S1] with S0 < S1.
func nextStationAbove(falls []float64, s0 float64) float64 {
	best := stdmath.Inf(1)
	for _, f := range falls {
		s := f
		if s <= s0 {
			s += twoPi
		}
		best = stdmath.Min(best, s)
	}
	return best
}

// foldStation refines a bracketed sign change of the discriminant to the fold itself: a bisection on a
// smooth periodic function with one root in the bracket — no derivative, unconditionally convergent.
//
// It returns the endpoint on the NON-POSITIVE side, never the midpoint. That side is where the two roots
// have already merged, so a fold-admitting root reader answers the SAME value for both branches there
// and a loop built on the window closes exactly. Taken from the positive side the two roots differ by
// the branch separation, which goes as √Δ: a discriminant bisected to 1e-20 still leaves a 1e-10 gap.
func foldStation(disc func(float64) float64, lo, hi float64) float64 {
	loPositive := disc(lo) > 0
	for range foldBisectionSteps {
		mid := (lo + hi) / 2
		if (disc(mid) > 0) == loPositive {
			lo = mid
			continue
		}
		hi = mid
	}
	if loPositive {
		return hi
	}
	return lo
}

// foldBisectionSteps halves a probe bracket down to the fold. The bracket is a few thousandths of a turn
// wide, so 60 halvings take it below the double's own resolution — which is what the window's two ends
// have to be for a loop built on it to close on itself.
const foldBisectionSteps = 60
