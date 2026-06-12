// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"math"
	"testing"

	gmath "oblikovati.org/math"
)

// TestTrimLineAtEllipseCrossings: an ellipse is a first-class trim boundary —
// the chord inside it is removed at the true (±major-radius) crossings
// (M06-F12, #627; registry row "trim/extend against curves").
func TestTrimLineAtEllipseCrossings(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(-5, 0), gmath.P2(5, 0))
	s.Ellipses().Add(gmath.P2(0, 0), gmath.V2(1, 0), 2, 1) // crosses the axis at (±2, 0)
	parts, err := s.TrimLine(l, gmath.P2(0, 0))
	if err != nil {
		t.Fatalf("TrimLine: %v", err)
	}
	if len(parts) != 2 {
		t.Fatalf("trim left %d parts, want 2 (the two outside stubs)", len(parts))
	}
	nearX(t, l.B.Position(), -2)
}

// TestTrimLineAtSplineCrossing: a fit spline cuts the line where the true
// NURBS curve crosses it — not where a coarse approximation would.
func TestTrimLineAtSplineCrossing(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(-3, 0), gmath.P2(3, 0))
	// A vertical zigzag spline crossing y=0 once, near x=0 (its middle point).
	s.Splines().AddByPoints([]gmath.Point2{
		gmath.P2(0, -2), gmath.P2(0.2, 0), gmath.P2(0, 2),
	}, false)
	parts, err := s.TrimLine(l, gmath.P2(2, 0)) // pick right of the crossing
	if err != nil {
		t.Fatalf("TrimLine: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("trim left %d parts, want 1", len(parts))
	}
	// The reshaped line ends on the spline: the kept part runs [-3, x*] with
	// x* the true crossing, which interpolation places at its middle fit
	// point (0.2, 0).
	if math.Abs(float64(l.B.Position().X)-0.2) > 1e-6 {
		t.Errorf("trimmed end X = %v, want the spline crossing at 0.2", l.B.Position().X)
	}
}

// TestExtendLineToEllipse: extend reaches past the line's end to the
// ellipse's true boundary.
func TestExtendLineToEllipse(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	s.Ellipses().Add(gmath.P2(0, 0), gmath.V2(1, 0), 3, 1.5)
	if _, err := s.ExtendLine(l, true); err != nil {
		t.Fatalf("ExtendLine: %v", err)
	}
	nearX(t, l.B.Position(), 3)
}

// TestTrimCircleAtSplineCrossings: a spline can open a circle — the trim uses
// the circle↔spline crossings (registry's last missing curve-pair class).
func TestTrimCircleAtSplineCrossings(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	c := s.Circles().AddByCenterRadius(gmath.P2(0, 0), 1)
	// A vertical fit spline through x=0 crossing the circle at (0, ±1).
	s.Splines().AddByPoints([]gmath.Point2{
		gmath.P2(0, -2), gmath.P2(0, 0), gmath.P2(0, 2),
	}, false)
	parts, err := s.TrimCircle(c, gmath.P2(-1, 0)) // remove the left half
	if err != nil {
		t.Fatalf("TrimCircle: %v", err)
	}
	if len(parts) != 1 {
		t.Fatalf("trim left %d parts, want the complementary arc", len(parts))
	}
	arc, ok := parts[0].(*Arc)
	if !ok {
		t.Fatalf("trim result is %T, want *Arc", parts[0])
	}
	mid := float64(arc.Start.Position().Y + arc.End.Position().Y)
	if math.Abs(mid) > 1e-6 {
		t.Errorf("kept arc endpoints %v/%v, want (0,±1)", arc.Start.Position(), arc.End.Position())
	}
}
