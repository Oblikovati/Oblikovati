// SPDX-License-Identifier: GPL-2.0-only

package geom

import (
	stdmath "math"
	"testing"

	"github.com/Oblikovati/oblikovati/math"
)

// nearPoint2 reports whether two 2D points coincide within a geometric tolerance.
func nearPoint2(p, q math.Point2) bool { return p.IsEqualTo(q, 1e-9) }

// onCircle asserts each point lies on the circle (center distance == radius).
func onCircle(t *testing.T, pts []math.Point2, c Circle2d) {
	t.Helper()
	for _, p := range pts {
		if !near(p.DistanceTo(c.Center), c.Radius) {
			t.Errorf("point %v not on circle (dist %g, r %g)", p, p.DistanceTo(c.Center), c.Radius)
		}
	}
}

func mustLine2d(t *testing.T, ox, oy, dx, dy float64) Line2d {
	t.Helper()
	l, err := NewLine2d(math.P2(math.Scalar(ox), math.Scalar(oy)), math.V2(math.Scalar(dx), math.Scalar(dy)))
	if err != nil {
		t.Fatalf("NewLine2d: %v", err)
	}
	return l
}

// --- LineCircle2dIntersection ---------------------------------------------

func TestLineCircleTwoPoints(t *testing.T) {
	c := NewCircle2d(math.P2(0, 0), 2)
	pts := LineCircle2dIntersection(mustLine2d(t, 0, 0, 1, 0), c, 0)
	if len(pts) != 2 {
		t.Fatalf("got %d points, want 2", len(pts))
	}
	onCircle(t, pts, c)
}

func TestLineCircleTangentIsOnePoint(t *testing.T) {
	c := NewCircle2d(math.P2(0, 0), 2)
	pts := LineCircle2dIntersection(mustLine2d(t, 0, 2, 1, 0), c, 0) // y = 2, tangent at top
	if len(pts) != 1 {
		t.Fatalf("got %d points, want 1 (tangent)", len(pts))
	}
	if !nearPoint2(pts[0], math.P2(0, 2)) {
		t.Errorf("tangent point = %v, want (0,2)", pts[0])
	}
}

func TestLineCircleMissesIsNone(t *testing.T) {
	c := NewCircle2d(math.P2(0, 0), 2)
	if pts := LineCircle2dIntersection(mustLine2d(t, 0, 3, 1, 0), c, 0); len(pts) != 0 {
		t.Errorf("got %d points, want 0 (miss)", len(pts))
	}
}

// --- SegmentCircle2dIntersection ------------------------------------------

func TestSegmentCircleFiltersToExtent(t *testing.T) {
	c := NewCircle2d(math.P2(0, 0), 2)
	// Segment from (0,0) to (3,0) crosses the circle only at (2,0); (-2,0) is off-segment.
	seg := NewLineSegment2d(math.P2(0, 0), math.P2(3, 0))
	pts := SegmentCircle2dIntersection(seg, c, 0)
	if len(pts) != 1 || !nearPoint2(pts[0], math.P2(2, 0)) {
		t.Fatalf("got %v, want one point (2,0)", pts)
	}
}

func TestSegmentCircleDegenerateIsNone(t *testing.T) {
	c := NewCircle2d(math.P2(0, 0), 2)
	seg := NewLineSegment2d(math.P2(1, 1), math.P2(1, 1))
	if pts := SegmentCircle2dIntersection(seg, c, 0); len(pts) != 0 {
		t.Errorf("degenerate segment gave %d points, want 0", len(pts))
	}
}

// --- Circle2dCircle2dIntersection -----------------------------------------

func TestCircleCircleTwoPoints(t *testing.T) {
	c1 := NewCircle2d(math.P2(0, 0), 2)
	c2 := NewCircle2d(math.P2(3, 0), 2)
	pts := Circle2dCircle2dIntersection(c1, c2, 0)
	if len(pts) != 2 {
		t.Fatalf("got %d points, want 2", len(pts))
	}
	onCircle(t, pts, c1)
	onCircle(t, pts, c2)
	h := stdmath.Sqrt(2*2 - 1.5*1.5)
	for _, p := range pts {
		if !near(float64(p.X), 1.5) || !near(stdmath.Abs(float64(p.Y)), h) {
			t.Errorf("point %v, want x=1.5 y=±%g", p, h)
		}
	}
}

func TestCircleCircleTangentExternal(t *testing.T) {
	c1 := NewCircle2d(math.P2(0, 0), 2)
	c2 := NewCircle2d(math.P2(4, 0), 2) // d = r1+r2
	pts := Circle2dCircle2dIntersection(c1, c2, 1e-6)
	if len(pts) != 1 || !nearPoint2(pts[0], math.P2(2, 0)) {
		t.Fatalf("got %v, want one tangent point (2,0)", pts)
	}
}

func TestCircleCircleDisjointConcentricNested(t *testing.T) {
	c1 := NewCircle2d(math.P2(0, 0), 2)
	cases := map[string]Circle2d{
		"disjoint":   NewCircle2d(math.P2(5, 0), 2),
		"concentric": NewCircle2d(math.P2(0, 0), 1),
		"nested":     NewCircle2d(math.P2(0.5, 0), 0.5), // wholly inside c1, no contact
	}
	for name, c2 := range cases {
		if pts := Circle2dCircle2dIntersection(c1, c2, 0); len(pts) != 0 {
			t.Errorf("%s: got %d points, want 0", name, len(pts))
		}
	}
}

// --- Arc2d.ContainsAngle / ContainsPoint ----------------------------------

func TestArcContainsAnglePositiveSweep(t *testing.T) {
	a := NewArc2d(math.P2(0, 0), 1, 0, stdmath.Pi) // upper half, CCW
	if !a.ContainsAngle(stdmath.Pi/2, 0) {
		t.Error("π/2 should be on the upper-half arc")
	}
	if a.ContainsAngle(-stdmath.Pi/2, 0) {
		t.Error("−π/2 should NOT be on the upper-half arc")
	}
}

func TestArcContainsAngleNegativeSweep(t *testing.T) {
	a := NewArc2d(math.P2(0, 0), 1, 0, -stdmath.Pi) // lower half, CW
	if !a.ContainsAngle(-stdmath.Pi/2, 0) {
		t.Error("−π/2 should be on the lower-half arc")
	}
	if a.ContainsAngle(stdmath.Pi/2, 0) {
		t.Error("π/2 should NOT be on the lower-half arc")
	}
}

func TestArcContainsPointFiltersCircleCrossings(t *testing.T) {
	a := NewArc2d(math.P2(0, 0), 1, 0, stdmath.Pi) // upper half
	if !a.ContainsPoint(math.P2(0, 1), 0) {
		t.Error("(0,1) should be on the upper-half arc")
	}
	if a.ContainsPoint(math.P2(0, -1), 0) {
		t.Error("(0,-1) should NOT be on the upper-half arc")
	}
}

func TestArcContainsGuardsNegativeTolAndZeroRadius(t *testing.T) {
	a := NewArc2d(math.P2(0, 0), 1, 0, stdmath.Pi)
	// Negative tol is clamped to 0 (no slack), so an in-sweep angle still passes.
	if !a.ContainsAngle(stdmath.Pi/2, -1) {
		t.Error("negative tol should be treated as 0, not reject an interior angle")
	}
	// Zero-radius arc: ContainsPoint must not divide by zero; it falls back to 0 slack.
	z := NewArc2d(math.P2(0, 0), 0, 0, stdmath.Pi)
	if z.ContainsPoint(math.P2(0, -1), -1) {
		t.Error("(0,-1) is below a zero-radius upper-half arc; should not be contained")
	}
}

func TestArcContainsAngleFullCircle(t *testing.T) {
	a := NewArc2d(math.P2(0, 0), 1, 0, twoPi)
	for _, th := range []float64{0, stdmath.Pi / 2, stdmath.Pi, -stdmath.Pi / 2, 3} {
		if !a.ContainsAngle(th, 0) {
			t.Errorf("full circle should contain angle %g", th)
		}
	}
}
