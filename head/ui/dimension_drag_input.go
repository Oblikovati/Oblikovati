//go:build cgo

// SPDX-License-Identifier: GPL-2.0-only

package ui

import (
	"oblikovati.org/app"
	"oblikovati.org/head/internal/native"
)

// Dragging a sketch dimension's value label to reposition it (#2017). A sketch dimension used to
// be inert: only a double-click on its label did anything (open the value editor), so it could not
// be selected, deleted or moved — unlike a drawing dimension, draggable on the sheet since M14-F03.

// dimensionDragSession is the session surface the dimension-label drag drives (audit I5, the
// arrowSession pattern) — the state machine and nothing else.
type dimensionDragSession interface {
	DimensionDragActive() bool
	BeginDimensionDrag(px, py float64, mods app.Modifier) bool
	UpdateDimensionDrag(px, py float64)
	CommitDimensionDrag()
}

var _ dimensionDragSession = (*app.Session)(nil)

// updateDimensionDrag advances a drag of a sketch dimension's value label and reports whether it
// consumed this frame's left input. It runs BEFORE the entity drag so a label lying over the
// geometry it measures grabs the label rather than the curve underneath — the label is the
// smaller, deliberately-aimed-at target. Unlike an entity drag it honours Shift/Ctrl, because the
// press also selects and a modified click is an extend-selection.
func updateDimensionDrag(s dimensionDragSession) bool {
	if s.DimensionDragActive() {
		lx, ly := viewportCursor()
		if native.MouseDown(native.MouseLeft) {
			s.UpdateDimensionDrag(lx, ly)
		} else {
			s.CommitDimensionDrag()
		}
		return true
	}
	if !native.IsItemClicked(native.MouseLeft) {
		return false
	}
	lx, ly := viewportCursor()
	return s.BeginDimensionDrag(lx, ly, sketchPickModifier())
}

// sketchPickModifier maps the held modifier keys to the selection modifier, so Shift/Ctrl-clicking
// a dimension extends the selection instead of replacing it.
func sketchPickModifier() app.Modifier {
	if native.KeyShift() {
		return app.ShiftMod
	}
	if native.KeyCtrl() {
		return app.CtrlMod
	}
	return 0
}
