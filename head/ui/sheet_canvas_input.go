//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
	"oblikovati.org/math"
	"oblikovati.org/model/drawing"
)

// Interactive input on the drawing sheet canvas (M14-F02 PBI-139, #386): place a view by
// moving the mouse (live preview) and clicking; select a view by clicking; right-click a view
// for Edit/Delete. This is the drawing counterpart of the 3D viewport's pick/right-click,
// which the sheet-canvas branch otherwise bypasses.

const viewCtxPopup = "##drawing-view-ctx"

// canvasCtx holds the view a right-click opened the context menu on (popups persist across
// frames, so the handle must outlive the click).
var canvasCtx struct {
	handle app.DrawingViewHandle
	valid  bool
}

// handleSheetCanvasInput routes the canvas region's mouse to placement (when a placement tool
// is active) or to selection / context menu (otherwise). hovered/mx/my are the region's hover
// state and mouse position captured right after the canvas's invisible button.
func handleSheetCanvasInput(s *app.Session, face rect, hovered bool, mx, my float32) {
	cx, cy := screenToSheet(face, mx, my)
	if ti := s.ActiveTool(); ti != nil {
		if pt, ok := ti.Tool().(app.DrawingPlacementTool); ok {
			handlePlacement(s, pt, face, hovered, cx, cy)
			return
		}
	}
	if handleDimensionDrag(s, face, hovered, cx, cy) {
		return
	}
	handleSelection(s, hovered, cx, cy)
	drawViewCtxMenu(s)
}

// handleDimensionDrag forwards the canvas item's ImGui state and the per-frame mouse move to the
// session's dimension-drag state machine, returning true while a drag is in progress so view
// selection is suppressed. The drag logic itself lives in app (Session.DragDimension).
func handleDimensionDrag(s *app.Session, face rect, hovered bool, cx, cy float64) bool {
	ddx, ddy := native.MouseDelta()
	var dx, dy float64
	if face.scale != 0 {
		dx, dy = float64(ddx)/float64(face.scale), -float64(ddy)/float64(face.scale)
	}
	clicked := hovered && native.IsItemClicked(native.MouseLeft)
	return s.DragDimension(native.IsItemActive(), clicked, cx, cy, dx, dy)
}

// handlePlacement tracks the cursor as the pending view centre, draws the preview there, and
// commits the tool (dropping the view) on a left-click in the canvas.
func handlePlacement(s *app.Session, pt app.DrawingPlacementTool, face rect, hovered bool, cx, cy float64) {
	if hovered {
		pt.SetPlacement(cx, cy)
	}
	drawPreviewAt(face, pt.PreviewCurves(s), cx, cy)
	if hovered && native.IsItemClicked(native.MouseLeft) {
		_ = s.OK() // commit the tool's view at the tracked placement
	}
}

// handleSelection selects the view under a left-click (clearing on empty), and opens the
// context menu on a right-click over a view.
func handleSelection(s *app.Session, hovered bool, cx, cy float64) {
	if !hovered {
		return
	}
	if native.IsItemClicked(native.MouseLeft) {
		if h, ok := s.PickDrawingViewAt(cx, cy); ok {
			s.SelectDrawingViewHandle(h)
		}
	}
	if native.IsItemClicked(native.MouseRight) {
		if h, ok := s.PickDrawingViewAt(cx, cy); ok {
			canvasCtx.handle, canvasCtx.valid = h, true
			native.OpenPopup(viewCtxPopup)
		}
	}
}

// drawViewCtxMenu renders the right-clicked view's Edit/Delete menu (the same items the
// browser shows), running each item's action on click.
func drawViewCtxMenu(s *app.Session) {
	if !canvasCtx.valid {
		return
	}
	if !native.BeginPopup(viewCtxPopup) {
		return
	}
	node := app.BrowserNode{Kind: "drawingView", Select: canvasCtx.handle}
	for _, item := range app.BrowserMenu(node) {
		if native.MenuItem(item.Label) && item.Invoke != nil {
			_ = item.Invoke(s)
		}
	}
	native.EndPopup()
}

// drawPreviewAt draws a view's origin-centred preview curves offset to the cursor (sheet mm),
// so the ghost view follows the mouse before it is dropped.
func drawPreviewAt(face rect, curves []drawing.DrawingCurve, cx, cy float64) {
	for _, c := range curves {
		ax, ay := curveToScreen(face, offset2(c.Start(), cx, cy))
		bx, by := curveToScreen(face, offset2(c.End(), cx, cy))
		native.DrawLine(ax, ay, bx, by, viewPreviewInk, 1.2)
	}
}

var viewPreviewInk = [4]float32{0.20, 0.45, 0.85, 0.9} // a blue ghost while placing

// offset2 shifts a sheet point by the cursor placement (millimetres).
func offset2(p math.Point2, cx, cy float64) math.Point2 {
	return math.P2(p.X+math.Scalar(cx), p.Y+math.Scalar(cy))
}

// screenToSheet converts a screen pixel to sheet millimetres (inverse of curveToScreen).
func screenToSheet(face rect, mx, my float32) (float64, float64) {
	if face.scale == 0 {
		return 0, 0
	}
	return float64((mx - face.x) / face.scale), float64((face.y + face.h - my) / face.scale)
}
