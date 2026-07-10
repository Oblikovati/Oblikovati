// SPDX-License-Identifier: GPL-2.0-only

package sketch

import (
	"testing"

	"oblikovati.org/math"
)

// dirParallelTo reports whether line l's direction is parallel to (vx,vy) within tol (zero cross
// product).
func dirParallelTo(l *Line, vx, vy, tol float64) bool {
	d := l.Direction()
	return vecParallelTo(d, vx, vy, tol)
}

// vecParallelTo reports whether v is parallel to (vx,vy) within tol (zero cross product).
func vecParallelTo(v math.Vector2, vx, vy, tol float64) bool {
	cross := float64(v.X)*vy - float64(v.Y)*vx
	return cross < tol && cross > -tol
}

func mustEllipseOp(t *testing.T, e *Ellipse, major bool) AxisOperand {
	t.Helper()
	op, ok := EllipseAxisOf(e, major)
	if !ok {
		t.Fatal("ellipse is not an axis source")
	}
	return op
}

// TestEllipseParallelAlignsLineToMajorAxis: parallel(line, ellipse-major) rotates the line to
// the ellipse's major-axis direction; the minor selector uses the perpendicular axis (#1879). The
// ellipse's own rotation DOF is pinned horizontal so it is the LINE that moves (#1879 AC2).
func TestEllipseParallelAlignsLineToMajorAxis(t *testing.T) {
	for _, tc := range []struct {
		name         string
		major        bool
		wantX, wantY float64
	}{
		{"major", true, 1, 0},  // major axis along +X
		{"minor", false, 0, 1}, // minor axis is its left perpendicular, +Y
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSketches().Add(XYPlane())
			g := s.GeometricConstraints()
			ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 3, 1)
			g.AddEllipseHorizontal(mustEllipseOp(t, ell, true))         // pin the ellipse's rotation
			l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 1)) // slanted
			g.AddEllipseParallel(LineAxis(l), mustEllipseOp(t, ell, tc.major))
			g.AddFix(l.A)
			if r := s.Solve(); !r.Converged {
				t.Fatalf("solve did not converge: %+v", r)
			}
			if !dirParallelTo(l, tc.wantX, tc.wantY, solveTol) {
				t.Errorf("line direction = %v, want parallel to (%v,%v)", l.Direction(), tc.wantX, tc.wantY)
			}
		})
	}
}

// TestEllipsePerpendicularAlignsLine: perpendicular(line, ellipse-major) makes the line
// perpendicular to the major axis, with the ellipse's rotation pinned horizontal (#1879).
func TestEllipsePerpendicularAlignsLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 3, 1)
	g.AddEllipseHorizontal(mustEllipseOp(t, ell, true))
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 1))
	g.AddEllipsePerpendicular(LineAxis(l), mustEllipseOp(t, ell, true))
	g.AddFix(l.A)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	// Perpendicular to +X major axis → the line is vertical, parallel to (0,1).
	if !dirParallelTo(l, 0, 1, solveTol) {
		t.Errorf("line direction = %v, want vertical (perpendicular to the major axis)", l.Direction())
	}
}

// TestEllipseParallelRotatedEllipse grounds the axis direction against a rotated ellipse (major
// axis at 45°): a fixed 45° reference line pins the ellipse there (its orientation is now a DOF),
// then a slanted line aligns to that rotated axis — proving the operand reads the ellipse's actual
// direction, not an axis-aligned accident (#1879).
func TestEllipseParallelRotatedEllipse(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 1), 3, 1) // major axis along (1,1)
	ref := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 1))
	g.AddFix(ref.A)
	g.AddFix(ref.B)                                                  // ref is a fixed 45° datum
	g.AddEllipseParallel(LineAxis(ref), mustEllipseOp(t, ell, true)) // pin the ellipse at 45°
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0))      // horizontal, to be moved
	g.AddEllipseParallel(LineAxis(l), mustEllipseOp(t, ell, true))
	g.AddFix(l.A)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if !dirParallelTo(l, 1, 1, solveTol) {
		t.Errorf("line direction = %v, want parallel to the 45° major axis (1,1)", l.Direction())
	}
}

// TestEllipseCollinearThroughCentre: collinear(line, ellipse-major) makes the line parallel to
// the major axis AND pass through the ellipse centre, with the ellipse pinned (#1879).
func TestEllipseCollinearThroughCentre(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 3, 1)
	g.AddEllipseHorizontal(mustEllipseOp(t, ell, true))         // pin the ellipse's rotation
	l := s.Lines().AddByTwoPoints(math.P2(0, 2), math.P2(4, 3)) // off the X axis, slanted
	g.AddEllipseCollinear(LineAxis(l), mustEllipseOp(t, ell, true))
	g.AddFix(ell.Center) // pin the centre (a DOF) so the line is what moves
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if !dirParallelTo(l, 1, 0, solveTol) {
		t.Errorf("collinear line direction = %v, want horizontal", l.Direction())
	}
	// On the major-axis line through the origin ⇒ both endpoints at Y≈0.
	if y0, y1 := float64(l.A.Position().Y), float64(l.B.Position().Y); y0 > solveTol || y0 < -solveTol || y1 > solveTol || y1 < -solveTol {
		t.Errorf("collinear endpoints Y = %v, %v, want on the X axis (0)", y0, y1)
	}
}

// TestEllipseHorizontalRotatesMajorAxis: horizontal(ellipse-major) rotates a 45°-oriented ellipse
// until its major axis is horizontal — the core of #1879 AC2 (the ellipse turns, since the world
// axis is fixed). The minor selector makes the MINOR axis horizontal instead.
func TestEllipseHorizontalRotatesMajorAxis(t *testing.T) {
	for _, tc := range []struct {
		name         string
		major        bool
		wantX, wantY float64
	}{
		{"major", true, 1, 0},  // major axis becomes horizontal
		{"minor", false, 1, 0}, // minor axis becomes horizontal ⇒ major becomes vertical
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := NewSketches().Add(XYPlane())
			g := s.GeometricConstraints()
			ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 1), 3, 1) // 45°
			g.AddEllipseHorizontal(mustEllipseOp(t, ell, tc.major))
			if r := s.Solve(); !r.Converged {
				t.Fatalf("solve did not converge: %+v", r)
			}
			axis := ellipseAxisVectorFor(ell, tc.major)
			if !vecParallelTo(axis, tc.wantX, tc.wantY, solveTol) {
				t.Errorf("selected axis = %v, want horizontal (%v,%v)", axis, tc.wantX, tc.wantY)
			}
		})
	}
}

// TestEllipseVerticalRotatesMajorAxis: vertical(ellipse-major) rotates the ellipse until its major
// axis is vertical (#1879 AC2). The ellipse starts at 45° (a non-degenerate angle): an ellipse
// EXACTLY perpendicular to the target sits at the sine residual's zero-gradient dead point, which
// the iterative solver cannot escape — the same limitation the existing perpendicular/parallel
// constraints have on exactly-parallel lines, and which real (non-exact) geometry never hits.
func TestEllipseVerticalRotatesMajorAxis(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 1), 3, 1) // 45°, off the dead point
	g.AddEllipseVertical(mustEllipseOp(t, ell, true))
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if !vecParallelTo(ell.MajorAxis, 0, 1, solveTol) {
		t.Errorf("major axis = %v, want vertical (0,1)", ell.MajorAxis)
	}
}

// TestEllipticalArcHorizontal: an elliptical arc is an ellipse-axis source too, so its axis can be
// made horizontal by rotation (#1879 AC2).
func TestEllipticalArcHorizontal(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ea := s.EllipticalArcs().Add(math.P2(0, 0), math.V2(1, 1), 3, 1, 0, 1) // 45°
	op, ok := EllipseAxisOf(ea, true)
	if !ok {
		t.Fatal("elliptical arc is not an axis source")
	}
	g.AddEllipseHorizontal(op)
	if r := s.Solve(); !r.Converged {
		t.Fatalf("solve did not converge: %+v", r)
	}
	if !vecParallelTo(ea.MajorAxis, 1, 0, solveTol) {
		t.Errorf("arc major axis = %v, want horizontal (1,0)", ea.MajorAxis)
	}
}

// ellipseAxisVectorFor returns the ellipse's major axis, or its minor axis (a quarter turn on).
func ellipseAxisVectorFor(e *Ellipse, major bool) math.Vector2 {
	if major {
		return e.MajorAxis
	}
	return math.V2(-e.MajorAxis.Y, e.MajorAxis.X)
}

// TestEllipseAxisOfRejectsNonEllipse: a line is not an ellipse-axis source (#1879).
func TestEllipseAxisOfRejectsNonEllipse(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	if _, ok := EllipseAxisOf(l, true); ok {
		t.Error("a line must not resolve as an ellipse-axis operand")
	}
}
