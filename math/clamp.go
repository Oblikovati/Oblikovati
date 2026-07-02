// SPDX-License-Identifier: GPL-2.0-only

package math

import "cmp"

// Clamp limits v to [lo, hi]; it requires lo <= hi (an inverted range is a
// caller bug, not a case Clamp arbitrates). It is the single clamp for the
// whole codebase (#1652) — the eight hand-rolled shapes it replaced disagreed
// on range conventions, so one documented contract lives here instead.
// NaN handling is comparison-based: a NaN v passes through unchanged.
//
//	u := math.Clamp(u, uMin, uMax) // pin a surface parameter to its domain
func Clamp[T cmp.Ordered](v, lo, hi T) T {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Clamp01 limits v to the unit interval [0, 1] — the ubiquitous
// curve/segment-parameter guard (#1652).
//
//	t := math.Clamp01(dot / lenSq) // project onto a bounded segment
func Clamp01(v float64) float64 {
	return Clamp(v, 0, 1)
}
