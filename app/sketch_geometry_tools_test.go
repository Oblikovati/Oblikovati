// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/math"
	"oblikovati/model/sketch"
)

// sketchSession returns a session in the sketch environment on the XY plane (the
// camera frames XY straight-on so a pixel maps to a sketch point).
func sketchSession(t *testing.T) (*Session, *sketch.Sketch) {
	t.Helper()
	s, def := emptyPartSession(t)
	sk := def.Sketches().Add(sketch.XYPlane())
	s.EnterSketch(sk)
	return s, sk
}

func TestCircleToolDrawsCircle(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewCircleTool())
	s.Click(100, 100) // center
	s.Click(140, 100) // radius point — auto-commits on the second click
	if sk.Circles().Count() != 1 {
		t.Fatalf("circle tool added %d circles, want 1", sk.Circles().Count())
	}
	if r := sk.Circles().Item(0).Radius; r <= 0 {
		t.Errorf("circle radius = %v, want > 0", r)
	}
}

func TestArcToolDrawsArc(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewArcTool())
	s.Click(60, 100)  // start
	s.Click(140, 100) // end
	s.Click(100, 60)  // through — auto-commits on the third click
	if sk.Arcs().Count() != 1 {
		t.Errorf("arc tool added %d arcs, want 1", sk.Arcs().Count())
	}
}

func TestArcToolRejectsCollinearPoints(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewArcTool())
	s.Click(40, 100)
	s.Click(160, 100)
	s.Click(100, 100) // all on the same horizontal line
	if err := s.OK(); err == nil {
		t.Error("collinear arc points should fail to commit")
	}
	if sk.Arcs().Count() != 0 {
		t.Errorf("collinear arc added %d arcs, want 0", sk.Arcs().Count())
	}
}

func TestPolygonToolDrawsClosedProfile(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewPolygonTool(6))
	s.Click(100, 100) // center
	s.Click(150, 100) // a vertex — auto-commits on the second click
	if sk.Lines().Count() != 6 {
		t.Errorf("hexagon added %d lines, want 6", sk.Lines().Count())
	}
	if p := sk.Profiles(); p.Count() != 1 || !p.Item(0).IsClosed() {
		t.Errorf("hexagon did not form one closed profile (count=%d)", p.Count())
	}
}

func TestPolygonToolDefaultsSidesWhenInvalid(t *testing.T) {
	if NewPolygonTool(2).Sides != 6 {
		t.Errorf("polygon with <3 sides should default to 6, got %d", NewPolygonTool(2).Sides)
	}
}

func TestEllipseToolDrawsEllipse(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewEllipseTool())
	s.Click(100, 100) // center
	s.Click(150, 100) // major endpoint
	s.Click(100, 130) // minor radius point — auto-commits on the third click
	if sk.Ellipses().Count() != 1 {
		t.Fatalf("ellipse tool added %d ellipses, want 1", sk.Ellipses().Count())
	}
	e := sk.Ellipses().Item(0)
	if e.MajorRadius <= 0 || e.MinorRadius <= 0 {
		t.Errorf("ellipse radii = (%v,%v), want both > 0", e.MajorRadius, e.MinorRadius)
	}
}

func TestSplineToolDrawsSpline(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewSplineTool())
	s.Click(40, 40)
	s.Click(90, 120)
	s.Click(150, 60)
	if err := s.OK(); err != nil {
		t.Fatalf("spline OK: %v", err)
	}
	if sk.Splines().Count() != 1 {
		t.Errorf("spline tool added %d splines, want 1", sk.Splines().Count())
	}
}

func TestSplineToolNeedsTwoPoints(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewSplineTool()
	s.StartTool(tool)
	s.Click(40, 40)
	if tool.CanCommit() {
		t.Error("spline should not commit with a single point")
	}
}

func TestPointToolPlacesPoint(t *testing.T) {
	s, sk := sketchSession(t)
	before := sk.Points().Count()
	s.StartTool(NewPointTool())
	s.Click(60, 60) // a single click auto-commits one point and deactivates
	if got := sk.Points().Count() - before; got != 1 {
		t.Errorf("point tool placed %d points, want 1", got)
	}
	if s.ActiveTool() != nil {
		t.Error("the point tool should deactivate after placing a point")
	}
}

func TestGeometryToolsNeedActiveSketch(t *testing.T) {
	// A tool committing with no active sketch errors rather than panicking.
	s, _ := emptyPartSession(t)
	tool := NewCircleTool()
	tool.pts = append(tool.pts, math.P2(0, 0), math.P2(1, 0))
	if err := tool.Commit(s); err == nil {
		t.Error("circle commit with no active sketch should error")
	}
}

func TestSketchToolAutoCommitsAndDeactivates(t *testing.T) {
	s, sk := sketchSession(t)
	// A line completes on its second click with no OK, then the tool deactivates.
	s.StartTool(NewLineTool())
	s.Click(40, 40)
	s.Click(80, 40)
	if sk.Lines().Count() != 1 {
		t.Errorf("auto-commit drew %d lines, want 1", sk.Lines().Count())
	}
	if s.ActiveTool() != nil {
		t.Error("the tool should deactivate after creating one shape")
	}
	// The tool is single-shot: re-activating is needed to draw another.
	s.StartTool(NewCircleTool())
	s.Click(100, 100)
	s.Click(140, 100)
	if sk.Circles().Count() != 1 || s.ActiveTool() != nil {
		t.Errorf("second tool: %d circles, active=%v, want 1 / inactive",
			sk.Circles().Count(), s.ActiveTool() != nil)
	}
}

func TestEscapeCancelsToolMidOperation(t *testing.T) {
	s, sk := sketchSession(t)
	s.StartTool(NewLineTool())
	s.Click(40, 40) // first point placed; operation in progress
	if err := s.PressKey(KeyEvent{Key: "Escape"}); err != nil {
		t.Fatalf("Escape: %v", err)
	}
	if s.ActiveTool() != nil {
		t.Error("Esc should cancel the active tool mid-operation")
	}
	if sk.Lines().Count() != 0 {
		t.Errorf("cancelled line left %d lines, want 0", sk.Lines().Count())
	}
}

func TestEscapeClearsSelectionWhenNoTool(t *testing.T) {
	s, sk := sketchSession(t)
	selectEntities(s, sk.Lines().AddByTwoPoints(math.P2(0, 0), math.P2(1, 0)))
	if s.Selection().Count() == 0 {
		t.Fatal("setup: expected a selection")
	}
	_ = s.PressKey(KeyEvent{Key: "Escape"})
	if s.Selection().Count() != 0 {
		t.Error("Esc with no tool should clear the selection")
	}
}
