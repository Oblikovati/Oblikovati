// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	stdmath "math"
	"testing"

	"oblikovati/math"
)

func polyAbsArea(p []math.Point2) float64 { return stdmath.Abs(polygonSignedArea(p)) }

// TestOffsetClosedPolygonMinkowski pins the rounded region offset against the exact Minkowski
// area: growing a convex polygon by d adds perimeter·d + π·d² (the swept band + corner discs).
func TestOffsetClosedPolygonMinkowski(t *testing.T) {
	sq := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(2, 2), math.P2(0, 2)} // side 2
	for _, d := range []float64{0.5, 1.0} {
		got := polyAbsArea(offsetClosedPolygon(sq, d, 128))
		want := 4 + 4*2*d + stdmath.Pi*d*d // s² + perimeter·d + π·d²
		if stdmath.Abs(got-want)/want > 5e-4 {
			t.Errorf("grow d=%.1f area=%.5f, want %.5f", d, got, want)
		}
	}
	// Shrinking by 0.5 mitres the corners (no arcs) → a side-1 square, area exactly 1.
	if a := polyAbsArea(offsetClosedPolygon(sq, -0.5, 128)); stdmath.Abs(a-1) > 1e-9 {
		t.Errorf("shrink area = %.6f, want 1", a)
	}
	// Winding-independent: a CW square offsets to the same area.
	cw := []math.Point2{math.P2(0, 0), math.P2(0, 2), math.P2(2, 2), math.P2(2, 0)}
	if a := polyAbsArea(offsetClosedPolygon(cw, 1, 128)); stdmath.Abs(a-(4+8+stdmath.Pi))/(4+8+stdmath.Pi) > 5e-4 {
		t.Errorf("CW-input grow area = %.5f, want %.5f", a, 4+8+stdmath.Pi)
	}
}

// TestOffsetClosedPolygonReflex covers a corner that mitres rather than rounds: growing an
// L-shape (one reflex vertex) increases its area and keeps a sane vertex count.
func TestOffsetClosedPolygonReflex(t *testing.T) {
	l := []math.Point2{math.P2(0, 0), math.P2(3, 0), math.P2(3, 1), math.P2(1, 1), math.P2(1, 3), math.P2(0, 3)}
	base := polyAbsArea(l)
	grown := offsetClosedPolygon(l, 0.3, 16)
	if a := polyAbsArea(grown); a <= base {
		t.Errorf("grown L area %.4f should exceed base %.4f", a, base)
	}
	if len(grown) < len(l) {
		t.Errorf("grown L has %d vertices, want ≥ %d", len(grown), len(l))
	}
}

// TestOffsetClosedLoopAddsClosedLoop checks the Sketch wrapper adds a closed line loop.
func TestOffsetClosedLoopAddsClosedLoop(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	sq := []math.Point2{math.P2(0, 0), math.P2(2, 0), math.P2(2, 2), math.P2(0, 2)}
	ents := s.OffsetClosedLoop(sq, 0.5, 8)
	if len(ents) < 4 {
		t.Fatalf("offset loop produced %d entities, want a closed loop (≥4)", len(ents))
	}
	for _, e := range ents {
		if _, ok := e.(*Line); !ok {
			t.Errorf("offset loop entity is %T, want *Line", e)
		}
	}
}
