// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
)

func TestSnapToExistingPoint(t *testing.T) {
	s, sk := sketchSession(t)
	sk.Points().Add(math.P2(2, 0))
	// A click just off the existing point snaps onto it, reported as an endpoint snap.
	got := s.snapAt(math.P2(2.1, 0.05))
	if got.Kind != SnapPoint || !got.Point.IsEqualTo(math.P2(2, 0), 1e-9) {
		t.Errorf("snap = %+v, want SnapPoint at (2,0)", got)
	}
}

func TestSnapToOriginAlways(t *testing.T) {
	s, _ := sketchSession(t)
	if got := s.snapAt(math.P2(0.1, -0.08)); got.Kind != SnapPoint || !got.Point.IsEqualTo(math.P2(0, 0), 1e-9) {
		t.Errorf("snap near origin = %+v, want SnapPoint at (0,0)", got)
	}
}

func TestSnapToLineMidpoint(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0)) // midpoint (2,0)
	got := s.snapAt(math.P2(2.05, 0.05))
	if got.Kind != SnapMidpoint || !got.Point.IsEqualTo(math.P2(2, 0), 1e-9) {
		t.Errorf("snap = %+v, want SnapMidpoint at (2,0)", got)
	}
}

func TestSnapToLineEdge(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))
	// Near the line but away from its endpoints/midpoint → on-curve (edge) snap.
	got := s.snapAt(math.P2(1, 0.1))
	if got.Kind != SnapOnCurve || !got.Point.IsEqualTo(math.P2(1, 0), 1e-9) {
		t.Errorf("snap = %+v, want SnapOnCurve at (1,0)", got)
	}
}

func TestSnapToCircleEdge(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	sk.Circles().AddByCenterRadius(math.P2(0, 0), 5)
	// Just outside the ring on the +X axis snaps to (5,0).
	got := s.snapAt(math.P2(5.1, 0))
	if got.Kind != SnapOnCurve || !got.Point.IsEqualTo(math.P2(5, 0), 1e-9) {
		t.Errorf("snap = %+v, want SnapOnCurve at (5,0)", got)
	}
}

func TestSnapToArcEdge(t *testing.T) {
	s, sk := sketchSession(t)
	s.Grid().SnapToGrid = false
	// A radius-5 arc centred at (5,0); the ring's top point is (5,5), far from any
	// defining vertex, so a near click must produce an on-curve (edge) snap to the ring.
	sk.Arcs().AddByCenterStartEnd(math.P2(5, 0), math.P2(0, 0), math.P2(10, 0), true)
	got := s.snapAt(math.P2(5, 5.1))
	if got.Kind != SnapOnCurve || !got.Point.IsEqualTo(math.P2(5, 5), 1e-9) {
		t.Errorf("arc edge snap = %+v, want SnapOnCurve at (5,5)", got)
	}
}

func TestSnapToGridIntersection(t *testing.T) {
	s, _ := sketchSession(t)
	s.Grid().SnapToPoints = false // isolate grid snapping
	got := s.snapAt(math.P2(3.1, 1.95))
	if got.Kind != SnapGrid || !got.Point.IsEqualTo(math.P2(3, 2), 1e-9) {
		t.Errorf("grid snap = %+v, want SnapGrid at (3,2)", got)
	}
}

func TestSnapDisabledReturnsRaw(t *testing.T) {
	s, _ := sketchSession(t)
	s.Grid().SnapToPoints = false
	s.Grid().SnapToGrid = false
	raw := math.P2(3.13, 1.97)
	if got := s.snapAt(raw); got.Kind != SnapNone || !got.Point.IsEqualTo(raw, 1e-9) {
		t.Errorf("with snapping off, snap = %+v, want SnapNone at the raw point", got)
	}
}

func TestSnapFarFromAnythingReturnsRaw(t *testing.T) {
	s, _ := sketchSession(t)
	s.Grid().SnapToGrid = false // only point snapping; nothing near (0.4,0.4)
	raw := math.P2(0.4, 0.4)
	if got := s.snapAt(raw); got.Kind != SnapNone || !got.Point.IsEqualTo(raw, 1e-9) {
		t.Errorf("far-from-anything snap = %+v, want SnapNone", got)
	}
}

func TestSnapAtMapsPixel(t *testing.T) {
	s, sk := sketchSession(t)
	sk.Points().Add(math.P2(0, 0))
	// The centre pixel maps to the origin → a reported endpoint snap.
	if r, ok := s.SnapAt(100, 100); !ok || r.Kind != SnapPoint {
		t.Errorf("SnapAt centre = %+v ok=%v, want a SnapPoint", r, ok)
	}
}
