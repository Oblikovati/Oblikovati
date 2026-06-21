// SPDX-License-Identifier: GPL-2.0-only

package math

import "sort"

// Median returns the middle value of v — averaging the two middle values for an even count —
// or 0 for an empty slice. It sorts v in place, so callers pass a scratch slice they own. The
// median is robust to outliers (unlike the mean), which is why the importer uses it to find a
// drawing's centre and the viewport uses it to frame a drawing's bulk.
func Median(v []float64) float64 {
	return Percentile(v, 0.5)
}

// Percentile returns the q-quantile (q in [0,1]) of v by linear interpolation between the two
// nearest ranks, or 0 for an empty slice. It sorts v in place. q is clamped to [0,1].
//
// Example:
//
//	p99 := Percentile(xs, 0.99) // the value below which 99% of xs fall
func Percentile(v []float64, q float64) float64 {
	if len(v) == 0 {
		return 0
	}
	if q < 0 {
		q = 0
	} else if q > 1 {
		q = 1
	}
	sort.Float64s(v)
	if len(v) == 1 {
		return v[0]
	}
	pos := q * float64(len(v)-1)
	lo := int(pos)
	frac := pos - float64(lo)
	if lo+1 >= len(v) {
		return v[len(v)-1]
	}
	return v[lo] + frac*(v[lo+1]-v[lo])
}
