// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// TestDragSolvePullsCoincidentPoint checks that dragging one of two coincident points re-solves
// the sketch so the other point follows — the constraint stays satisfied, unlike a blind translate.
func TestDragSolvePullsCoincidentPoint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(1, 1))
	b := s.Points().Add(math.P2(1, 1))
	co := s.GeometricConstraints().AddCoincident(a, b)

	s.DragSolve([]PinTarget{{P: a, Target: math.P2(4, 3)}})

	if !a.Position().IsEqualTo(math.P2(4, 3), 1e-6) {
		t.Errorf("dragged point a = %v, want it pulled to (4,3)", a.Position())
	}
	if !b.Position().IsEqualTo(a.Position(), 1e-6) {
		t.Errorf("coincident point b = %v, should follow a to %v", b.Position(), a.Position())
	}
	for _, r := range co.Residuals() {
		if r > 1e-6 || r < -1e-6 {
			t.Errorf("coincident constraint violated after drag (residual %v)", r)
		}
	}
}

// TestDragSolveRespectsHorizontalConstraint checks a horizontal line dragged by one endpoint
// keeps both endpoints at the same Y (the constraint slides the other endpoint), so the line
// follows the cursor within its remaining freedom rather than tilting.
func TestDragSolveRespectsHorizontalConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	a := s.Points().Add(math.P2(0, 0))
	b := s.Points().Add(math.P2(4, 0))
	s.GeometricConstraints().AddHorizontal(a, b)

	// Drag endpoint b up and to the right; horizontal forces a's Y to track b's Y.
	s.DragSolve([]PinTarget{{P: b, Target: math.P2(6, 2)}})

	if d := b.Position().Y - a.Position().Y; d > 1e-6 || d < -1e-6 {
		t.Errorf("horizontal violated: a.Y=%v b.Y=%v should match", a.Position().Y, b.Position().Y)
	}
	if b.Position().X <= 4 {
		t.Errorf("dragged endpoint did not follow the cursor in +x: %v", b.Position())
	}
}

// TestDragSolveFreePointMovesToTarget checks the unconstrained case still lands the point exactly
// on the cursor (so the existing free-drag behaviour is preserved by the solve path).
func TestDragSolveFreePointMovesToTarget(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.Points().Add(math.P2(2, 2))
	s.DragSolve([]PinTarget{{P: p, Target: math.P2(-1, 5)}})
	if !p.Position().IsEqualTo(math.P2(-1, 5), 1e-6) {
		t.Errorf("free point drag = %v, want exactly (-1,5)", p.Position())
	}
}

// TestDragKeepsOriginCoincidentEndpointPinned is the #2160 regression: an endpoint coincident to a
// fixed reference anchor (the projected origin) must stay on that anchor through a drag. A drag pin
// is an equal-weight positional residual, so before the fix a body drag pinned that endpoint to the
// cursor and the least-squares solve split the difference against the coincidence — the endpoint
// slid to the midpoint and "dragged off the origin". It must now stay put while the free end follows.
func TestDragKeepsOriginCoincidentEndpointPinned(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	origin := s.newRefPoint(math.P2(0, 0)) // a fixed anchor, like the projected origin
	line := s.Lines().AddByTwoPoints(math.P2(2, 0), math.P2(4, 0))
	s.GeometricConstraints().AddCoincident(line.A, origin)
	s.Solve() // line.A snaps onto the origin

	if !line.A.Position().IsEqualTo(math.P2(0, 0), 1e-6) {
		t.Fatalf("setup: line.A should sit on the origin after solve, got %v", line.A.Position())
	}

	// Body drag: both endpoints pinned to a cursor offset up and to the right.
	s.DragSolve([]PinTarget{
		{P: line.A, Target: math.P2(3, 1)},
		{P: line.B, Target: math.P2(5, 1)},
	})

	if !line.A.Position().IsEqualTo(math.P2(0, 0), 1e-6) {
		t.Errorf("origin-coincident endpoint drifted to %v; the coincidence must hold it at the origin (#2160)", line.A.Position())
	}
	if !line.B.Position().IsEqualTo(math.P2(5, 1), 1e-6) {
		t.Errorf("free endpoint = %v, should follow the drag to (5,1)", line.B.Position())
	}
}

// TestGroundedPointsFloodsCoincidenceChain checks the grounded flag reaches a point coincident to a
// point that is itself coincident to the origin (transitive), while an unconstrained point stays free.
func TestGroundedPointsFloodsCoincidenceChain(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	origin := s.newRefPoint(math.P2(0, 0))
	p := s.Points().Add(math.P2(1, 1))
	q := s.Points().Add(math.P2(2, 2))
	s.GeometricConstraints().AddCoincident(p, origin) // p grounded directly
	s.GeometricConstraints().AddCoincident(q, p)      // q grounded through p
	free := s.Points().Add(math.P2(9, 9))

	g := s.groundedPoints()
	if !g[origin] || !g[p] || !g[q] {
		t.Errorf("origin, p and q should all be grounded; got origin=%v p=%v q=%v", g[origin], g[p], g[q])
	}
	if g[free] {
		t.Error("an unconstrained point must not be grounded")
	}
}

// TestGroundedPointsIncludesFixConstraint checks a Fix constraint grounds its point, so a drag does
// not tug a user-fixed point off its position.
func TestGroundedPointsIncludesFixConstraint(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	p := s.Points().Add(math.P2(3, 3))
	s.GeometricConstraints().AddFix(p)
	if !s.groundedPoints()[p] {
		t.Error("a Fix-constrained point must be grounded")
	}
}

// TestDragSoftPinYieldsToDimension is the safety property behind allowing dimension-determined
// entities to be dragged (#2160, symptom A): the drag pin is soft, so it must not stretch a hard
// dimension. A point fixed 2 units from a grounded anchor, dragged out to 10, must stay ~2 away —
// the dimension wins. With an equal-weight pin it would drift toward the midpoint and violate it.
func TestDragSoftPinYieldsToDimension(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	anchor := s.Points().Add(math.P2(0, 0))
	s.GeometricConstraints().AddFix(anchor)
	p := s.Points().Add(math.P2(2, 0))
	s.GeometricConstraints().AddHorizontal(anchor, p)
	if _, err := s.DimensionConstraints().AddDistance(anchor, p, "2 cm"); err != nil {
		t.Fatalf("AddDistance: %v", err)
	}
	s.Solve()

	s.DragSolve([]PinTarget{{P: p, Target: math.P2(10, 0)}})

	if d := float64(p.Position().DistanceTo(anchor.Position())); d < 1.9 || d > 2.1 {
		t.Errorf("drag stretched the distance dimension: |p-anchor| = %.4f, want ~2 (soft pin must yield)", d)
	}
}

func TestDefiningPointsByEntity(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	pt := s.Points().Add(math.P2(0, 0))
	ln := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	ci := s.Circles().AddByCenterRadius(math.P2(0, 0), 1)
	ar := s.Arcs().AddByCenterStartEnd(math.P2(0, 0), math.P2(1, 0), math.P2(0, 1), true)
	el := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 2, 1)
	ea := s.EllipticalArcs().Add(math.P2(0, 0), math.V2(1, 0), 2, 1, 0, 1)

	cases := []struct {
		name string
		ent  Entity
		want int
	}{
		{"point", pt, 1},
		{"line", ln, 2},
		{"circle", ci, 1},
		{"arc", ar, 3},
		{"ellipse", el, 1},
		{"ellipticalArc", ea, 1},
		{"unknown", nil, 0},
	}
	for _, c := range cases {
		if got := DefiningPoints(c.ent); len(got) != c.want {
			t.Errorf("DefiningPoints(%s) = %d points, want %d", c.name, len(got), c.want)
		}
	}
	if got := DefiningPoints(pt); got[0] != pt {
		t.Error("DefiningPoints(point) should return the point itself")
	}
}
