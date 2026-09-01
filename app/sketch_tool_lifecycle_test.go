// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// allSketchTools is every geometry tool, for table-driven lifecycle coverage.
func allSketchTools() []Tool {
	return []Tool{
		NewLineTool(), NewRectangleTool(), NewCircleTool(), NewArcTool(),
		NewSplineTool(), NewEllipseTool(), NewPolygonTool(5), NewPointTool(),
	}
}

func TestSketchToolsLifecycleAndPrompts(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	for _, tool := range allSketchTools() {
		if tool.Name() == "" {
			t.Errorf("%T has an empty name", tool)
		}
		tool.Start(s)
		tool.Pick(s, nil) // geometry tools ignore entity picks
		// A fresh tool prompts for its first input and cannot commit yet.
		if p, ok := tool.(Prompted); ok && p.Prompt(s) == "" {
			t.Errorf("%s gave an empty initial prompt", tool.Name())
		}
		if tool.CanCommit() {
			t.Errorf("%s should not commit before any input", tool.Name())
		}
		tool.Cancel(s)
	}
}

func TestSketchToolPromptsAdvanceWithClicks(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	cases := []struct {
		tool   Tool
		clicks int
	}{
		{NewCircleTool(), 2},
		{NewArcTool(), 3},
		{NewEllipseTool(), 3},
		{NewPolygonTool(6), 2},
	}
	for _, c := range cases {
		s.StartTool(c.tool)
		prev := promptOf(s, c.tool)
		for i := 0; i < c.clicks; i++ {
			s.Click(float64(50+i*30), float64(60+i*20))
			cur := promptOf(s, c.tool)
			if cur == "" {
				t.Errorf("%s prompt empty after %d clicks", c.tool.Name(), i+1)
			}
			prev = cur
		}
		if !c.tool.CanCommit() {
			t.Errorf("%s should be ready after %d clicks", c.tool.Name(), c.clicks)
		}
		_ = prev
		s.CancelTool()
	}
}

// promptOf returns a tool's current prompt, or "" if it provides none.
func promptOf(s *Session, t Tool) string {
	if p, ok := t.(Prompted); ok {
		return p.Prompt(s)
	}
	return ""
}

func TestPointAndSplineFinalPrompts(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	pt := NewPointTool()
	s.StartTool(pt)
	s.Click(40, 40)
	if pt.Prompt(s) == "" {
		t.Error("point tool prompt empty after a click")
	}
	sp := NewSplineTool()
	s.StartTool(sp)
	s.Click(40, 40)
	s.Click(80, 80)
	if sp.Prompt(s) == "" {
		t.Error("spline tool prompt empty after two clicks")
	}
}

func TestLineAndRectanglePromptsAndCancel(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	line := NewLineTool()
	s.StartTool(line)
	if line.Prompt(s) == "" {
		t.Error("line initial prompt empty")
	}
	s.Click(10, 10)
	s.Click(50, 50)
	if line.Prompt(s) == "" {
		t.Error("line final prompt empty")
	}
	line.Cancel(s)

	rect := NewRectangleTool()
	s.StartTool(rect)
	s.Click(10, 10)
	if rect.Prompt(s) == "" {
		t.Error("rectangle mid prompt empty")
	}
	s.Click(60, 60)
	if rect.Prompt(s) == "" {
		t.Error("rectangle final prompt empty")
	}
	rect.Cancel(s)
}

func TestViewCommandsExecute(t *testing.T) {
	t.Parallel()
	s := registeredSession(t)
	if err := s.Execute("View.ZoomAll"); err != nil {
		t.Errorf("Zoom All: %v", err)
	}
	if err := s.Execute("View.Home"); err != nil {
		t.Errorf("Home: %v", err)
	}
}

func TestCreateSketchOnEachOriginPlane(t *testing.T) {
	t.Parallel()
	for _, p := range []OriginPlane{OriginXY, OriginXZ, OriginYZ} {
		s, _ := emptyPartSession(t)
		if _, err := s.CreateSketchOnOrigin(p); err != nil {
			t.Errorf("CreateSketch on origin %d: %v", p, err)
		}
	}
}
