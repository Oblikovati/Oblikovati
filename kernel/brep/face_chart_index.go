// SPDX-License-Identifier: GPL-2.0-only

package brep

import (
	stdmath "math"

	"oblikovati.org/math"
)

// A chart's containment index (ADR-0063).
//
// The chart is read far more often than it is built: the flux quadrature asks cellTrimFraction for a
// grid of samples per quadrature cell per face, and every containment query walks the contours. That was
// affordable while the contours were the projected loops — a few hundred points — and is not now: a
// chart traced from the arrangement's own cell boundaries carries a sample per arrangement segment, and
// on a fine-pitch coil that is thousands. Measured, model/feature went from ~810 s to over 3000 s on the
// unindexed scan.
//
// The ray is cast along +u, so only the segments whose v-range straddles the query can cross it. Bucket
// the segments by v once, and each query visits the few in its own bucket. The verdict is unchanged —
// this indexes the same even-odd count, it does not approximate it.

// chartIndex buckets a chart's segments by the v-range they span.
type chartIndex struct {
	contours [][]math.Point2
	buckets  [][]chartSeg
	v0, dv   float64
}

// chartSeg is one contour segment, kept by value so a query does not chase the contour slices.
type chartSeg struct{ a, b math.Point2 }

// chartBucketLoad is the average number of segments per bucket the index aims for. Fewer buckets makes
// each query scan more; more buckets makes a long segment (a seam, which spans the whole v range)
// occupy more of them. It is a sizing constant, not a tolerance.
const chartBucketLoad = 8

// newChartIndex builds the index, or returns nil for a chart small enough that scanning it beats
// bucketing it.
func newChartIndex(contours [][]math.Point2) *chartIndex {
	n := 0
	for _, c := range contours {
		n += len(c)
	}
	if n < 4*chartBucketLoad {
		return nil
	}
	v0, v1 := chartVRange(contours)
	if !(v1 > v0) {
		return nil
	}
	ix := &chartIndex{contours: contours, buckets: make([][]chartSeg, n/chartBucketLoad), v0: v0}
	ix.dv = (v1 - v0) / float64(len(ix.buckets))
	for _, c := range contours {
		for i, m := 0, len(c); i < m; i++ {
			ix.insert(chartSeg{a: c[i], b: c[(i+1)%m]})
		}
	}
	return ix
}

// chartVRange is the v-extent the buckets divide.
func chartVRange(contours [][]math.Point2) (float64, float64) {
	lo, hi := stdmath.Inf(1), stdmath.Inf(-1)
	for _, c := range contours {
		for _, p := range c {
			lo, hi = stdmath.Min(lo, float64(p.Y)), stdmath.Max(hi, float64(p.Y))
		}
	}
	return lo, hi
}

// insert files a segment into every bucket its v-range touches.
func (ix *chartIndex) insert(s chartSeg) {
	lo, hi := ix.bucketOf(float64(s.a.Y)), ix.bucketOf(float64(s.b.Y))
	if lo > hi {
		lo, hi = hi, lo
	}
	for i := lo; i <= hi; i++ {
		ix.buckets[i] = append(ix.buckets[i], s)
	}
}

// bucketOf is the bucket a v falls in, clamped to the index's range.
func (ix *chartIndex) bucketOf(v float64) int {
	i := int((v - ix.v0) / ix.dv)
	return max(0, min(len(ix.buckets)-1, i))
}

// contains is the even-odd verdict, counting only the bucketed segments a +u ray can cross.
func (ix *chartIndex) contains(q math.Point2) bool {
	crossings := 0
	for _, s := range ix.buckets[ix.bucketOf(float64(q.Y))] {
		if segCrossesURay(q, s.a, s.b) {
			crossings++
		}
	}
	return crossings%2 == 1
}
