// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati/math"
)

// nearX asserts a point's X coordinate.
func nearX(t *testing.T, p gmath.Point2, want float64) {
	t.Helper()
	if math.Abs(float64(p.X)-want) > 1e-9 {
		t.Errorf("X = %v, want %v", p.X, want)
	}
}

// A line crossing a circle is trimmed at the two chord crossings: picking the segment
// inside the circle removes it and leaves the two outside stubs.
func TestTrimLineAtCircleCrossings(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(-3, 0), gmath.P2(3, 0))
	s.Circles().AddByCenterRadius(gmath.P2(0, 0), 1) // crosses at (±1,0)
	parts, err := s.TrimLine(l, gmath.P2(0, 0))      // pick the chord inside the circle
	if err != nil {
		t.Fatalf("TrimLine: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("trim left %d parts, want 2 (the two outside stubs)", len(parts))
	}
	nearX(t, l.B.Position(), -1) // reshaped original keeps [-3,-1]
}

// A line trimmed at an arc crossing: only the crossing within the arc's sweep cuts the
// line (the other circle crossing is outside the sweep and ignored).
func TestTrimLineAtArcCrossingHonorsSweep(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(-3, 0), gmath.P2(3, 0))
	// Left semicircle (angles π/2→3π/2) crosses the line only at (-1,0); (1,0) is outside.
	s.Arcs().AddByCenterStartEnd(gmath.P2(0, 0), gmath.P2(0, 1), gmath.P2(0, -1), true)
	parts, err := s.TrimLine(l, gmath.P2(0, 0)) // pick right of the single crossing at (-1,0)
	if err != nil {
		t.Fatalf("TrimLine: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("trim left %d parts, want 1 (single crossing)", len(parts))
	}
	nearX(t, l.B.Position(), -1) // pick is right of the crossing → tail removed, keeps [-3,-1]
}

// A short line extends past its B end to the nearest crossing of its support with a circle.
func TestExtendLineToCircle(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(-3, 0), gmath.P2(-2, 0)) // B end at x=-2
	s.Circles().AddByCenterRadius(gmath.P2(0, 0), 1)                // support crosses at (±1,0)
	if _, err := s.ExtendLine(l, true); err != nil {                // extend the B end
		t.Fatalf("ExtendLine: %v", err)
	}
	nearX(t, l.B.Position(), -1) // nearest crossing beyond B is (-1,0)
}

// Extending toward an arc reaches only a crossing within the arc's sweep.
func TestExtendLineToArc(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(-3, 0), gmath.P2(-2, 0))
	// Left semicircle: support crossing at (-1,0) is in-sweep, (1,0) is not.
	s.Arcs().AddByCenterStartEnd(gmath.P2(0, 0), gmath.P2(0, 1), gmath.P2(0, -1), true)
	if _, err := s.ExtendLine(l, true); err != nil {
		t.Fatalf("ExtendLine: %v", err)
	}
	nearX(t, l.B.Position(), -1)
}

// Extending with no reachable geometry beyond the end errors (no line/curve to reach).
func TestExtendLineNoTargetErrors(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	s.Circles().AddByCenterRadius(gmath.P2(0, 5), 1) // off the support line entirely
	if _, err := s.ExtendLine(l, true); err == nil {
		t.Error("extend should fail when nothing lies beyond the picked end")
	}
}
