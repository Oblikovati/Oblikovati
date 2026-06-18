//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Box-select input + rubber-band rendering for the single viewport. A left-drag that
// starts on empty space sweeps a window/crossing selection box (app.Session owns the state
// machine + hit-test); a press on geometry falls through to the normal click select. The
// rectangle is drawn on top of the rendered viewport image (#916, follows the Inventor
// model chosen in the audit: left-drag is selection, never navigation).

// box-select rectangle colours (0..1 RGBA): a window select (drag left→right) is blue, a
// crossing select (drag right→left) is green — Inventor's convention. The fill is faint so
// the geometry stays visible through it.
var (
	boxWindowFill   = [4]float32{0.30, 0.55, 0.95, 0.18}
	boxWindowBorder = [4]float32{0.40, 0.65, 1.00, 0.90}
	boxCrossFill    = [4]float32{0.35, 0.80, 0.40, 0.18}
	boxCrossBorder  = [4]float32{0.45, 0.90, 0.50, 0.90}
)

// handleViewportSelection routes a left press/drag over the viewport: in the sketch editor a
// press on an unconstrained entity drags it; otherwise a drag begun on empty space runs
// box-select; anything else (a press on geometry, or no left input) falls through to the
// single-pick click handler. Replaces the bare handleViewportClick call so they never both
// fire on the same press.
func handleViewportSelection(s *app.Session) {
	if heldNavMode() != NavNone {
		return // a held F2/F3/F4 turns a left-drag into navigation, not selection (#911)
	}
	if updateZoomWindow(s) {
		return // the Zoom Window tool owns the left-drag while armed (#913 N16)
	}
	if s.SelectOtherActive() {
		if native.IsItemClicked(native.MouseLeft) {
			s.CommitSelectOther() // a click in the viewport accepts the highlighted candidate (#910)
		}
		return // the Select Other cycle owns viewport picking until it ends
	}
	if updateSketchDrag(s) {
		return
	}
	if updateBoxSelect(s) {
		return
	}
	handleViewportClick(s)
}

// updateSketchDrag advances direct drag-to-move of sketch entities and reports whether it
// consumed this frame's left input. A press on an unconstrained entity (no modifier, no active
// tool) begins the drag; the cursor moves it; release commits. A Shift/Ctrl press is a
// selection-extend, not a drag, so it falls through.
func updateSketchDrag(s *app.Session) bool {
	if !s.InSketch() {
		return false
	}
	if s.EntityDragActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateEntityDrag(lx, ly)
		} else {
			s.CommitEntityDrag()
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) || native.KeyShift() || native.KeyCtrl() {
		return false
	}
	lx, ly := viewportCursor()
	return s.BeginEntityDrag(lx, ly)
}

// updateBoxSelect advances the box-select state machine and reports whether it consumed this
// frame's left input. It begins a box only on a fresh left press over empty space, tracks the
// cursor while the button is held, and commits on release. In the sketch editor it selects 2D
// entities; in the model env it selects bodies (#909). updateSketchDrag runs first, so a press on
// a draggable sketch entity moves it rather than starting a box.
func updateBoxSelect(s *app.Session) bool {
	if s.BoxSelectActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateBoxSelect(lx, ly)
		} else {
			s.CommitBoxSelect(viewportSelectMods())
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	lx, ly := viewportCursor()
	if _, hit := s.PickAt(lx, ly, app.NewSelectionFilter()); hit {
		return false // pressed on geometry → normal click select (drag-to-move is future work)
	}
	s.BeginBoxSelect(lx, ly)
	return s.BoxSelectActive() // false when no RegionPicker is installed → fall through to click
}

// updateZoomWindow drives the Zoom Window rubber band while the tool is armed: a fresh left press
// anchors the box, the cursor sweeps it, and release zooms the view to fit it (#913 N16). It consumes
// all left input while armed so the drag never also selects. Esc disarms via cancelViewportTool.
func updateZoomWindow(s *app.Session) bool {
	if !s.ZoomWindowArmed() {
		return false
	}
	lx, ly := viewportCursor()
	if s.ZoomWindowDragging() {
		if native.MouseDown(native.MouseLeft) {
			s.UpdateZoomWindow(lx, ly)
		} else {
			s.CommitZoomWindow()
		}
		return true
	}
	if native.IsItemClicked(native.MouseLeft) {
		s.BeginZoomWindow(lx, ly)
	}
	return true
}

// drawZoomWindowRect draws the in-progress Zoom Window rubber band over the viewport image, in a
// neutral white (distinct from box-select's blue/green window/crossing colours). No-op when idle.
func drawZoomWindowRect(s *app.Session, bx, by float32) {
	if !s.ZoomWindowDragging() {
		return
	}
	x0, y0, x1, y1 := s.ZoomWindowRect()
	sx0, sy0 := bx+float32(x0), by+float32(y0)
	sx1, sy1 := bx+float32(x1), by+float32(y1)
	fill := [4]float32{0.85, 0.85, 0.90, 0.15}
	border := [4]float32{0.95, 0.95, 1.00, 0.95}
	native.DrawQuadFilled(sx0, sy0, sx1, sy0, sx1, sy1, sx0, sy1, fill)
	native.DrawLine(sx0, sy0, sx1, sy0, border, 1.2)
	native.DrawLine(sx1, sy0, sx1, sy1, border, 1.2)
	native.DrawLine(sx1, sy1, sx0, sy1, border, 1.2)
	native.DrawLine(sx0, sy1, sx0, sy0, border, 1.2)
}

// viewportSelectMods reads the Shift/Ctrl modifiers held this frame (Shift adds, Ctrl
// inverts the box's hits).
func viewportSelectMods() app.Modifier {
	var m app.Modifier
	if native.KeyShift() {
		m |= app.ShiftMod
	}
	if native.KeyCtrl() {
		m |= app.CtrlMod
	}
	return m
}

// drawBoxSelectRect draws the in-progress rubber-band rectangle over the viewport image.
// bx,by is the viewport image's top-left in screen pixels (the box stores viewport-local
// pixels). A no-op when no box-select drag is active.
func drawBoxSelectRect(s *app.Session, bx, by float32) {
	if !s.BoxSelectActive() {
		return
	}
	x0, y0, x1, y1, crossing := s.BoxSelectRect()
	sx0, sy0 := bx+float32(x0), by+float32(y0)
	sx1, sy1 := bx+float32(x1), by+float32(y1)
	fill, border := boxWindowFill, boxWindowBorder
	if crossing {
		fill, border = boxCrossFill, boxCrossBorder
	}
	native.DrawQuadFilled(sx0, sy0, sx1, sy0, sx1, sy1, sx0, sy1, fill)
	native.DrawLine(sx0, sy0, sx1, sy0, border, 1.2)
	native.DrawLine(sx1, sy0, sx1, sy1, border, 1.2)
	native.DrawLine(sx1, sy1, sx0, sy1, border, 1.2)
	native.DrawLine(sx0, sy1, sx0, sy0, border, 1.2)
}
