// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	stdmath "math"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

// edgeArcTable is a dense arc-length table of the picked edge's curve, from which the
// anchors are placed UNIFORM IN ARC LENGTH — never in the raw parameter, which an imported
// B-spline edge can cluster arbitrarily. Split out of fillet_bspline_host_band.go (CLAUDE.md
// 500-line file cap): this is a self-contained curve-sampling primitive, distinct from the
// band's march/loft orchestration that consumes it.
type edgeArcTable struct {
	pts    []math.Point3
	cum    []float64
	length float64
}

// edgeArcTableSamples is the dense polyline resolution behind the arc-length table.
const edgeArcTableSamples = 2048

// newEdgeArcTable samples the edge curve into the table; ok=false on an unbounded domain
// or a degenerate (zero-length) edge.
func newEdgeArcTable(c geom.Curve3) (*edgeArcTable, bool) {
	lo, hi := c.Domain()
	// NaN-REJECTING form, deliberate: every comparison with NaN is false, so `!(a > b)` fires on a
	// NaN operand while `a <= b` stays silent and lets it through. Do not "simplify" (sonar go:S1940).
	if stdmath.IsInf(lo, 0) || stdmath.IsInf(hi, 0) || !(hi > lo) {
		return nil, false
	}
	t := &edgeArcTable{pts: make([]math.Point3, edgeArcTableSamples+1), cum: make([]float64, edgeArcTableSamples+1)}
	for i := 0; i <= edgeArcTableSamples; i++ {
		t.pts[i] = c.PointAt(lo + (hi-lo)*float64(i)/edgeArcTableSamples)
	}
	for i := 1; i <= edgeArcTableSamples; i++ {
		t.cum[i] = t.cum[i-1] + float64(t.pts[i-1].DistanceTo(t.pts[i]))
	}
	t.length = t.cum[edgeArcTableSamples]
	return t, t.length > 0
}

// uniformAnchors places count anchors uniform in arc length, walking forward (dir=+1) or
// backward (dir=−1); tangents are the local polyline directions oriented with the walk.
func (t *edgeArcTable) uniformAnchors(count int, dir float64) []geom.CanalEdgeAnchor {
	out := make([]geom.CanalEdgeAnchor, count)
	for k := 0; k < count; k++ {
		frac := float64(k) / float64(count-1)
		if dir < 0 {
			frac = 1 - frac
		}
		p, tan := t.at(frac * t.length)
		out[k] = geom.CanalEdgeAnchor{P: p, T: tan.Scale(math.Scalar(dir))}
	}
	return out
}

// at evaluates the table at an arc length: the interpolated point and unit tangent.
func (t *edgeArcTable) at(s float64) (math.Point3, math.Vector3) {
	i := t.segmentAt(s)
	seg := t.pts[i].VectorTo(t.pts[i+1])
	segLen := t.cum[i+1] - t.cum[i]
	frac := 0.0
	if segLen > 0 {
		frac = (s - t.cum[i]) / segLen
	}
	tan, err := math.UnitVector3FromVector(seg)
	if err != nil {
		return t.pts[i], math.V3(1, 0, 0)
	}
	return t.pts[i].TranslateBy(seg.Scale(math.Scalar(frac))), tan.AsVector()
}

// closedSeamTangent is the seam tangent of a CLOSED edge averaged across the wrap (the
// chord from the last interior sample to the first), oriented with the walk direction.
func (t *edgeArcTable) closedSeamTangent(dir float64) math.Vector3 {
	n := len(t.pts) - 1
	tan, err := math.UnitVector3FromVector(t.pts[n-1].VectorTo(t.pts[1]))
	if err != nil {
		return math.V3(1, 0, 0)
	}
	return tan.AsVector().Scale(math.Scalar(dir))
}

// segmentAt binary-searches the cumulative table for the segment containing arc length s.
func (t *edgeArcTable) segmentAt(s float64) int {
	lo, hi := 0, len(t.cum)-2
	for lo < hi {
		mid := (lo + hi + 1) / 2
		if t.cum[mid] <= s {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	return lo
}
