// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"
)

// TestSketchRectangleToolDropsRectangle: the rectangle tool drops a sketch with a rectangle entity.
func TestSketchRectangleToolDropsRectangle(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewSketchRectangleTool()
	tool.Start(s)
	tool.SetPlacement(150, 150)
	tool.Pick(s, nil)
	tool.Cancel(s)
	if tool.Name() != "Sketch Rectangle" || !tool.CanCommit() {
		t.Fatalf("sketch-rectangle tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	ss := c.Sheets().Active().Sketches()
	if ss.Count() != 1 || ss.Item(0).EntityCount() != 1 || ss.Item(0).CurveCount() != 4 {
		t.Fatalf("rectangle sketch not added (sketches=%d)", ss.Count())
	}
}

// TestSketchCircleToolDropsCircle: the circle tool drops a sketch with a circle entity.
func TestSketchCircleToolDropsCircle(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewSketchCircleTool()
	tool.Start(s)
	tool.SetPlacement(120, 120)
	tool.Pick(s, nil)
	tool.Cancel(s)
	if tool.Name() != "Sketch Circle" || !tool.CanCommit() {
		t.Fatalf("sketch-circle tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	ss := c.Sheets().Active().Sketches()
	if ss.Count() != 1 || ss.Item(0).EntityCount() != 1 || ss.Item(0).CurveCount() == 0 {
		t.Fatalf("circle sketch not added (sketches=%d)", ss.Count())
	}
}

// TestHatchRegionToolFillsRegion: the hatch tool drops a sketch whose curves are the fill lines.
func TestHatchRegionToolFillsRegion(t *testing.T) {
	s := drawingWithModelSession(t)
	c, _ := ActiveDrawing(s)
	tool := NewHatchRegionTool()
	tool.Start(s)
	tool.SetPlacement(150, 150)
	tool.Params().Choices[0].Set(1) // Pattern = Cross
	if tool.Name() != "Hatch Region" || !tool.CanCommit() {
		t.Fatalf("hatch tool name/commit wrong: %q / %v", tool.Name(), tool.CanCommit())
	}
	if err := tool.Commit(s); err != nil {
		t.Fatalf("Commit: %v", err)
	}
	ss := c.Sheets().Active().Sketches()
	if ss.Count() != 1 || ss.Item(0).CurveCount() == 0 {
		t.Fatalf("hatch sketch not added (sketches=%d)", ss.Count())
	}
}
