// SPDX-License-Identifier: GPL-2.0-only

package drawing

import (
	stdmath "math"
	"testing"

	"oblikovati.org/kernel/hlr"
	"oblikovati.org/math"
)

// facetedLoopSegments returns the circle as N separate one-segment edges sharing endpoints — what
// a sketch extrude-cut produces (each facet is its own B-rep edge), so clustering must reunite them
// by connectivity, not by edge key.
func facetedLoopSegments(cx, cy, r float64, n int) []hlr.Segment {
	pts := circleLoop(cx, cy, r, n)
	segs := make([]hlr.Segment, n)
	for i := range n {
		segs[i] = hlr.Segment{A: pts[i], B: pts[(i+1)%n], EdgeKey: []byte{byte(i)}}
	}
	return segs
}

// circleLoop returns n points evenly spaced on the circle of radius r about (cx, cy) — a stand-in
// for a hole rim tessellated into segment endpoints by the projection.
func circleLoop(cx, cy, r float64, n int) []math.Point2 {
	pts := make([]math.Point2, n)
	for i := range n {
		a := 2 * stdmath.Pi * float64(i) / float64(n)
		pts[i] = math.P2(math.Scalar(cx+r*stdmath.Cos(a)), math.Scalar(cy+r*stdmath.Sin(a)))
	}
	return pts
}

// TestFitCircleRecoversFacetedCircle guards the live-verify finding (#391): a hole rim arrives as
// a loop of faceted points (not an analytic circle), and fitCircle must recover its centre and
// radius so the hole table can tabulate cut holes — the case body.Edges() misses.
func TestFitCircleRecoversFacetedCircle(t *testing.T) {
	c, ok := fitCircle(circleLoop(3, 2, 0.4, 24))
	if !ok {
		t.Fatal("fitCircle rejected a full faceted circle loop, want it recovered")
	}
	if stdmath.Abs(float64(c.center.X)-3) > 1e-6 || stdmath.Abs(float64(c.center.Y)-2) > 1e-6 {
		t.Errorf("fitted centre = (%v, %v), want (3, 2)", c.center.X, c.center.Y)
	}
	if stdmath.Abs(c.radius-0.4) > 1e-6 {
		t.Errorf("fitted radius = %v, want 0.4", c.radius)
	}
}

// TestFitCircleRejectsNonCircles checks fitCircle rejects loops that are not full circles: a
// straight run of points (a flat edge) and a half-circle arc (a fillet), so neither is mistaken
// for a hole.
func TestFitCircleRejectsNonCircles(t *testing.T) {
	line := []math.Point2{}
	for i := 0; i <= 12; i++ {
		line = append(line, math.P2(math.Scalar(float64(i)), 0))
	}
	if _, ok := fitCircle(line); ok {
		t.Error("fitCircle accepted a straight line, want it rejected")
	}

	full := circleLoop(0, 0, 1, 24)
	if _, ok := fitCircle(full[:7]); ok { // first ~half of the loop = an arc
		t.Error("fitCircle accepted a half-arc, want it rejected (a fillet is not a hole)")
	}
}

// TestClusterConnectedSegmentsReunitesFacetedRim is the regression for the live-verify finding
// (#391): a sketch extrude-cut splits a hole rim into many separate edges, so clustering must
// reunite them by shared endpoints into one loop — recovering the hole that per-edge grouping
// would shatter. Two disjoint rims stay two clusters.
func TestClusterConnectedSegmentsReunitesFacetedRim(t *testing.T) {
	one := clusterConnectedSegments(facetedLoopSegments(3, 2, 0.4, 16))
	if len(one) != 1 {
		t.Fatalf("a faceted rim of 16 separate edges = %d clusters, want 1 (reunited by connectivity)", len(one))
	}
	if c, ok := fitCircle(one[0]); !ok || stdmath.Abs(c.radius-0.4) > 1e-6 {
		t.Errorf("reunited rim did not fit a 0.4 circle: ok=%v c=%+v", ok, c)
	}

	two := clusterConnectedSegments(append(
		facetedLoopSegments(0, 0, 1, 12),
		facetedLoopSegments(10, 0, 1, 12)...,
	))
	if len(two) != 2 {
		t.Errorf("two disjoint rims = %d clusters, want 2", len(two))
	}
}
