// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// ClosestPointOnEntity is what puts a constraint marker on the geometry it annotates, so its
// clamping and its treatment of closed curves are load-bearing: an unclamped projection would put
// a marker off the end of the segment it belongs to.

// TestClosestPointProjectsOntoASegment: a target beside a line lands at its foot on that line.
func TestClosestPointProjectsOntoASegment(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	ln := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))

	at, ok := ClosestPointOnEntity(ln, math.P2(3, 7))
	if !ok {
		t.Fatal("a line should have an outline")
	}
	if want := math.P2(3, 0); !closeTo(at, want) {
		t.Errorf("closest point %v, want the foot of the perpendicular %v", at, want)
	}
}

// TestClosestPointClampsToTheSegmentEnd: a target beyond the end of a segment projects past it on
// the infinite line, so the result must be clamped or the marker leaves the geometry entirely.
func TestClosestPointClampsToTheSegmentEnd(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	ln := sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(10, 0))

	at, _ := ClosestPointOnEntity(ln, math.P2(50, 4))
	if want := math.P2(10, 0); !closeTo(at, want) {
		t.Errorf("closest point %v, want the segment's end %v — the projection ran off the line", at, want)
	}
}

// TestClosestPointOnACircleIsTheNearSide: the nearest point of a closed curve, which is what puts
// a tangency marker on the touch point rather than at the centre.
func TestClosestPointOnACircleIsTheNearSide(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	ci := sk.Circles().AddByCenterRadius(math.P2(0, 0), 5)

	at, _ := ClosestPointOnEntity(ci, math.P2(20, 0))
	if d := float64(at.DistanceTo(math.P2(5, 0))); d > 0.05 {
		t.Errorf("closest point %v is %.3f from the near side (5,0)", at, d)
	}
}

// TestClosestPointSearchesTheClosingSpan: a closed outline's last vertex joins back to its first,
// and a nearest point falling in that span is only found if the closing segment is walked too.
func TestClosestPointSearchesTheClosingSpan(t *testing.T) {
	pts := []math.Point2{math.P2(0, 0), math.P2(10, 0), math.P2(10, 10), math.P2(0, 10)}
	target := math.P2(-4, 5) // nearest the closing edge (0,10)–(0,0)

	at := closestOnPolyline(pts, true, target)
	if want := math.P2(0, 5); !closeTo(at, want) {
		t.Errorf("closest point %v, want %v on the closing span", at, want)
	}
}

// TestClosestPointOnADegenerateSegment: a zero-length segment has no direction to project along,
// and dividing by its length would give NaN — which would silently poison every marker downstream.
func TestClosestPointOnADegenerateSegment(t *testing.T) {
	at := closestOnSegment(math.P2(3, 4), math.P2(3, 4), math.P2(9, 9))
	if want := math.P2(3, 4); !closeTo(at, want) {
		t.Errorf("closest point on a degenerate segment = %v, want the point itself %v", at, want)
	}
}

// TestClosestPointReportsEntitiesWithNoOutline: text and images have no curve, and callers have to
// be able to tell rather than receive the zero point as if it were a real position.
func TestClosestPointReportsEntitiesWithNoOutline(t *testing.T) {
	sk := NewSketches().Add(XYPlane())
	tb := sk.TextBoxes().Add(math.P2(1, 1), "hello", 0.5, 0, TextLeft)

	if _, ok := ClosestPointOnEntity(tb, math.P2(0, 0)); ok {
		t.Error("a text box reported an outline to project onto")
	}
}
