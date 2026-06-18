// SPDX-License-Identifier: GPL-2.0-only

package app

import (
	"math"

	"oblikovati.org/model/drawing"
)

// Drawing-dimension canvas interaction (M14-F03): a dimension's value text and its dimension line
// can be grabbed and dragged on the sheet, so overlapping dimensions are separated for readability.

// dimDragState tracks an in-progress drag of a dimension's text or line across frames.
type dimDragState struct {
	name   string
	text   bool
	active bool
}

// DragDimension drives a dimension drag on the canvas from the canvas item's ImGui state
// (itemActive/itemClicked), the cursor in sheet mm (cx,cy) and the per-frame move in sheet mm
// (dx,dy). It returns true while a drag is in progress, so the head suppresses view selection.
func (s *Session) DragDimension(itemActive, itemClicked bool, cx, cy, dx, dy float64) bool {
	if s.dimDrag.active {
		if !itemActive {
			s.dimDrag.active = false
			return false
		}
		if dx != 0 || dy != 0 {
			if s.dimDrag.text {
				s.MoveDimensionText(s.dimDrag.name, dx, dy)
			} else {
				s.MoveDimensionLine(s.dimDrag.name, dx, dy)
			}
		}
		return true
	}
	if itemClicked {
		if name, text, ok := s.PickDrawingDimensionAt(cx, cy); ok {
			s.dimDrag.name, s.dimDrag.text, s.dimDrag.active = name, text, true
			return true
		}
	}
	return false
}

// PickDrawingDimensionAt finds the topmost active-sheet dimension near the sheet point (mm) and
// reports whether the pick landed on its value text (grabText) rather than its line.
func (s *Session) PickDrawingDimensionAt(xMM, yMM float64) (name string, grabText, ok bool) {
	c, err := ActiveDrawing(s)
	if err != nil {
		return "", false, false
	}
	const textR, lineR = 7.0, 3.0 // mm pick tolerances
	ds := c.Sheets().Active().Dimensions()
	for i := ds.Count() - 1; i >= 0; i-- {
		d := ds.Item(i)
		tx, ty := d.TextAnchorMM()
		if math.Hypot(xMM-tx, yMM-ty) <= textR {
			return d.Name(), true, true
		}
		if nearDimensionLine(d, xMM, yMM, lineR) {
			return d.Name(), false, true
		}
	}
	return "", false, false
}

// nearDimensionLine reports whether (x, y) lies within tol of any of the dimension's curves.
func nearDimensionLine(d *drawing.DrawingDimension, x, y, tol float64) bool {
	for _, c := range d.Curves() {
		if pointSegmentDistanceMM(x, y, float64(c.Start().X), float64(c.Start().Y), float64(c.End().X), float64(c.End().Y)) <= tol {
			return true
		}
	}
	return false
}

// MoveDimensionText nudges a dimension's value text by (dx, dy) sheet mm (drag-the-text).
func (s *Session) MoveDimensionText(name string, dx, dy float64) {
	if c, err := ActiveDrawing(s); err == nil {
		c.Sheets().Active().Dimensions().MoveText(name, dx, dy)
		s.markActiveDirty()
	}
}

// MoveDimensionLine shifts a dimension's line by (dx, dy) sheet mm (drag-the-line).
func (s *Session) MoveDimensionLine(name string, dx, dy float64) {
	if c, err := ActiveDrawing(s); err == nil {
		c.Sheets().Active().Dimensions().MoveLine(name, dx, dy)
		s.markActiveDirty()
	}
}

// markActiveDirty marks the active document dirty after a canvas edit.
func (s *Session) markActiveDirty() {
	if d := s.ActiveDocument(); d != nil {
		d.MarkDirty()
	}
}

// pointSegmentDistanceMM is the distance from (px, py) to the segment (ax, ay)-(bx, by).
func pointSegmentDistanceMM(px, py, ax, ay, bx, by float64) float64 {
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	if l2 < 1e-12 {
		return math.Hypot(px-ax, py-ay)
	}
	t := math.Max(0, math.Min(1, ((px-ax)*dx+(py-ay)*dy)/l2))
	return math.Hypot(px-(ax+t*dx), py-(ay+t*dy))
}
