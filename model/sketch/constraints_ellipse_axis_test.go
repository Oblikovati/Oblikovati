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
	cross := float64(d.X)*vy - float64(d.Y)*vx
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
// the ellipse's major-axis direction; the minor selector uses the perpendicular axis (#1879).
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
// perpendicular to the major axis (#1879).
func TestEllipsePerpendicularAlignsLine(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 3, 1)
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
// axis at 45°), so the residual is not accidentally satisfied by an axis-aligned ellipse (#1879).
func TestEllipseParallelRotatedEllipse(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 1), 3, 1) // major axis along (1,1)
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(4, 0)) // horizontal
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
// the major axis AND pass through the ellipse centre (#1879).
func TestEllipseCollinearThroughCentre(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	g := s.GeometricConstraints()
	ell := s.Ellipses().Add(math.P2(0, 0), math.V2(1, 0), 3, 1)
	l := s.Lines().AddByTwoPoints(math.P2(0, 2), math.P2(4, 3)) // off the X axis, slanted
	g.AddEllipseCollinear(LineAxis(l), mustEllipseOp(t, ell, true))
	g.AddFix(ell.Center) // pin the ellipse (its centre is a DOF) so the line is what moves
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

// TestEllipseAxisOfRejectsNonEllipse: a line is not an ellipse-axis source (#1879).
func TestEllipseAxisOfRejectsNonEllipse(t *testing.T) {
	s := NewSketches().Add(XYPlane())
	l := s.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0))
	if _, ok := EllipseAxisOf(l, true); ok {
		t.Error("a line must not resolve as an ellipse-axis operand")
	}
}
