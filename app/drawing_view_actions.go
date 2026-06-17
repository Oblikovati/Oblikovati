// SPDX-License-Identifier: GPL-2.0-only

package app

import "oblikovati.org/model/drawing"

// Drawing view selection + browser/canvas actions (M14-F02 PBI-139, #386). A drawing view is
// a first-class selectable object: it appears in the browser, can be selected on the sheet
// canvas, and supports Edit (re-open its settings) and Delete from a right-click menu.

// DrawingViewHandle wraps a drawing view for selection. It carries the owning collection so
// Delete/Edit can act, and the view itself for identity and field reads.
type DrawingViewHandle struct {
	Views *drawing.DrawingViews
	View  *drawing.DrawingView
}

// SelectionKind reports a drawing-view selection.
func (DrawingViewHandle) SelectionKind() SelectionKind { return SelectDrawingView }

// DeleteDrawingView removes the view (cascading to views projected from it) and re-renders.
func (s *Session) DeleteDrawingView(h DrawingViewHandle) error {
	if h.Views == nil || h.View == nil {
		return nil
	}
	if err := h.Views.Remove(h.View.Name()); err != nil {
		return err
	}
	s.selection.Clear()
	if d := s.ActiveDocument(); d != nil {
		d.MarkDirty()
	}
	return nil
}

// BeginEditDrawingView opens the view's settings in a dialog bound to the existing view, so OK
// re-projects it in place (mirrors BeginEditFeature for part features).
func (s *Session) BeginEditDrawingView(h DrawingViewHandle) {
	if h.Views == nil || h.View == nil {
		return
	}
	s.StartTool(newDrawingViewEditTool(h.Views, h.View))
}

// PickDrawingViewAt returns the active-sheet view whose bounds contain the sheet point
// (millimetres), topmost first, and false if none — the canvas click/right-click picker.
func (s *Session) PickDrawingViewAt(xMM, yMM float64) (DrawingViewHandle, bool) {
	c, err := ActiveDrawing(s)
	if err != nil {
		return DrawingViewHandle{}, false
	}
	views := c.Sheets().Active().Views()
	for i := views.Count() - 1; i >= 0; i-- {
		v := views.Item(i)
		if minX, minY, maxX, maxY, ok := v.BoundsMM(); ok && xMM >= minX && xMM <= maxX && yMM >= minY && yMM <= maxY {
			return DrawingViewHandle{Views: views, View: v}, true
		}
	}
	return DrawingViewHandle{}, false
}

// SelectDrawingViewHandle puts a drawing-view handle in the selection set (canvas/browser click).
func (s *Session) SelectDrawingViewHandle(h DrawingViewHandle) { s.Select(h) }
