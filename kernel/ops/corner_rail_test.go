// SPDX-License-Identifier: GPL-2.0-only
package ops

import (
	"testing"

	"oblikovati.org/kernel/geom"
	"oblikovati.org/math"
)

func lineSide(a, b math.Point3, cont Continuity) Side {
	return Side{Curve: geom.LineSegment{StartPoint: a, EndPoint: b}, Cont: cont}
}

func TestRailLoopClosedAndValence(t *testing.T) {
	p0, p1, p2 := math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)
	loop := RailLoop{Sides: []Side{
		lineSide(p0, p1, G1), lineSide(p1, p2, G1), lineSide(p2, p0, G0),
	}}
	if !loop.Closed(1e-9) {
		t.Fatalf("triangle loop should be Closed within 1e-9")
	}
	if got := loop.Valence(); got != 3 {
		t.Fatalf("Valence = %d, want 3", got)
	}
}

func TestRailLoopOpenIsNotClosed(t *testing.T) {
	p0, p1, p2 := math.P3(0, 0, 0), math.P3(1, 0, 0), math.P3(1, 1, 0)
	open := RailLoop{Sides: []Side{lineSide(p0, p1, G1), lineSide(p1, p2, G1)}} // gap p2->p0
	if open.Closed(1e-9) {
		t.Fatalf("open loop must not report Closed")
	}
}
