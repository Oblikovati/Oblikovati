// SPDX-License-Identifier: GPL-2.0-only

package app

import "testing"

// A press and release without movement is a click: it places one point and waits, exactly as
// the pre-#2014 click-click flow did.
func TestPlacementClickPlacesOnePoint(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	tool := NewRectangleTool()
	s.StartTool(tool)
	if !s.BeginPlacement(100, 100) {
		t.Fatal("a geometry tool must claim the press")
	}
	s.EndPlacement(100, 100)
	if len(tool.corners) != 1 {
		t.Fatalf("corners = %d, want 1", len(tool.corners))
	}
	if n := len(sk.Entities()); n != 0 {
		t.Errorf("entities = %d, want 0 — a click must not commit", n)
	}
}

// A press, drag past the slop, and release places both points and commits the shape. This is
// the behaviour the issue reported missing: creation ran off the mouse PRESS with no release
// handler, so a press-drag-release placed one point and the shape appeared on a later,
// unrelated press.
func TestPlacementDragPlacesTwoPointsAndCommits(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	s.StartTool(NewRectangleTool())
	s.BeginPlacement(100, 100)
	s.UpdatePlacement(160, 150)
	s.EndPlacement(160, 150)
	if n := len(sk.Entities()); n == 0 {
		t.Fatal("a drag-release must commit the rectangle")
	}
	a := sk.AnalyzeConstraints()
	if a.DOF != 4 || a.Redundant != 0 {
		t.Errorf("DOF = %d redundant = %d, want 4 and 0", a.DOF, a.Redundant)
	}
}

// A drag shorter than the slop is a click, not a drag — a shaky hand must not create geometry.
func TestPlacementSlopBoundary(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	tool := NewRectangleTool()
	s.StartTool(tool)
	s.BeginPlacement(100, 100)
	s.UpdatePlacement(102, 101) // 2.24 px, under the 4 px slop
	s.EndPlacement(102, 101)
	if len(tool.corners) != 1 {
		t.Errorf("corners = %d, want 1 — a sub-slop drag is a click", len(tool.corners))
	}
}

// Click-click and drag-release must produce the same geometry; they are the same path, differing
// only in when the second point arrives.
func TestPlacementClickClickEqualsDrag(t *testing.T) {
	t.Parallel()
	dragged, _ := sketchSession(t)
	dragged.StartTool(NewRectangleTool())
	dragged.BeginPlacement(100, 100)
	dragged.UpdatePlacement(160, 150)
	dragged.EndPlacement(160, 150)

	clicked, _ := sketchSession(t)
	clicked.StartTool(NewRectangleTool())
	clicked.BeginPlacement(100, 100)
	clicked.EndPlacement(100, 100)
	clicked.BeginPlacement(160, 150)
	clicked.EndPlacement(160, 150)

	da := dragged.ActiveSketch().AnalyzeConstraints()
	ca := clicked.ActiveSketch().AnalyzeConstraints()
	if da.Variables != ca.Variables || da.DOF != ca.DOF {
		t.Errorf("drag gave vars=%d DOF=%d, click-click gave vars=%d DOF=%d — they must agree",
			da.Variables, da.DOF, ca.Variables, ca.DOF)
	}
}

// A three-point tool takes the drag for its first two points and stays click-click for the rest,
// which falls out of the state machine with no special-casing.
func TestPlacementDragThenClickForThreePointTool(t *testing.T) {
	t.Parallel()
	s, sk := sketchSession(t)
	tool := NewThreePointRectangleTool()
	s.StartTool(tool)
	s.BeginPlacement(80, 100)
	s.UpdatePlacement(160, 100)
	s.EndPlacement(160, 100)
	if len(tool.pts) != 2 {
		t.Fatalf("after the drag, points = %d, want 2", len(tool.pts))
	}
	if n := len(sk.Entities()); n != 0 {
		t.Fatalf("entities = %d, want 0 — a three-point shape needs its third point", n)
	}
	s.BeginPlacement(160, 60)
	s.EndPlacement(160, 60)
	if n := len(sk.Entities()); n == 0 {
		t.Error("the third click must commit the rectangle")
	}
}

// With no tool active the press is not claimed, so selection and box-select still see it.
func TestPlacementIgnoredWithoutTool(t *testing.T) {
	t.Parallel()
	s, _ := sketchSession(t)
	if s.BeginPlacement(100, 100) {
		t.Error("no active tool must leave the press to the selection path")
	}
	if s.PlacementActive() {
		t.Error("no placement should be armed")
	}
}
