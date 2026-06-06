// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	stdmath "math"
	"testing"

	gmath "oblikovati/math"
)

// near2 asserts a 2D point within tolerance.
func near2(t *testing.T, got gmath.Point2, x, y float64) {
	t.Helper()
	if !got.IsEqualTo(gmath.P2(gmath.Scalar(x), gmath.Scalar(y)), 1e-9) {
		t.Errorf("point = %v, want (%v,%v)", got, x, y)
	}
}

func TestSketchMoveTool(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	tool := NewSketchMoveTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{})
	tool.SetVector(5, 0)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Fatal("move tool should deactivate after OK")
	}
	near2(t, l.A.Position(), 5, 0)
	near2(t, l.B.Position(), 6, 0)
}

func TestSketchCopyTool(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(1, 0))
	tool := NewSketchCopyTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{})
	tool.SetVector(0, 3)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sk.Lines().Count() != 2 {
		t.Fatalf("lines after copy = %d, want 2", sk.Lines().Count())
	}
	near2(t, l.A.Position(), 0, 0) // the original is untouched
}

func TestSketchRotateTool(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(gmath.P2(1, 0), gmath.P2(2, 0))
	tool := NewSketchRotateTool()
	s.StartTool(tool)
	tool.PickSnap(l, SnapResult{})
	tool.SetCenter(0, 0)
	tool.SetAngle(stdmath.Pi / 2)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	near2(t, l.A.Position(), 0, 1) // (1,0) rotated 90° about origin
	near2(t, l.B.Position(), 0, 2)
}

func TestSketchScaleTool(t *testing.T) {
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(gmath.P2(2, 0), 1)
	tool := NewSketchScaleTool()
	s.StartTool(tool)
	tool.PickSnap(c, SnapResult{})
	tool.SetCenter(0, 0)
	tool.SetFactor(2)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	near2(t, c.Center.Position(), 4, 0)
	if stdmath.Abs(float64(c.Radius)-2) > 1e-9 {
		t.Errorf("radius = %v, want 2", c.Radius)
	}
}

func TestSketchScaleToolRejectsNonPositive(t *testing.T) {
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(gmath.P2(2, 0), 1)
	tool := NewSketchScaleTool()
	s.StartTool(tool)
	tool.PickSnap(c, SnapResult{})
	tool.SetFactor(0)
	if err := s.OK(); err == nil {
		t.Error("scale with factor 0 should error")
	}
}

func TestSketchRectPatternTool(t *testing.T) {
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(gmath.P2(0, 0), 0.5)
	tool := NewSketchRectPatternTool()
	s.StartTool(tool)
	tool.PickSnap(c, SnapResult{})
	tool.SetStep1(2, 0)
	tool.SetCount1(3) // 3 columns, 1 row → 2 new copies
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sk.Circles().Count() != 3 {
		t.Fatalf("circles after rect pattern = %d, want 3", sk.Circles().Count())
	}
}

func TestSketchCircPatternTool(t *testing.T) {
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(gmath.P2(2, 0), 0.5)
	tool := NewSketchCircPatternTool()
	s.StartTool(tool)
	tool.PickSnap(c, SnapResult{})
	tool.SetCenter(0, 0)
	tool.SetCount(4)
	tool.SetTotalAngle(2 * stdmath.Pi)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if sk.Circles().Count() != 4 {
		t.Fatalf("circles after circular pattern = %d, want 4", sk.Circles().Count())
	}
}

func TestSketchStretchTool(t *testing.T) {
	s, sk := sketchSession(t)
	l := sk.Lines().AddByTwoPoints(gmath.P2(0, 0), gmath.P2(4, 0))
	tool := NewSketchStretchTool()
	s.StartTool(tool)
	tool.PickSnap(l.B, SnapResult{}) // pick only the B endpoint
	tool.SetVector(0, 3)
	if err := s.OK(); err != nil {
		t.Fatalf("OK: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Fatal("stretch tool should deactivate after OK")
	}
	near2(t, l.A.Position(), 0, 0) // A stays
	near2(t, l.B.Position(), 4, 3) // B stretched up
}

// A pattern that would make only one instance is not ready to commit.
func TestSketchRectPatternNeedsMoreThanOne(t *testing.T) {
	s, sk := sketchSession(t)
	c := sk.Circles().AddByCenterRadius(gmath.P2(0, 0), 0.5)
	tool := NewSketchRectPatternTool()
	s.StartTool(tool)
	tool.PickSnap(c, SnapResult{})
	tool.SetCount1(1)
	tool.SetCount2(1)
	if tool.CanCommit() {
		t.Error("a 1×1 rectangular pattern should not be committable")
	}
}
