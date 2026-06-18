// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"testing"

	"oblikovati.org/model/drawing"
)

// dimensionedDrawingSession builds a boxed-part drawing with one placed base view + linear
// dimension — the shared fixture for the dimension-canvas-interaction tests.
func dimensionedDrawingSession(t *testing.T) (*Session, *drawing.DrawingDimension) {
	t.Helper()
	s := drawingWithModelSession(t)
	base := NewBaseViewTool()
	base.Start(s)
	base.SetPlacement(120, 100)
	if err := base.Commit(s); err != nil {
		t.Fatalf("place base view: %v", err)
	}
	lin := NewLinearDimensionTool()
	lin.Start(s)
	if err := lin.Commit(s); err != nil {
		t.Fatalf("place dimension: %v", err)
	}
	c, _ := ActiveDrawing(s)
	return s, c.Sheets().Active().Dimensions().Item(0)
}

// TestPickAndMoveDrawingDimension: a dimension's text and line can be picked on the canvas and
// moved, so overlapping dimensions are separated.
func TestPickAndMoveDrawingDimension(t *testing.T) {
	s, d := dimensionedDrawingSession(t)

	// Picking at the text anchor grabs the text.
	tx, ty := d.TextAnchorMM()
	name, grabText, ok := s.PickDrawingDimensionAt(tx, ty)
	if !ok || !grabText || name != d.Name() {
		t.Fatalf("pick at text = (%q,%v,%v), want (%q,true,true)", name, grabText, ok, d.Name())
	}
	s.MoveDimensionText(name, 6, 3)
	if nx, ny := d.TextAnchorMM(); nx != tx+6 || ny != ty+3 {
		t.Errorf("text moved to (%v,%v), want (%v,%v)", nx, ny, tx+6, ty+3)
	}

	// Picking on the dimension line away from the (centred) text grabs the line, not the text.
	line := d.Curves()[2]
	lx := float64(line.Start().X)*0.8 + float64(line.End().X)*0.2
	ly := float64(line.Start().Y)*0.8 + float64(line.End().Y)*0.2
	_, grabText, ok = s.PickDrawingDimensionAt(lx, ly)
	if !ok || grabText {
		t.Errorf("pick on line = (grabText %v, ok %v), want (false, true)", grabText, ok)
	}
}

// TestDragDimensionStateMachine: a press over the text starts a drag, subsequent active frames
// move the text, and releasing ends it.
func TestDragDimensionStateMachine(t *testing.T) {
	s, d := dimensionedDrawingSession(t)
	tx, ty := d.TextAnchorMM()

	// Press on the text (itemClicked) starts the drag and consumes the input.
	if !s.DragDimension(true, true, tx, ty, 0, 0) {
		t.Fatal("press on a dimension's text should start a drag")
	}
	if name, text, active := s.DraggingDimension(); !active || !text || name != d.Name() {
		t.Errorf("DraggingDimension = (%q,%v,%v), want the text of %q active", name, text, active, d.Name())
	}
	// Held with a move nudges the text.
	s.DragDimension(true, false, tx, ty, 4, -2)
	if nx, ny := d.TextAnchorMM(); nx != tx+4 || ny != ty-2 {
		t.Errorf("dragged text to (%v,%v), want (%v,%v)", nx, ny, tx+4, ty-2)
	}
	// Release ends the drag (no longer consumes input).
	if s.DragDimension(false, false, tx, ty, 0, 0) {
		t.Error("releasing should end the drag")
	}
}
