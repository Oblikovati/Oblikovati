// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/math"
	"oblikovati.org/model/sketch"
)

// TestSketch3DSplineToolDrawsFitSpline drives the ribbon command: place three
// points, commit, and the active 3D sketch holds an interpolating spline.
func TestSketch3DSplineToolDrawsFitSpline(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	if err := s.Execute("Sketch3D.Spline"); err != nil {
		t.Fatalf("Sketch3D.Spline: %v", err)
	}
	tool, ok := s.ActiveTool().Tool().(*Spline3DTool)
	if !ok {
		t.Fatalf("active tool = %T, want *Spline3DTool", s.ActiveTool().Tool())
	}
	tool.AddPoint(math.P3(0, 0, 0))
	if tool.CanCommit() {
		t.Fatal("one point must not be committable")
	}
	tool.AddPoint(math.P3(1, 2, 0))
	tool.AddPoint(math.P3(3, 1, 1))
	if err := s.OK(); err != nil {
		t.Fatalf("commit spline: %v", err)
	}
	sp, ok := sk.Entities()[0].(*sketch.Spline3D)
	if !ok || !sp.IsFitType() || sp.PointCount() != 3 {
		t.Errorf("entity = %T (%+v), want a 3-point fit Spline3D", sk.Entities()[0], sp)
	}
}

// TestSketch3DControlPointSplineToolDrawsControlSpline covers the control-polygon
// variant, including the Closed dialog flag.
func TestSketch3DControlPointSplineToolDrawsControlSpline(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	if err := s.Execute("Sketch3D.ControlPointSpline"); err != nil {
		t.Fatalf("Sketch3D.ControlPointSpline: %v", err)
	}
	tool := s.ActiveTool().Tool().(*Spline3DTool)
	tool.AddPoint(math.P3(0, 0, 0))
	tool.AddPoint(math.P3(1, 0, 2))
	tool.AddPoint(math.P3(2, 0, 0))
	tool.Params().Bools[0].Set(true) // Closed
	if err := s.OK(); err != nil {
		t.Fatalf("commit control spline: %v", err)
	}
	sp, ok := sk.Entities()[0].(*sketch.Spline3D)
	if !ok || sp.IsFitType() || !sp.Closed {
		t.Errorf("entity = %T (%+v), want a closed control Spline3D", sk.Entities()[0], sp)
	}
}

// TestSketch3DEquationCurveToolDrawsCurve fills the dialog expressions and range
// and commits a parametric curve.
func TestSketch3DEquationCurveToolDrawsCurve(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	if err := s.Execute("Sketch3D.EquationCurve"); err != nil {
		t.Fatalf("Sketch3D.EquationCurve: %v", err)
	}
	tool := s.ActiveTool().Tool().(*EquationCurve3DTool)
	if tool.CanCommit() {
		t.Fatal("empty expressions must not be committable")
	}
	p := tool.Params()
	p.Texts[0].Set("cos(t)")
	p.Texts[1].Set("sin(t)")
	p.Texts[2].Set("t")
	p.Floats[1].Set(6.28)
	if err := s.OK(); err != nil {
		t.Fatalf("commit equation curve: %v", err)
	}
	eq, ok := sk.Entities()[0].(*sketch.EquationCurve3D)
	if !ok || eq.YExpr != "sin(t)" {
		t.Errorf("entity = %T, want an EquationCurve3D with y=sin(t)", sk.Entities()[0])
	}
}

// TestSketch3DEquationCurveToolRejectsBadExpression keeps the tool open when the
// kernel rejects an expression, instead of silently dropping the input.
func TestSketch3DEquationCurveToolRejectsBadExpression(t *testing.T) {
	t.Parallel()
	s, sk := sketch3DSession(t)
	if err := s.Execute("Sketch3D.EquationCurve"); err != nil {
		t.Fatalf("Sketch3D.EquationCurve: %v", err)
	}
	tool := s.ActiveTool().Tool().(*EquationCurve3DTool)
	p := tool.Params()
	p.Texts[0].Set("@@")
	p.Texts[1].Set("t")
	p.Texts[2].Set("t")
	if err := s.OK(); err == nil {
		t.Fatal("expected the bad x(t) expression to fail the commit")
	}
	if len(sk.Entities()) != 0 {
		t.Errorf("entities = %d, want none after a rejected expression", len(sk.Entities()))
	}
}
