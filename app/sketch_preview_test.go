// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati/math"
)

func TestLinePreviewFollowsCursor(t *testing.T) {
	s, _ := sketchSession(t)
	tool := NewLineTool()
	s.StartTool(tool)
	// No preview before the first click.
	if pts, _ := s.ActiveToolPreview(math.P2(5, 5)); len(pts) != 0 {
		t.Error("a line with no points should have no preview")
	}
	s.Click(100, 100) // one endpoint placed
	pts, closed := s.ActiveToolPreview(math.P2(3, 1))
	if len(pts) != 2 || closed {
		t.Errorf("line preview = %d pts closed=%v, want 2 open", len(pts), closed)
	}
	if !pts[1].IsEqualTo(math.P2(3, 1), 1e-9) {
		t.Errorf("line preview end = %v, want the cursor (3,1)", pts[1])
	}
}

func TestRectangleAndCirclePreview(t *testing.T) {
	s, _ := sketchSession(t)
	r := NewRectangleTool()
	s.StartTool(r)
	s.Click(100, 100) // first corner
	pts, closed := s.ActiveToolPreview(math.P2(4, 2))
	if len(pts) != 4 || !closed {
		t.Errorf("rectangle preview = %d pts closed=%v, want 4 closed", len(pts), closed)
	}

	c := NewCircleTool()
	s.StartTool(c)
	s.Click(100, 100) // centre
	pts, closed = s.ActiveToolPreview(math.P2(3, 0))
	if len(pts) < 8 || !closed {
		t.Errorf("circle preview = %d pts closed=%v, want a closed ring", len(pts), closed)
	}
}

func TestNoPreviewWithoutTool(t *testing.T) {
	s, _ := sketchSession(t)
	if pts, _ := s.ActiveToolPreview(math.P2(0, 0)); pts != nil {
		t.Error("no active tool should yield no preview")
	}
}

func TestCursorSketchPointSnaps(t *testing.T) {
	s, sk := sketchSession(t)
	sk.Points().Add(math.P2(2, 0))
	// The centre pixel maps to ~origin, which snaps to the sketch origin.
	if p, ok := s.CursorSketchPoint(100, 100); !ok || !p.IsEqualTo(math.P2(0, 0), 1e-9) {
		t.Errorf("cursor sketch point = %v (ok=%v), want the snapped origin", p, ok)
	}
}
