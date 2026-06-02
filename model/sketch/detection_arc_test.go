// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// polygonArea is the shoelace area (absolute) of a closed polygon.
func polygonArea(poly []math.Point2) float64 {
	a := 0.0
	n := len(poly)
	for i := 0; i < n; i++ {
		j := (i + 1) % n
		a += poly[i].X*poly[j].Y - poly[j].X*poly[i].Y
	}
	return stdmath.Abs(a) / 2
}

// TestArcLoopPolygonFollowsArc is a regression for the extrude-of-arcs bug: a
// half-disc (a diameter line + a semicircular arc) must yield a polygon whose area
// approaches πr²/2, not the zero-area chord the old endpoint-only walk produced.
func TestArcLoopPolygonFollowsArc(t *testing.T) {
	const r = 5.0
	s := NewSketches().Add(XYPlane())
	left := s.Points().Add(math.P2(-r, 0))
	right := s.Points().Add(math.P2(r, 0))
	s.Lines().Add(right, left)                                             // diameter (closes the loop)
	s.Arcs().Add(s.Points().Add(math.P2(0, 0)), left, right, true /*ccw*/) // semicircle over the top

	ps := s.Profiles()
	if ps.Count() != 1 {
		t.Fatalf("profile count = %d, want 1", ps.Count())
	}
	p := ps.Item(0)
	if !p.IsClosed() {
		t.Fatal("half-disc profile is not closed")
	}
	got := polygonArea(p.OuterLoop().Polygon())
	want := stdmath.Pi * r * r / 2
	if stdmath.Abs(got-want)/want > 0.02 { // within 2% of the true semicircle area
		t.Errorf("half-disc polygon area = %.3f, want ≈ %.3f (arc flattened to a chord?)", got, want)
	}
}
