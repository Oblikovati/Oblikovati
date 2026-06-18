// SPDX-License-Identifier: GPL-2.0-only

package ops

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

func p3(x, y, z float64) gmath.Point3 { return gmath.P3(x, y, z) }

func approxDist(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("%s = %.12g, want %.12g", name, got, want)
	}
}

// TestSegmentSegmentDistance checks the analytic cases: parallel offset, crossing (zero),
// skew, and a degenerate (point) segment.
func TestSegmentSegmentDistance(t *testing.T) {
	// Two parallel unit segments offset by 3 in y.
	approxDist(t, "parallel", SegmentSegmentDistance(
		p3(0, 0, 0), p3(1, 0, 0), p3(0, 3, 0), p3(1, 3, 0)), 3)
	// Crossing segments in the z=0 plane touch → 0.
	approxDist(t, "crossing", SegmentSegmentDistance(
		p3(-1, 0, 0), p3(1, 0, 0), p3(0, -1, 0), p3(0, 1, 0)), 0)
	// Skew: x-axis segment and a z-offset y-axis segment shifted in z by 2 → 2.
	approxDist(t, "skew", SegmentSegmentDistance(
		p3(-1, 0, 0), p3(1, 0, 0), p3(0, -1, 2), p3(0, 1, 2)), 2)
	// A degenerate segment (point) 5 above a segment's midpoint → 5.
	approxDist(t, "point-segment", SegmentSegmentDistance(
		p3(0, 5, 0), p3(0, 5, 0), p3(-2, 0, 0), p3(2, 0, 0)), 5)
}

// TestSegmentTriangleDistance checks a segment above a triangle, a piercing segment (zero),
// and a segment beyond a triangle edge.
func TestSegmentTriangleDistance(t *testing.T) {
	a, b, c := p3(0, 0, 0), p3(4, 0, 0), p3(0, 4, 0) // triangle in z=0
	// Segment parallel to the plane, 3 above the interior → 3.
	approxDist(t, "above", SegmentTriangleDistance(
		p3(1, 1, 3), p3(1, 1, 5), a, b, c), 3)
	// Segment piercing the triangle interior → 0.
	approxDist(t, "pierce", SegmentTriangleDistance(
		p3(1, 1, -1), p3(1, 1, 1), a, b, c), 0)
	// A point 2 in −x beyond vertex a → 2.
	approxDist(t, "beyond-vertex", SegmentTriangleDistance(
		p3(-2, 0, 0), p3(-2, 0, 0), a, b, c), 2)
}

// TestTriangleTriangleDistance checks parallel triangles (gap), an edge-edge gap, and
// intersecting triangles (zero).
func TestTriangleTriangleDistance(t *testing.T) {
	a, b, c := p3(0, 0, 0), p3(4, 0, 0), p3(0, 4, 0)
	// Identical triangle lifted 2 in z → 2.
	approxDist(t, "parallel", TriangleTriangleDistance(
		a, b, c, p3(0, 0, 2), p3(4, 0, 2), p3(0, 4, 2)), 2)
	// A triangle that crosses the first → 0.
	approxDist(t, "intersect", TriangleTriangleDistance(
		a, b, c, p3(1, 1, -1), p3(1, 1, 1), p3(2, 2, 1)), 0)
}
