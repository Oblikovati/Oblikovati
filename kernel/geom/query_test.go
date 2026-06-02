// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

func TestClosestPointAndDistanceToLine(t *testing.T) {
	l, _ := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0)) // the X axis
	if got := ClosestPointOnLine(l, math.P3(3, 4, 0)); !got.IsEqualTo(math.P3(3, 0, 0), eqScalar) {
		t.Errorf("closest = %v, want {3 0 0}", got)
	}
	approxScalar(t, DistancePointToLine(l, math.P3(3, 4, 0)), 4, "distance to line")
}

func TestClosestPointOnSegmentClamps(t *testing.T) {
	s := NewLineSegment(math.P3(0, 0, 0), math.P3(10, 0, 0))
	// Projection beyond the end clamps to the endpoint.
	if got := ClosestPointOnSegment(s, math.P3(20, 5, 0)); !got.IsEqualTo(math.P3(10, 0, 0), eqScalar) {
		t.Errorf("clamped closest = %v, want endpoint {10 0 0}", got)
	}
	// Interior projection is unclamped.
	if got := ClosestPointOnSegment(s, math.P3(4, 5, 0)); !got.IsEqualTo(math.P3(4, 0, 0), eqScalar) {
		t.Errorf("interior closest = %v, want {4 0 0}", got)
	}
	approxScalar(t, DistancePointToSegment(s, math.P3(4, 5, 0)), 5, "distance to segment")
}

func TestPlaneProjectionAndSignedDistance(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1)) // z=0 plane
	approxScalar(t, SignedDistanceToPlane(pl, math.P3(1, 2, 5)), 5, "signed distance above")
	approxScalar(t, SignedDistanceToPlane(pl, math.P3(1, 2, -3)), -3, "signed distance below")
	if got := ProjectPointToPlane(pl, math.P3(1, 2, 5)); !got.IsEqualTo(math.P3(1, 2, 0), eqScalar) {
		t.Errorf("projection = %v, want {1 2 0}", got)
	}
}

func TestLinePlaneIntersection(t *testing.T) {
	pl, _ := NewPlane(math.P3(0, 0, 0), math.V3(0, 0, 1))
	l, _ := NewLine(math.P3(2, 3, -4), math.V3(0, 0, 1)) // straight up through z=0
	pt, ok := LinePlaneIntersection(l, pl, 0)
	if !ok {
		t.Fatal("line should meet plane")
	}
	if !pt.IsEqualTo(math.P3(2, 3, 0), eqScalar) {
		t.Errorf("intersection = %v, want {2 3 0}", pt)
	}
	// A line in a parallel plane never meets z=0.
	par, _ := NewLine(math.P3(0, 0, 5), math.V3(1, 0, 0))
	if _, ok := LinePlaneIntersection(par, pl, 0); ok {
		t.Error("parallel line should not intersect")
	}
}

func TestLineLineClosestAndIntersection(t *testing.T) {
	// Two perpendicular lines offset in Z: closest points are at the crossing
	// in XY, separated by the Z gap.
	xAxis, _ := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	yAt2, _ := NewLine(math.P3(0, 0, 2), math.V3(0, 1, 0))
	onA, onB, ok := LineLineClosest(xAxis, yAt2, 0)
	if !ok {
		t.Fatal("skew lines should have a closest pair")
	}
	if !onA.IsEqualTo(math.P3(0, 0, 0), 1e-9) || !onB.IsEqualTo(math.P3(0, 0, 2), 1e-9) {
		t.Errorf("closest pair = %v,%v, want {0 0 0},{0 0 2}", onA, onB)
	}
	if _, ok := LineLineIntersection(xAxis, yAt2, 1e-9); ok {
		t.Error("skew lines should not report an intersection")
	}
	// Coplanar crossing lines do intersect.
	yAxis, _ := NewLine(math.P3(0, 0, 0), math.V3(0, 1, 0))
	pt, ok := LineLineIntersection(xAxis, yAxis, 1e-9)
	if !ok || !pt.IsEqualTo(math.P3(0, 0, 0), 1e-9) {
		t.Errorf("intersection = %v ok=%v, want origin", pt, ok)
	}
}

func TestParallelLinesHaveNoClosestPair(t *testing.T) {
	a, _ := NewLine(math.P3(0, 0, 0), math.V3(1, 0, 0))
	b, _ := NewLine(math.P3(0, 1, 0), math.V3(1, 0, 0))
	if _, _, ok := LineLineClosest(a, b, 0); ok {
		t.Error("parallel lines should not yield a unique closest pair")
	}
}

func TestLine2dIntersection(t *testing.T) {
	a, _ := NewLine2d(math.P2(0, 0), math.V2(1, 0))
	b, _ := NewLine2d(math.P2(3, -2), math.V2(0, 1))
	pt, ok := Line2dIntersection(a, b, 0)
	if !ok || !pt.IsEqualTo(math.P2(3, 0), eqScalar) {
		t.Errorf("intersection = %v ok=%v, want {3 0}", pt, ok)
	}
	par, _ := NewLine2d(math.P2(0, 5), math.V2(1, 0))
	if _, ok := Line2dIntersection(a, par, 0); ok {
		t.Error("parallel 2d lines should not intersect")
	}
}
